package handler

import (
	"fmt"
	"testing"
	"time"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/mock"

	"github.com/hailo-platform/H2O/config-service/domain"
	"github.com/hailo-platform/H2O/platform/server"
	platformtesting "github.com/hailo-platform/H2O/platform/testing"
	"github.com/hailo-platform/H2O/service/auth"
	"github.com/hailo-platform/H2O/service/nsq"
	ssync "github.com/hailo-platform/H2O/service/sync"
	zk "github.com/hailo-platform/H2O/service/zookeeper"
	gozk "github.com/hailo-platform/H2O/go-zookeeper/zk"
	"github.com/hailo-platform/H2O/protobuf/proto"

	rproto "github.com/hailo-platform/H2O/config-service/proto/read"
	uproto "github.com/hailo-platform/H2O/config-service/proto/update"
)

type UpdateSuite struct {
	platformtesting.Suite
	zk            *zk.MockZookeeperClient
	realPublisher nsq.Publisher
	nsq           *nsq.MockPublisher
}

func TestRunUpdateSuite(t *testing.T) {
	platformtesting.RunSuite(t, new(UpdateSuite))
}

func (s *UpdateSuite) SetupTest() {
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

func (s *UpdateSuite) TearDownTest() {
	s.Suite.TearDownTest()
	s.zk.On("Close").Return().Once()
	zk.ActiveMockZookeeperClient = nil
	zk.Connector = zk.DefaultConnector
	zk.TearDown()
	nsq.DefaultPublisher = s.realPublisher
}

func newTestRequest(p proto.Message) *server.Request {
	protoBytes, _ := proto.Marshal(p)

	d := amqp.Delivery{
		Body:        protoBytes,
		ContentType: "application/octetstream",
		Headers: amqp.Table{
			"from": "test",
		},
	}

	return server.NewRequestFromDelivery(d)
}

func (s *UpdateSuite) TestUpdateHandlerAuth() {
	id := "H2:REGION:eu-west-1"
	path := "hailo/service/memcache/hosts"
	data := map[string]*domain.ChangeSet{
		id: &domain.ChangeSet{
			Id:        id,
			Body:      []byte(`{}`),
			Timestamp: time.Now(),
		},
	}

	domain.DefaultRepository = domain.NewMemoryRepository(data)

	testCases := []struct {
		uid  string
		mech string
		auth bool
	}{
		{
			"mockUid",
			"mock",
			true,
		},
		{
			"test",
			defaultMech,
			false,
		},
	}

	for _, test := range testCases {
		serverReq := newTestRequest(&uproto.Request{
			Id:      proto.String(id),
			Path:    proto.String(path),
			Message: proto.String("Update test"),
			Config:  proto.String("[\"10.0.0.1\", \"10.0.0.2\", \"10.0.0.3\"]"),
		})

		if test.auth {
			serverReq.SetAuth(&auth.MockScope{MockUid: test.uid, MockRoles: []string{"ADMIN"}})
		}

		lock := &zk.MockLock{}
		lock.On("Lock").Return(nil)
		lock.On("Unlock").Return(nil)
		lock.On("SetTTL", mock.AnythingOfType("time.Duration")).Return()
		lock.On("SetTimeout", mock.AnythingOfType("time.Duration")).Return()

		lockPath := fmt.Sprintf("/com.hailo-platform/H2O.service.config/%s", id)
		s.zk.
			On("NewLock", lockPath, gozk.WorldACL(gozk.PermAll)).
			Return(lock)
		s.zk.On("Exists", lockPath).Return(false, &gozk.Stat{}, nil)
		s.zk.On("Delete", lockPath, int32(-1)).Return(nil)

		s.nsq.On("Publish", broadcastTopic, mock.Anything).Return(nil)
		s.nsq.On("Publish", platformTopicName, mock.Anything).Return(nil)

		_, err := Update(serverReq)
		s.NoError(err)

		serverReq = newTestRequest(&rproto.Request{
			Id:   proto.String(id),
			Path: proto.String(path),
		})

		rsp, err := Read(serverReq)
		s.NoError(err)

		meta := rsp.(*rproto.Response).GetMeta()
		s.Equal(test.mech, meta.GetAuthMechanism())
		s.Equal(test.uid, meta.GetUserId())
	}
}
