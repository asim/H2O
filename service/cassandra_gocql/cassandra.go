package cassandra_gocql

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/go-hostpool"

	"github.com/gocql/gocql"
	platutil "github.com/HailoOSS/platform/util"
	"github.com/HailoOSS/service/config"
	"github.com/HailoOSS/service/dns"
)

var cql_port = 9042

var (
	mtx                       sync.RWMutex
	readSessions              map[string]*gocql.Session
	writeSessions             map[string]*gocql.Session
	retries                   int
	readCl                    string
	writeCl                   string
	timeout                   time.Duration
	nodes                     []string
	once                      sync.Once
	DefaultSessionConstructor func([]string, int, string, bool) (*gocql.Session, error) = newSession
	SessionConstructor        func([]string, int, string, bool) (*gocql.Session, error) = DefaultSessionConstructor

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
	readSessions = make(map[string]*gocql.Session)
	writeSessions = make(map[string]*gocql.Session)
	auth = make(map[string]*authenticationOptions)
}

func getHosts() []string {
	cassandraHostKey := getCassandraHostConfigKey()
	if hosts := config.AtPath("hailo", "service", "cassandra", cassandraHostKey).AsStringArray(); len(hosts) > 0 {
		// strip any port numbers from config
		for i, host := range hosts {
			portSplit := strings.Split(host, ":")
			if len(portSplit) > 1 {
				// just use hostname
				hosts[i] = portSplit[0]
			}
		}
		return hosts
	}

	// no hosts returned so try dns
	tier := config.AtPath("hailo", "service", "cassandra", "tier").AsString("premium")
	hosts, err := dns.Hosts("cassandra-" + tier)
	if err != nil {
		log.Errorf("Failed to load Cassandra hosts from dns: %v", err)
		return []string{"localhost"}
	}

	// for safety fall back to localhost
	if len(hosts) == 0 {
		return []string{"localhost"}
	}

	// append port
	for i, host := range hosts {
		hosts[i] = host
	}

	return hosts
}

// try to connect, then kick off listener for config changes
func setup() {
	log.Warnf("DEPRECATED: cassandra_gocql should NOT to be used on any new project")

	ch := config.SubscribeChanges()
	go func() {
		for {
			<-ch
			reconnectDefault()
		}
	}()

	reconnectDefault()
}

// JKW It seems to me that relative to the gocassa cnfig code there is an awful lot less here - review
func reconnectDefault() {
	mtx.Lock()

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

	readKeyspaces := make([]string, 0, len(readSessions))
	writeKeyspaces := make([]string, 0, len(writeSessions))

	// Close all the sessions - avoid copying the Session objects
	for ks, s := range readSessions {
		readKeyspaces = append(readKeyspaces, ks)
		s.Close()
	}
	for ks, s := range writeSessions {
		writeKeyspaces = append(writeKeyspaces, ks)
		s.Close()
	}

	// Reset the sessions maps
	readSessions = make(map[string]*gocql.Session, len(readKeyspaces))
	writeSessions = make(map[string]*gocql.Session, len(writeKeyspaces))

	mtx.Unlock()

	// recreates the connections
	for _, ks := range readKeyspaces {
		SessionConstructor(nodes, cql_port, ks, true)
	}

	for _, ks := range writeKeyspaces {
		SessionConstructor(nodes, cql_port, ks, false)
	}
}

// ReadSession yields a configured cql session for the given keyspace with consistency set based upon "readConsistencyLevel"
// This should be called on each use, as this will mean that you will always
// get an up-to-date Session (automatically updated if the config
// changes)
func ReadSession(ks string) (*gocql.Session, error) {
	return session(ks, true)

}

// WriteSession yields a configured cql session for the given keyspace with consistency set based upon "writeConsistencyLevel"
// This should be called on each use, as this will mean that you will always
// get an up-to-date Session (automatically updated if the config
// changes)
func WriteSession(ks string) (*gocql.Session, error) {
	return session(ks, false)

}

func session(ks string, readOnly bool) (*gocql.Session, error) {
	once.Do(setup)

	mtx.RLock()
	if session := getSession(ks, readOnly); session != nil {
		mtx.RUnlock()
		return session, nil
	}

	mtx.RUnlock()

	return SessionConstructor(nodes, cql_port, ks, readOnly)
}

func newSession(hosts []string, port int, keyspace string, readonly bool) (*gocql.Session, error) {
	mtx.Lock()
	defer mtx.Unlock()

	// Check again
	if session := getSession(keyspace, readonly); session != nil {
		return session, nil
	}

	log.Infof("Host list for CQL: %v", hosts)
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = keyspace
	cluster.Port = port

	cluster.Timeout = timeout
	cluster.RetryPolicy = &gocql.SimpleRetryPolicy{NumRetries: retries}

	if readonly {
		cluster.Consistency = cl(readCl)
	} else {
		cluster.Consistency = cl(writeCl)
	}

	cluster.ProtoVersion = config.AtPath("hailo", "service", "cassandra", "defaults", "protoVersion").AsInt(2)
	cluster.Compressor = gocql.SnappyCompressor{}
	cluster.NumConns = config.AtPath("hailo", "service", "cassandra", "defaults", "maxHostConns").AsInt(2)
	//	cluster.Authenticator = gocql.PasswordAuthenticator{
	//		Username: c.username,
	//		Password: c.password,
	//	}

	cluster.PoolConfig.HostSelectionPolicy = gocql.HostPoolHostPolicy(
		hostpool.NewEpsilonGreedy(hosts, 5*time.Minute, &hostpool.LinearEpsilonValueCalculator{}),
	)
	cluster.Discovery.DcFilter = platutil.GetAwsRegionName()

	s, err := cluster.CreateSession()
	if err == nil {
		// save new session for keyspace
		saveSession(s, keyspace, readonly)
	}

	return s, err
}

func getSession(ks string, readOnly bool) *gocql.Session {
	if readOnly {
		if readSession, ok := readSessions[ks]; ok {
			return readSession
		}

	} else {
		if writeSession, ok := writeSessions[ks]; ok {
			return writeSession
		}

	}

	return nil
}

func saveSession(session *gocql.Session, ks string, readOnly bool) {
	if readOnly {
		readSessions[ks] = session

	} else {
		writeSessions[ks] = session
	}
}

func cl(val string) gocql.Consistency {
	switch val {
	case "ONE":
		return gocql.One
	case "LOCAL_QUORUM":
		return gocql.LocalQuorum
	case "QUORUM":
		return gocql.Quorum
	case "EACH_QUORUM":
		return gocql.EachQuorum
	case "ALL":
		return gocql.All
	case "TWO":
		return gocql.Two
	case "THREE":
		return gocql.Three
	}

	return gocql.Any
}

// getCassandraHostConfigPath gets the config key that should be used to load cassandra hosts
// this is to support multiple 'tiered' cassandra clusters
func getCassandraHostConfigKey() string {

	// Check what cluster are we supposed to contact, revert to the default config if not specified
	tier := config.AtPath("hailo", "service", "cassandra", "tier").AsString("general")
	if tier == "" {
		tier = "general"
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
