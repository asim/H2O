package statusmonitor

import (
	"fmt"
	"github.com/hailo-platform/H2O/platform/multiclient"
	platformtesting "github.com/hailo-platform/H2O/platform/testing"
	zk "github.com/hailo-platform/H2O/service/zookeeper"
	gozk "github.com/hailo-platform/H2O/go-zookeeper/zk"
	"testing"
	"time"
)

type SMTestSuite struct {
	platformtesting.Suite
	zkRunner *zk.MockZookeeperClient
	monitor  *StatusMonitor
	client   *multiclient.Mock
}

func TestRunSMTestSuite(t *testing.T) {
	platformtesting.RunSuite(t, new(SMTestSuite))
}

func (s *SMTestSuite) SetupTest() {
	s.Suite.SetupTest()

	// Setup ZK
	s.zkRunner = &zk.MockZookeeperClient{}
	zk.ActiveMockZookeeperClient = s.zkRunner
	zk.Connector = zk.MockConnector

	// Init our monitor
	s.monitor = &StatusMonitor{
		IsHealthy:   true,
		LastChanged: time.Now(),
		AZName:      "test",
	}

	// Setup the platform client
	s.client = multiclient.NewMock()
	multiclient.SetCaller(s.client.Caller())
}

func (s *SMTestSuite) TearDownTest() {
	s.Suite.TearDownTest()
	zk.ActiveMockZookeeperClient = nil
	zk.Connector = zk.DefaultConnector
	s.zkRunner.AssertExpectations(s.T())
	s.zkRunner.On("Close").Return().Once()
	zk.TearDown()
	multiclient.SetCaller(multiclient.PlatformCaller())
}

func (s *SMTestSuite) TestAZFailoverAndRecover() {
	s.monitor.IsHealthy = true
	// If online and rabbit is not connected we should failover
	s.zkRunner.
		On("Create", "/hailo-2-api-az-failover", []byte("test"), int32(gozk.FlagEphemeral), gozk.WorldACL(gozk.PermAll)).
		Return("/hailo-2-api-az-failover", nil).
		Once()

	azStatus := false
	s.monitor.interpretAZStatus(azStatus)
	s.Equal(s.monitor.IsHealthy, false)
	s.Equal(s.monitor.FailureType, MonitoringFailure)

	// Make sure this persists and we correctly detect that we have failed over
	s.monitor.interpretAZStatus(azStatus)
	s.Equal(s.monitor.IsHealthy, false)
	s.Equal(s.monitor.FailureType, MonitoringFailure)

	// Recover if we are holding the lock
	azStatus = true
	s.zkRunner.
		On("Delete", "/hailo-2-api-az-failover", int32(0)).
		Return(nil).
		Once()
	s.monitor.interpretAZStatus(azStatus)
	s.Equal(s.monitor.FailureType, NoFailure)
	s.Equal(s.monitor.IsHealthy, true)

}

// TestOnlyOnceAZFailover tests if another AZ has been failed over, we should abort
// @TODO: this should be gone once we have more confidence in the failover
func (s *SMTestSuite) TestOnlyOnceAZFailover() {
	s.monitor.IsHealthy = true
	s.zkRunner.
		On("Create", "/hailo-2-api-az-failover", []byte("test"), int32(gozk.FlagEphemeral), gozk.WorldACL(gozk.PermAll)).
		Return("", fmt.Errorf("Already exists")).
		Once()
	s.zkRunner.
		On("Get", "/hailo-2-api-az-failover").
		Return([]byte("test2"), new(gozk.Stat), nil).
		Once()

	// Currently we expect to abort the AZ failover in case we can't process it
	azStatus := false
	s.monitor.interpretAZStatus(azStatus)
	s.Equal(s.monitor.FailureType, NoFailure)
	s.Equal(s.monitor.IsHealthy, true)
}

// TestSecondaryInstanceSameAZFailover tests if we can failover a second API instance in the same AZ
// if the AZ has been marked as failed
func (s *SMTestSuite) TestSecondaryInstanceSameAZFailover() {
	s.monitor.IsHealthy = true
	s.zkRunner.
		On("Create", "/hailo-2-api-az-failover", []byte("test"), int32(gozk.FlagEphemeral), gozk.WorldACL(gozk.PermAll)).
		Return("", fmt.Errorf("Already exists")).
		Once()
	s.zkRunner.
		On("Get", "/hailo-2-api-az-failover").
		Return([]byte("test"), new(gozk.Stat), nil).
		Once()

	azStatus := false
	s.monitor.interpretAZStatus(azStatus)
	s.Equal(s.monitor.IsHealthy, false)
	s.Equal(s.monitor.FailureType, MonitoringFailure)

	// Recover
	azStatus = true
	s.monitor.interpretAZStatus(azStatus)
	s.Equal(s.monitor.IsHealthy, true)
	s.Equal(s.monitor.FailureType, NoFailure)

}

// TestConnectivityFailover tests failures caused by rabbitmq
func (s *SMTestSuite) TestConnectivityFailover() {
	s.monitor.IsHealthy = true
	s.zkRunner.
		On("Create", "/hailo-2-api-az-failover", []byte("test"), int32(gozk.FlagEphemeral), gozk.WorldACL(gozk.PermAll)).
		Return("/hailo-2-api-az-failover", nil).
		Once()

	isConnected := false
	s.monitor.interpretRavenStatus(isConnected)
	s.Equal(s.monitor.IsHealthy, false)
	s.Equal(s.monitor.FailureType, ConnectivityFailure)

	// Recover
	s.zkRunner.
		On("Delete", "/hailo-2-api-az-failover", int32(0)).
		Return(nil).
		Once()
	isConnected = true
	s.monitor.interpretRavenStatus(isConnected)
	s.Equal(s.monitor.IsHealthy, true)
	s.Equal(s.monitor.FailureType, NoFailure)
}

// TestComplexFailover tests the failover behaviour with multiple failure sources
func (s *SMTestSuite) TestComplexFailover() {
	s.monitor.IsHealthy = true
	s.zkRunner.
		On("Create", "/hailo-2-api-az-failover", []byte("test"), int32(gozk.FlagEphemeral), gozk.WorldACL(gozk.PermAll)).
		Return("/hailo-2-api-az-failover", nil).
		Once()

	isConnected := false
	s.monitor.interpretRavenStatus(isConnected)
	s.Equal(s.monitor.IsHealthy, false)
	s.Equal(s.monitor.FailureType, ConnectivityFailure)

	// Recover Connectivity but report AZ failure
	s.zkRunner.
		On("Delete", "/hailo-2-api-az-failover", int32(0)).
		Return(nil).
		Once()
	isConnected = true
	s.monitor.interpretRavenStatus(isConnected)

	s.zkRunner.
		On("Create", "/hailo-2-api-az-failover", []byte("test"), int32(gozk.FlagEphemeral), gozk.WorldACL(gozk.PermAll)).
		Return("/hailo-2-api-az-failover", nil).
		Once()
	azStatus := false
	s.monitor.interpretAZStatus(azStatus)
	s.Equal(s.monitor.IsHealthy, false)
	s.Equal(s.monitor.FailureType, MonitoringFailure)
}
