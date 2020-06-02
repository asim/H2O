package zookeeper

import (
	"time"

	log "github.com/cihub/seelog"

	gozk "github.com/HailoOSS/go-zookeeper/zk"
	"github.com/stretchr/testify/mock"
)

// Implements the ZookeeperClient interface
type MockZookeeperClient struct {
	mock.Mock
}

// This must be set to an instance of MockZookeeperClient before tests begin. MockConnector will always return this
// pointer
var ActiveMockZookeeperClient *MockZookeeperClient = nil

// Connector which returns a MockZooKeeperClient. To use a ZK mock during testing, replace zookeeper.Connector with this
// function (and be sure to set it back to zookeeper.DefaultConnector when the test exits [via a deferred call])
func MockConnector(servers []string, recvTimeout time.Duration) (ZookeeperClient, <-chan gozk.Event, error) {
	return ActiveMockZookeeperClient, nil, nil
}

func (c *MockZookeeperClient) Children(path string) ([]string, *gozk.Stat, error) {
	log.Tracef("[ZooKeeper mock] Children(path=%s) called", path)
	returnArgs := c.Mock.Called(path)
	return returnArgs.Get(0).([]string),
		returnArgs.Get(1).(*gozk.Stat),
		returnArgs.Error(2)
}

func (c *MockZookeeperClient) ChildrenW(path string) ([]string, *gozk.Stat, <-chan gozk.Event, error) {
	log.Tracef("[ZooKeeper mock] ChildrenW(path=%s) called", path)
	returnArgs := c.Mock.Called(path)
	return returnArgs.Get(0).([]string),
		returnArgs.Get(1).(*gozk.Stat),
		returnArgs.Get(2).(<-chan gozk.Event),
		returnArgs.Error(3)
}

func (c *MockZookeeperClient) Close() {
	log.Trace("[ZooKeeper mock] Close() called")
	c.Mock.Called()
}

func (c *MockZookeeperClient) Create(path string, data []byte, flags int32, acl []gozk.ACL) (string, error) {
	log.Tracef("[ZooKeeper mock] Create(path=%s) called", path)
	returnArgs := c.Mock.Called(path, data, flags, acl)
	return returnArgs.String(0),
		returnArgs.Error(1)
}

func (c *MockZookeeperClient) CreateProtectedEphemeralSequential(path string, data []byte, acl []gozk.ACL) (string, error) {
	log.Tracef("[ZooKeeper mock] CreateProtectedEphemeralSequential(path=%s) called", path)
	returnArgs := c.Mock.Called(path, data, acl)
	return returnArgs.String(0),
		returnArgs.Error(1)
}

func (c *MockZookeeperClient) Delete(path string, version int32) error {
	log.Tracef("[ZooKeeper mock] Delete(path=%s) called", path)
	return c.Mock.Called(path, version).Error(0)
}

func (c *MockZookeeperClient) Exists(path string) (bool, *gozk.Stat, error) {
	log.Tracef("[ZooKeeper mock] Exists(path=%s) called", path)
	returnArgs := c.Mock.Called(path)
	return returnArgs.Bool(0),
		returnArgs.Get(1).(*gozk.Stat),
		returnArgs.Error(2)
}

func (c *MockZookeeperClient) ExistsW(path string) (bool, *gozk.Stat, <-chan gozk.Event, error) {
	log.Tracef("[ZooKeeper mock] ExistsW(path=%s) called", path)
	returnArgs := c.Mock.Called(ExistsW)
	return returnArgs.Bool(0),
		returnArgs.Get(1).(*gozk.Stat),
		returnArgs.Get(2).(<-chan gozk.Event),
		returnArgs.Error(3)
}

func (c *MockZookeeperClient) Get(path string) ([]byte, *gozk.Stat, error) {
	log.Tracef("[ZooKeeper mock] Get(path=%s) called", path)
	returnArgs := c.Mock.Called(path)
	return returnArgs.Get(0).([]byte),
		returnArgs.Get(1).(*gozk.Stat),
		returnArgs.Error(2)
}

func (c *MockZookeeperClient) GetACL(path string) ([]gozk.ACL, *gozk.Stat, error) {
	log.Tracef("[ZooKeeper mock] GetACL(path=%s) called", path)
	returnArgs := c.Mock.Called(path)
	return returnArgs.Get(0).([]gozk.ACL),
		returnArgs.Get(1).(*gozk.Stat),
		returnArgs.Error(2)
}

func (c *MockZookeeperClient) GetW(path string) ([]byte, *gozk.Stat, <-chan gozk.Event, error) {
	log.Tracef("[ZooKeeper mock] GetW(path=%s) called", path)
	returnArgs := c.Mock.Called(path)
	return returnArgs.Get(0).([]byte),
		returnArgs.Get(1).(*gozk.Stat),
		returnArgs.Get(2).(<-chan gozk.Event),
		returnArgs.Error(3)
}

func (c *MockZookeeperClient) Multi(ops gozk.MultiOps) error {
	log.Trace("[ZooKeeper mock] Multi() called")
	return c.Mock.Called(ops).Error(0)
}

func (c *MockZookeeperClient) Reconnect() error {
	log.Trace("[ZooKeeper mock] Reconnect() called")
	return c.Mock.Called().Error(0)
}

func (c *MockZookeeperClient) Set(path string, data []byte, version int32) (*gozk.Stat, error) {
	log.Tracef("[ZooKeeper mock] Set(path=%s) called", path)
	returnArgs := c.Mock.Called(path, data, version)
	return returnArgs.Get(0).(*gozk.Stat),
		returnArgs.Error(1)
}

func (c *MockZookeeperClient) SetACL(path string, acl []gozk.ACL, version int32) (*gozk.Stat, error) {
	log.Tracef("[ZooKeeper mock] SetACL(path=%s) called", path)
	returnArgs := c.Mock.Called(path, acl, version)
	return returnArgs.Get(0).(*gozk.Stat),
		returnArgs.Error(1)
}

func (c *MockZookeeperClient) State() gozk.State {
	log.Trace("[ZooKeeper mock] State() called")
	return c.Mock.Called().Get(0).(gozk.State)
}

func (c *MockZookeeperClient) Sync(path string) (string, error) {
	log.Tracef("[ZooKeeper mock] Sync(path=%s) called", path)
	returnArgs := c.Mock.Called(path)
	return returnArgs.String(0),
		returnArgs.Error(1)
}

func (c *MockZookeeperClient) UpdateAddrs(addrs []string) error {
	log.Trace("[ZooKeeper mock] UpdateAddrs() called")
	return c.Mock.Called().Error(0)
}

func (c *MockZookeeperClient) NewLock(path string, acl []gozk.ACL) gozk.Locker {
	log.Tracef("[ZooKeeper mock] NewLock(path=%s) called", path)
	returnArgs := c.Mock.Called(path, acl)
	return returnArgs.Get(0).(gozk.Locker)
}

type MockLock struct {
	mock.Mock
}

func (l *MockLock) Lock() error {
	log.Tracef("[ZooKeeper mock] Lock() called")
	return l.Mock.Called().Error(0)
}

func (l *MockLock) Unlock() error {
	log.Tracef("[ZooKeeper mock] Unlock() called")
	return l.Mock.Called().Error(0)
}

func (l *MockLock) SetTTL(d time.Duration) {
	log.Tracef("[ZooKeeper mock] SetTTL() called")
	l.Mock.Called(d)
}

func (l *MockLock) SetTimeout(d time.Duration) {
	log.Tracef("[ZooKeeper mock] SetTimeout() called")
	l.Mock.Called(d)
}
