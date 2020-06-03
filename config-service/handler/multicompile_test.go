package handler

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/hailo-platform/H2O/config-service/domain"
	platformtesting "github.com/hailo-platform/H2O/platform/testing"
	"github.com/hailo-platform/H2O/service/config"
	"github.com/hailo-platform/H2O/service/nsq"
	ssync "github.com/hailo-platform/H2O/service/sync"
	zk "github.com/hailo-platform/H2O/service/zookeeper"
	gozk "github.com/hailo-platform/H2O/go-zookeeper/zk"
	"github.com/hailo-platform/H2O/protobuf/proto"

	multicompile "github.com/hailo-platform/H2O/config-service/proto/multicompile"
	uproto "github.com/hailo-platform/H2O/config-service/proto/update"
)

type MulticompileSuite struct {
	platformtesting.Suite
	zk            *zk.MockZookeeperClient
	realPublisher nsq.Publisher
	nsq           *nsq.MockPublisher
}

func TestRunMulticompileSuite(t *testing.T) {
	platformtesting.RunSuite(t, new(MulticompileSuite))
}

func (s *MulticompileSuite) SetupTest() {
	s.Suite.SetupTest()

	// Mock ZK
	s.zk = &zk.MockZookeeperClient{}
	zk.ActiveMockZookeeperClient = s.zk
	zk.Connector = zk.MockConnector
	ssync.SetRegionLockNamespace("com.hailo-platform/H2O.service.config")

	// Mock NSQ
	s.realPublisher = nsq.DefaultPublisher
	s.nsq = &nsq.MockPublisher{}
	nsq.DefaultPublisher = s.nsq
}

func (s *MulticompileSuite) TearDownTest() {
	s.Suite.TearDownTest()
	s.zk.On("Close").Return().Once()
	zk.ActiveMockZookeeperClient = nil
	zk.Connector = zk.DefaultConnector
	zk.TearDown()
	nsq.DefaultPublisher = s.realPublisher
}

func (s *MulticompileSuite) TestMulticompileHandlerAuth() {
	// load nothing so config loader does not complain
	buf := bytes.NewBufferString(`{}`)
	config.Load(buf)

	ids := []string{"H2:REGION:eu-west-1:test_multiconfig1", "H2:REGION:eu-west-1:test_multiconfig2"}
	data := make(map[string]*domain.ChangeSet)
	for _, id := range ids {
		data[id] = &domain.ChangeSet{
			Id:        id,
			Body:      []byte(`{}`),
			Timestamp: time.Now(),
		}
	}

	domain.DefaultRepository = domain.NewMemoryRepository(data)

	testCases := []struct {
		uid    string
		config string
	}{
		{
			"test_id1",
			"\"test_value1\"",
		},
		{
			"test_id2",
			"\"test_value2\"",
		},
	}

	path := "path"
	for i, test := range testCases {
		serverReq := newTestRequest(&uproto.Request{
			Id:      proto.String(ids[i]),
			Path:    proto.String(path),
			Message: proto.String("Update test:" + test.uid),
			Config:  proto.String(test.config),
		})

		lock := &zk.MockLock{}
		lock.On("Lock").Return(nil)
		lock.On("Unlock").Return(nil)
		lock.On("SetTTL", mock.AnythingOfType("time.Duration")).Return()
		lock.On("SetTimeout", mock.AnythingOfType("time.Duration")).Return()

		lockPath := fmt.Sprintf("/com.hailo-platform/H2O.service.config/%s", ids[i])
		s.zk.
			On("NewLock", lockPath, gozk.WorldACL(gozk.PermAll)).
			Return(lock)
		s.zk.On("Exists", lockPath).Return(false, &gozk.Stat{}, nil)
		s.zk.On("Delete", lockPath, int32(-1)).Return(nil)

		s.nsq.On("Publish", broadcastTopic, mock.Anything).Return(nil)
		s.nsq.On("Publish", platformTopicName, mock.Anything).Return(nil)

		_, err := Update(serverReq)
		s.NoError(err)
	}

	serverReq := newTestRequest(&multicompile.Request{
		CompileRequests: []*multicompile.Request_CompileRequest{
			&multicompile.Request_CompileRequest{
				Id:   []string{ids[0]},
				Path: proto.String(path)},
			&multicompile.Request_CompileRequest{
				Id:   []string{ids[1]},
				Path: proto.String(path)},
		},
	})
	rsp, err := MultiCompile(serverReq)
	s.NoError(err)

	s.Equal(2, len(rsp.(*multicompile.Response).GetCompileResponses()),
		"Expected 2 compile responses: %v", len(rsp.(*multicompile.Response).GetCompileResponses()))

	for i, compileresponse := range rsp.(*multicompile.Response).GetCompileResponses() {
		s.Equal(compileresponse.GetConfig(), testCases[i].config,
			"Expected response:%v\n but received:%v", testCases[i].config, compileresponse.GetConfig())
	}
}
