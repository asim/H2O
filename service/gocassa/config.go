package gocassa

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/gocql/gocql"
	"github.com/HailoOSS/go-hostpool"

	platutil "github.com/HailoOSS/platform/util"
	"github.com/HailoOSS/service/config"
	"github.com/HailoOSS/service/dns"
)

var (
	defaultPort  = 9042
	defaultHosts = []string{"localhost:" + strconv.Itoa(defaultPort)}
	defaultTier  = "general"
)

// ksConfig represents an (immutable) keyspace configuration.
type ksConfig struct {
	ks       string
	hosts    []string
	username string
	password string
	retries  int
	cl       gocql.Consistency
	timeout  time.Duration
	cc       *gocql.ClusterConfig
}

// hash returns a hashsum of the contents, used to determine if configuration has changed
func (c ksConfig) hash() uint32 {
	hasher := fnv.New32a()
	io.WriteString(hasher, c.username)
	io.WriteString(hasher, c.password)
	io.WriteString(hasher, strconv.Itoa(c.retries))
	io.WriteString(hasher, strconv.Itoa(int(c.cl)))
	io.WriteString(hasher, strconv.Itoa(int(c.timeout.Nanoseconds())))
	for _, h := range sort.StringSlice(c.hosts) { // Ordering variations are insignificant
		io.WriteString(hasher, h)
	}
	return hasher.Sum32()
}

// String returns a description of the config (excluding sensitive data)
func (c ksConfig) String() string {
	result := []string{}
	result = append(result, fmt.Sprintf("ks=%s", c.ks))
	result = append(result, fmt.Sprintf("hosts=%v", c.hosts))
	if c.username != "" {
		result = append(result, fmt.Sprintf("username=%s", c.username))
	}
	if c.password != "" {
		result = append(result, "password=***")
	}
	result = append(result, fmt.Sprintf("retries=%d", c.retries))
	result = append(result, fmt.Sprintf("timeout=%s", c.timeout.String()))
	return strings.Join(result, "; ")
}

func clFromString(clStr string) gocql.Consistency {
	switch strings.ToLower(clStr) {
	case "any":
		return gocql.Any
	case "one":
		return gocql.One
	case "two":
		return gocql.Two
	case "three":
		return gocql.Three
	case "quorum":
		return gocql.Quorum
	case "all":
		return gocql.All
	case "local_quorum", "localquorum":
		return gocql.LocalQuorum
	case "each_quorum", "eachquorum":
		return gocql.EachQuorum
	case "local_one", "localone":
		return gocql.LocalOne
	default:
		return gocql.LocalQuorum
	}
}

func getKsConfig(ks string) (ksConfig, error) {
	if !config.WaitUntilLoaded(5 * time.Second) {
		return ksConfig{}, fmt.Errorf("Config not loaded")
	}

	username, password, err := ksAuth(ks)
	if err != nil {
		return ksConfig{}, err
	}

	c := ksConfig{
		ks:       ks,
		hosts:    getHosts(),
		username: username,
		password: password,
		retries:  config.AtPath("hailo", "service", "cassandra", "defaults", "maxRetries").AsInt(5),
		cl:       clFromString(config.AtPath("hailo", "service", "cassandra", "defaults", "consistencyLevel").AsString("")),
		timeout:  config.AtPath("hailo", "service", "cassandra", "defaults", "recvTimeout").AsDuration("1s"),
	}
	cc := gocql.NewCluster(c.hosts...)
	cc.ProtoVersion = config.AtPath("hailo", "service", "cassandra", "defaults", "protoVersion").AsInt(2)
	cc.Consistency = c.cl
	cc.Compressor = gocql.SnappyCompressor{}
	cc.NumConns = config.AtPath("hailo", "service", "cassandra", "defaults", "maxHostConns").AsInt(2)
	cc.Authenticator = gocql.PasswordAuthenticator{
		Username: c.username,
		Password: c.password,
	}
	cc.Timeout = c.timeout
	cc.Keyspace = c.ks
	cc.RetryPolicy = &gocql.SimpleRetryPolicy{
		NumRetries: c.retries,
	}
	cc.PoolConfig.HostSelectionPolicy = gocql.HostPoolHostPolicy(
		hostpool.NewEpsilonGreedy(c.hosts, 5*time.Minute, &hostpool.LinearEpsilonValueCalculator{}),
	)
	cc.Discovery.DcFilter = platutil.GetAwsRegionName()
	c.cc = cc
	return c, nil
}

func getHosts() []string {
	port := config.AtPath("hailo", "service", "cassandra", "defaults", "cqlPort").AsInt(defaultPort)
	hosts := config.AtPath("hailo", "service", "cassandra", hostsCfgKey()).AsHostnameArray(port)
	if len(hosts) > 0 {
		return hosts
	}

	// No hosts returned: try DNS
	tier := config.AtPath("hailo", "service", "cassandra", "tier").AsString("premium")
	hosts, err := dns.Hosts("cassandra-" + tier)
	if err != nil {
		log.Errorf("[Cassandra] Failed to load hosts from DNS: %s", err.Error())
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

// hostsCfgKey gets the config key that should be used to load cassandra hosts. This is to support multiple 'tiered'
// Cassandra clusters.
func hostsCfgKey() string {
	// Check what cluster are we supposed to contact, revert to the default config if not specified
	tier := config.AtPath("hailo", "service", "cassandra", "tier").AsString("")
	if tier == "" {
		tier = defaultTier
	}

	switch tier {
	case "general":
		return "hosts"
	default:
		return fmt.Sprintf("%sHosts", tier)
	}
}

// ksAuth returns the username and password for the given keyspace
func ksAuth(ks string) (string, string, error) {
	if !config.AtPath("hailo", "service", "cassandra", "authentication", "enabled").AsBool() {
		return "", "", nil
	}

	confJson := config.AtPath("hailo", "service", "cassandra", "authentication", "keyspaces").AsJson()
	rawConf := make(map[string]map[string]string, 5)
	if err := json.Unmarshal(confJson, &rawConf); err != nil {
		// Don't log raw data in an attempt to not log our passwords
		log.Warnf("[Cassandra] Failed to unmarshal authentication configuration: %s", err.Error())
		return "", "", err
	}

	for candidateKs, v := range rawConf {
		if candidateKs == ks {
			return v["username"], v["password"], nil
		}
	}

	return "", "", nil
}
