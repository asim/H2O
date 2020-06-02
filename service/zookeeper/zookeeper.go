/*
Package to interface Hailo systems and config with Zookeeper

NOTE: All path parameters should be formated like below.
Slashes for the separator and the leading slash are required.
	path := "/path/to/my/data"
*/
package zookeeper

import (
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/platform/util"
	"github.com/HailoOSS/service/config"
	"github.com/HailoOSS/service/dns"
	gozk "github.com/HailoOSS/go-zookeeper/zk"
)

type ZookeeperClient interface {
	Children(path string) ([]string, *gozk.Stat, error)
	ChildrenW(path string) ([]string, *gozk.Stat, <-chan gozk.Event, error)
	Close()
	Create(path string, data []byte, flags int32, acl []gozk.ACL) (string, error)
	CreateProtectedEphemeralSequential(path string, data []byte, acl []gozk.ACL) (string, error)
	Delete(path string, version int32) error
	Exists(path string) (bool, *gozk.Stat, error)
	ExistsW(path string) (bool, *gozk.Stat, <-chan gozk.Event, error)
	Get(path string) ([]byte, *gozk.Stat, error)
	GetACL(path string) ([]gozk.ACL, *gozk.Stat, error)
	GetW(path string) ([]byte, *gozk.Stat, <-chan gozk.Event, error)
	Multi(ops gozk.MultiOps) error
	Reconnect() error
	Set(path string, data []byte, version int32) (*gozk.Stat, error)
	SetACL(path string, acl []gozk.ACL, version int32) (*gozk.Stat, error)
	State() gozk.State
	Sync(path string) (string, error)
	UpdateAddrs(addrs []string) error
	NewLock(path string, acl []gozk.ACL) gozk.Locker
}

// The default connection function used; replace the Connector value with this when you finish testing (if you mocked
// ZooKeeper)
func DefaultConnector(servers []string, recvTimeout time.Duration) (ZookeeperClient, <-chan gozk.Event, error) {
	// This is necessary because http://stackoverflow.com/a/24636486/414876
	return gozk.Connect(servers, recvTimeout)
}

var (
	mtx            sync.RWMutex
	once           syncOnce
	didSetup       bool            = false
	defaultClient  ZookeeperClient = nil
	defaultTimeout time.Duration
	currConfigHash string
	// The connection function used when creating a new ZK connection. Replace this with a mock function during tests
	// that test ZooKeeper integration
	Connector func(servers []string, recvTimeout time.Duration) (ZookeeperClient, <-chan gozk.Event, error) = DefaultConnector
)

// Try to connect, then kick off listener for config changes
func setup() {
	ch := config.SubscribeChanges()
	go func() {
		for _ = range ch {
			reconnectDefault()
		}
	}()
	reconnectDefault()
	didSetup = true
}

func hasConfigChanged(hostsSlice []string, duration time.Duration) bool {
	thisBytes := make([]byte, 0)
	for _, h := range hostsSlice {
		thisBytes = append(thisBytes, []byte(h)...)
	}
	durBytes := make([]byte, binary.Size(duration))
	binary.PutVarint(durBytes, int64(duration))
	thisBytes = append(thisBytes, durBytes...)
	hash := util.GetMD5Hash(thisBytes)

	mtx.Lock()
	changed := currConfigHash != hash
	if changed {
		currConfigHash = hash
	}
	mtx.Unlock()
	return changed
}

func connectDefault(hosts []string, recvTimeout time.Duration) {
	log.Infof("Attempting to connect to ZK on %v with timeout %v", hosts, recvTimeout)
	var err error
	var eventChan <-chan gozk.Event

	mtx.Lock()
	defaultClient, eventChan, err = Connector(hosts, recvTimeout)
	mtx.Unlock()

	if err != nil {
		log.Warnf("Failed to connect to ZK: %v", err)
	}

	go func() {
		for ev := range eventChan {
			log.Tracef("Received zk connection event: %v", ev)
		}

		log.Warnf("ZK connection event loop has closed")
	}()
}

func getHosts() []string {
	hostsConfigPath := []string{"hailo", "service", "zookeeper", "hosts"}
	tier := config.AtPath("hailo", "service", "zookeeper", "tier").AsString("general")
	if tier != "general" {
		hostsConfigPath = append(hostsConfigPath, tier)
	}

	if hosts := config.AtPath(hostsConfigPath...).AsHostnameArray(2181); len(hosts) > 0 {
		return hosts
	}

	// no hosts returned so try dns
	hosts, err := dns.Hosts("zookeeper-" + tier)
	if err != nil {
		log.Errorf("Failed to load ZK hosts from dns: %v", err)
		return []string{"localhost:2181"}
	}

	// for safety fall back to localhost
	if len(hosts) == 0 {
		return []string{"localhost:2181"}
	}

	// append port
	for i, host := range hosts {
		hosts[i] = host + ":2181"
	}

	return hosts
}

func reconnectDefault() {
	hosts := getHosts()
	recvTimeout := config.AtPath("hailo", "service", "zookeeper", "recvTimeout").AsDuration("100ms")
	if !hasConfigChanged(hosts, recvTimeout) {
		log.Infof("ZooKeeper config has not changed")
		return
	}

	if defaultClient != nil {
		if recvTimeout != defaultTimeout {
			// cannot gracefully set timeout so close it
			defaultTimeout = recvTimeout
			defaultClient.Close()
		} else {
			// update the hosts only
			log.Tracef("Setting ZK hosts to %v", hosts)
			defaultClient.UpdateAddrs(hosts)
			return
		}
	}

	connectDefault(hosts, recvTimeout)
}

// WaitForConnect will wait until we are connected to ZK successfully for duration N
func WaitForConnect(d time.Duration) error {
	once.Do(setup)

	timeout := time.Now().Add(d)
	for {
		mtx.RLock()
		st := defaultClient.State()
		mtx.RUnlock()

		if st == gozk.StateHasSession {
			return nil
		}
		if time.Now().After(timeout) {
			break
		}
		time.Sleep(time.Millisecond * 10)
	}
	return fmt.Errorf("Failed to achieve ZooKeeper connection within %v", d)
}

// CloseConnection closes the underlying network connection to zookeeper
// This will trigger both the send and recv loops to exit and requests to flush
// and then we will automatically attempt to reconnect
//
// Needless to say, this is the nuclear option, and should only be used
// as a part of higher level constructs in the event of a near fatal error
func CloseConnection() error {
	once.Do(setup)
	mtx.RLock()
	defer mtx.RUnlock()

	return defaultClient.Reconnect()
}

// Close and remove the connection, WITHOUT attempting to automatically reconnect
func TearDown() {
	mtx.RLock()
	defer mtx.RUnlock()

	if didSetup {
		defaultClient.Close()
		defaultClient = nil
		once.Reset()
		didSetup = false
		currConfigHash = ""
	}
}

func Children(path string) ([]string, *gozk.Stat, error) {
	once.Do(setup)
	mtx.RLock()
	defer mtx.RUnlock()

	children, stat, err := defaultClient.Children(path)
	return children, stat, err
}

func ChildrenW(path string) ([]string, *gozk.Stat, <-chan gozk.Event, error) {
	once.Do(setup)
	mtx.RLock()
	defer mtx.RUnlock()

	children, stat, ch, err := defaultClient.ChildrenW(path)
	return children, stat, ch, err
}

func Create(path string, data []byte, flags int32, acl []gozk.ACL) (string, error) {
	once.Do(setup)
	mtx.RLock()
	defer mtx.RUnlock()

	path, err := defaultClient.Create(path, data, flags, acl)
	return path, err
}

func CreateProtectedEphemeralSequential(path string, data []byte, acl []gozk.ACL) (string, error) {
	once.Do(setup)
	mtx.RLock()
	defer mtx.RUnlock()

	path, err := defaultClient.CreateProtectedEphemeralSequential(path, data, acl)
	return path, err
}

func Delete(path string, version int32) error {
	once.Do(setup)
	mtx.RLock()
	defer mtx.RUnlock()

	err := defaultClient.Delete(path, version)
	return err
}

func Exists(path string) (bool, *gozk.Stat, error) {
	once.Do(setup)
	mtx.RLock()
	defer mtx.RUnlock()

	ex, stat, err := defaultClient.Exists(path)
	return ex, stat, err
}

func ExistsW(path string) (bool, *gozk.Stat, <-chan gozk.Event, error) {
	once.Do(setup)
	mtx.RLock()
	defer mtx.RUnlock()

	ex, stat, ch, err := defaultClient.ExistsW(path)
	return ex, stat, ch, err
}

func Get(path string) ([]byte, *gozk.Stat, error) {
	once.Do(setup)
	mtx.RLock()
	defer mtx.RUnlock()

	data, stat, err := defaultClient.Get(path)
	return data, stat, err
}

func GetACL(path string) ([]gozk.ACL, *gozk.Stat, error) {
	once.Do(setup)
	mtx.RLock()
	defer mtx.RUnlock()

	acls, stat, err := defaultClient.GetACL(path)
	return acls, stat, err
}

func GetW(path string) ([]byte, *gozk.Stat, <-chan gozk.Event, error) {
	once.Do(setup)
	mtx.RLock()
	defer mtx.RUnlock()

	data, stat, ch, err := defaultClient.GetW(path)
	return data, stat, ch, err
}

func Multi(ops gozk.MultiOps) error {
	once.Do(setup)
	mtx.RLock()
	defer mtx.RUnlock()

	err := defaultClient.Multi(ops)
	return err
}

func Set(path string, data []byte, version int32) (*gozk.Stat, error) {
	once.Do(setup)
	mtx.RLock()
	defer mtx.RUnlock()

	stat, err := defaultClient.Set(path, data, version)
	return stat, err
}

func SetACL(path string, acl []gozk.ACL, version int32) (*gozk.Stat, error) {
	once.Do(setup)
	mtx.RLock()
	defer mtx.RUnlock()

	stat, err := defaultClient.SetACL(path, acl, version)
	return stat, err
}

func State() gozk.State {
	once.Do(setup)
	mtx.RLock()
	defer mtx.RUnlock()

	state := defaultClient.State()
	return state
}

func Sync(path string) (string, error) {
	once.Do(setup)
	mtx.RLock()
	defer mtx.RUnlock()

	s, err := defaultClient.Sync(path)
	return s, err
}

func NewLock(path string, acl []gozk.ACL) gozk.Locker {
	once.Do(setup)
	mtx.RLock()
	defer mtx.RUnlock()

	return defaultClient.NewLock(path, acl)
}

// CreateParents creates any parent nodes for the given path if required. If all
// the parent nodes already exist then no error is returned.
func CreateParents(path string) error {
	parts := strings.Split(path, "/")
	pth := ""
	for _, p := range parts[1:] {
		pth += "/" + p
		_, err := Create(pth, []byte{}, 0, gozk.WorldACL(gozk.PermAll))
		if err != nil && err != gozk.ErrNodeExists {
			return err
		}
	}
	return nil
}
