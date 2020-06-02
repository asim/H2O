package cassandra

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/service/config"
	"github.com/HailoOSS/service/dns"
	"github.com/HailoOSS/gossie/src/gossie"
)

var (
	defaultPort  = 9160
	defaultHosts = []string{"localhost:" + strconv.Itoa(defaultPort)}
	defaultTier  = "general"
)

var (
	mtx                    sync.RWMutex
	pools                  map[string]gossie.ConnectionPool
	retries                int
	readCl                 string
	writeCl                string
	timeout                time.Duration
	nodes                  []string
	once                   sync.Once
	DefaultPoolConstructor func([]string, string, gossie.PoolOptions) (gossie.ConnectionPool, error) = gossie.NewConnectionPool
	// Pointer to the underlying ConnectionPool constructor to use (used to switch out the pool implementation when
	// mocking)
	PoolConstructor func([]string, string, gossie.PoolOptions) (gossie.ConnectionPool, error) = DefaultPoolConstructor

	// map of authentication options for each keyspace
	auth map[string]*authenticationOptions
)

type authenticationOptions struct {
	username string
	password string
}

// String obfuscates our password when printing out the authenticationOptions struct
func (a *authenticationOptions) String() string {
	return fmt.Sprintf("username: %s, password: xxxxxxxx", a.username)
}

func init() {
	pools = make(map[string]gossie.ConnectionPool)
	auth = make(map[string]*authenticationOptions)
}

func getHosts() []string {
	cassandraHostKey := getCassandraHostConfigKey()
	config.WaitUntilLoaded(5 * time.Second)
	port := config.AtPath("hailo", "service", "cassandra", "defaults", "thriftPort").AsInt(defaultPort)
	if hosts := config.AtPath("hailo", "service", "cassandra", cassandraHostKey).AsHostnameArray(port); len(hosts) > 0 {
		return hosts
	}

	// No hosts returned: try DNS
	tier := config.AtPath("hailo", "service", "cassandra", "tier").AsString("premium")
	hosts, err := dns.Hosts("cassandra-" + tier)
	if err != nil {
		log.Errorf("Failed to load Cassandra hosts from dns: %v", err)
		return defaultHosts
	}

	if len(hosts) == 0 {
		return defaultHosts
	}
	// We need to append the port to hosts coming from DNS
	for i, host := range hosts {
		hosts[i] = host + fmt.Sprintf(":%d", port)
	}

	return hosts
}

// try to connect, then kick off listener for config changes
func setup() {
	ch := config.SubscribeChanges()
	go func() {
		for {
			<-ch
			reconnectDefault()
		}
	}()

	reconnectDefault()
}

func reconnectDefault() {
	mtx.Lock()
	defer mtx.Unlock()

	log.Infof("Reloading cassandra configuration")

	retries = config.AtPath("hailo", "service", "cassandra", "defaults", "maxRetries").AsInt(5)
	readCl = config.AtPath("hailo", "service", "cassandra", "defaults", "readConsistencyLevel").AsString("ONE")
	writeCl = config.AtPath("hailo", "service", "cassandra", "defaults", "writeConsistencyLevel").AsString("ONE")
	timeout = config.AtPath("hailo", "service", "cassandra", "defaults", "recvTimeout").AsDuration("1s")

	log.Debugf("Setting Cassandra defaults retries:%v, readCl: %v, writeCl: %v, timeout: %v from config", retries, readCl, writeCl, timeout)

	nodes = getHosts()
	log.Debugf("Setting Cassandra nodes %v from config", nodes)

	// Set up authentication if enabled
	if authEnabled := config.AtPath("hailo", "service", "cassandra", "authentication", "enabled").AsBool(); authEnabled {

		// Get config as json as its effectively a map[string]map[string]string
		authconfig := config.AtPath("hailo", "service", "cassandra", "authentication", "keyspaces").AsJson()

		// Parse and set if successful
		a, err := parseAuth(authconfig)
		if err == nil {
			auth = a
			log.Debugf("Setting Cassandra authentication from config: %v", a)
		} else {
			log.Warnf("Failed to set Cassandra authentication from config: %v", err)
		}

	}

	// Reset the pools map
	pools = make(map[string]gossie.ConnectionPool)
}

// ConnectionPool yields a configured connection pool for the given keyspace
// This should be called on each use, as this will mean that you will always
// get an up-to-date connection pool (automatically updated if the config
// changes)
func ConnectionPool(ks string) (gossie.ConnectionPool, error) {
	once.Do(setup)
	if pool := getPool(ks); pool != nil {
		return pool, nil
	}

	p, err := newPool(ks)
	return p, err
}

func getPool(ks string) gossie.ConnectionPool {
	mtx.RLock()
	defer mtx.RUnlock()

	if pool, ok := pools[ks]; ok {
		return pool
	}

	return nil
}

func newPool(ks string) (gossie.ConnectionPool, error) {
	mtx.Lock()
	defer mtx.Unlock()

	// double check existence now we have full lock
	if pool, ok := pools[ks]; ok {
		return pool, nil
	}

	opts := gossie.PoolOptions{
		ReadConsistency:  cl(readCl),
		WriteConsistency: cl(writeCl),
		Timeout:          int(timeout.Nanoseconds() / int64(time.Millisecond)),
		Retries:          retries,
	}

	// Add authentication options if set for this keyspace
	if a := auth[ks]; a != nil {
		opts.Authentication = map[string]string{
			"keyspace": ks,
			"username": a.username,
			"password": a.password,
		}
	}

	log.Debugf("Initialising Cassandra connection pool for KS %s connecting to %v with options %v", ks, nodes, opts)
	p, err := PoolConstructor(nodes, ks, opts)
	if err != nil {
		return nil, err
	}
	pools[ks] = p

	return pools[ks], nil
}

func cl(val string) int {
	switch val {
	case "ONE":
		return gossie.CONSISTENCY_ONE
	case "LOCAL_QUORUM":
		return gossie.CONSISTENCY_LOCAL_QUORUM
	case "QUORUM":
		return gossie.CONSISTENCY_QUORUM
	case "EACH_QUORUM":
		return gossie.CONSISTENCY_EACH_QUORUM
	case "ALL":
		return gossie.CONSISTENCY_ALL
	case "TWO":
		return gossie.CONSISTENCY_TWO
	case "THREE":
		return gossie.CONSISTENCY_THREE
	}

	return gossie.CONSISTENCY_DEFAULT
}

// getCassandraHostConfigPath gets the config key that should be used to load cassandra hosts
// this is to support multiple 'tiered' cassandra clusters
func getCassandraHostConfigKey() string {

	// Check what cluster are we supposed to contact, revert to the default config if not specified
	tier := config.AtPath("hailo", "service", "cassandra", "tier").AsString(defaultTier)
	if tier == "" {
		tier = defaultTier
	}
	log.Debugf("Attempting to connect to the %v Cassandra cluster", tier)

	var cassandraHosts string
	switch tier {
	case "general":
		cassandraHosts = "hosts"
	default:
		cassandraHosts = fmt.Sprintf("%sHosts", tier)
	}

	return cassandraHosts
}

// parseAuth converts our config []byte to a map of authentication options per keyspace
func parseAuth(js []byte) (map[string]*authenticationOptions, error) {
	ret := make(map[string]*authenticationOptions)

	// Blank config = no auth
	if len(js) == 0 {
		return ret, nil
	}

	// Unmarshal json to intermediate structure
	c := make(map[string]map[string]string)
	if err := json.Unmarshal(js, &c); err != nil {
		log.Warnf("Failed to unmarshal cassandra authentication configuration: %v", err) // don't log data in an attempt to not log our passwords...
		return ret, err
	}

	// Pull out auth for each configured keyspace
	for ks, v := range c {
		ret[ks] = &authenticationOptions{
			username: v["username"],
			password: v["password"],
		}
		log.Debugf("[Cassandra] Authentication options for keyspace '%s' loaded: %v", ks, ret[ks])
	}

	return ret, nil
}
