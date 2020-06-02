package memcache

import (
	"fmt"
	"time"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/service/config"
	"github.com/HailoOSS/service/dns"
	inst "github.com/HailoOSS/service/instrumentation"
	"github.com/HailoOSS/gomemcache/memcache"
)

const (
	// The default timeout used for dialling memcached
	defaultDialTimeout = "500ms"
	// The default timeout used for memcached operations
	defaultOperationTimeout = "100ms"
	// Sample rate of timing events for Memcached
	timingSampleRate = 0.33
)

type MemcacheClient interface {
	Add(item *memcache.Item) error
	CompareAndSwap(item *memcache.Item) error
	Decrement(key string, delta uint64) (newValue uint64, err error)
	Delete(key string) error
	Get(key string) (item *memcache.Item, err error)
	GetMulti(keys []string) (map[string]*memcache.Item, error)
	Increment(key string, delta uint64) (newValue uint64, err error)
	Set(item *memcache.Item) error
}

var (
	defaultClient MemcacheClient = newdefaultClient()
)

func getHosts() []string {
	hostConfigPath := []string{"hailo", "service", "memcache", "servers"}
	host := "memcached"

	// check if tier is specified and act accordingly
	tier := config.AtPath("hailo", "service", "memcache", "tier").AsString("")
	if tier != "" {
		hostConfigPath = append(hostConfigPath, tier)
		host = fmt.Sprintf("%s-%s", host, tier)
	}

	if hosts := config.AtPath(hostConfigPath...).AsHostnameArray(11211); len(hosts) > 0 {
		return hosts
	}

	// no hosts returned so try dns
	hosts, err := dns.Hosts(host)
	if err != nil {
		log.Errorf("[Memcache] Failed to load hosts from dns, returning empty list: %v", err)
		return []string{}
	}

	// append port
	for i, host := range hosts {
		hosts[i] = host + ":11211"
	}

	return hosts
}

func loadFromConfig(sl *memcache.ServerList, client *memcache.Client) {
	hosts := getHosts()
	log.Tracef("[Memcache] Setting memcache servers from config: %v", hosts)
	err := sl.SetServers(hosts...)
	if err != nil {
		log.Errorf("[Memcache] Error setting memcache servers: %v", err)
	}

	// Technically we have a race here since the timeouts are not protected by a mutex, however it isn't really a
	// problem if the timeout is stale for a short period.
	client.Timeout = config.AtPath("hailo", "service", "memcache", "timeouts", "operationTimeout").
		AsDuration(defaultOperationTimeout)
	log.Tracef("[Memcache] Set Memcache operation timeout from config: %v", client.Timeout)
	client.DialTimeout = config.AtPath("hailo", "service", "memcache", "timeouts", "dialTimeout").
		AsDuration(defaultDialTimeout)
	log.Tracef("[Memcache] Set Memcache dial timeout from config: %v", client.DialTimeout)
}

func newdefaultClient() MemcacheClient {
	serverSelector := new(memcache.ServerList)
	client := memcache.NewFromSelector(serverSelector)

	// Listen for config changes
	ch := config.SubscribeChanges()
	go func() {
		for _ = range ch {
			loadFromConfig(serverSelector, client)
		}
	}()

	loadFromConfig(serverSelector, client)

	// Log on init
	hosts := config.AtPath("hailo", "service", "memcache", "servers").AsHostnameArray(11211)
	operationTimeout := config.AtPath("hailo", "service", "memcache", "timeouts", "dialTimeout").
		AsDuration(defaultDialTimeout)
	dialTimeout := config.AtPath("hailo", "service", "memcache", "timeouts", "operationTimeout").
		AsDuration(defaultOperationTimeout)

	log.Infof("[Memcache] Initialising Memcache client to hosts %v: dial timeout %v, op timeout: %v", hosts,
		dialTimeout, operationTimeout)

	return client
}

func Add(item *memcache.Item) error {
	start := time.Now()
	defer inst.Timing(timingSampleRate, "memcached.add", time.Since(start))
	return defaultClient.Add(item)
}

func CompareAndSwap(item *memcache.Item) error {
	start := time.Now()
	defer inst.Timing(timingSampleRate, "memcached.compare-and-swap", time.Since(start))
	return defaultClient.CompareAndSwap(item)
}

func Decrement(key string, delta uint64) (newValue uint64, err error) {
	start := time.Now()
	defer inst.Timing(timingSampleRate, "memcached.decrement", time.Since(start))
	return defaultClient.Decrement(key, delta)
}

func Delete(key string) error {
	start := time.Now()
	defer inst.Timing(timingSampleRate, "memcached.delete", time.Since(start))
	return defaultClient.Delete(key)
}

func Get(key string) (item *memcache.Item, err error) {
	start := time.Now()
	defer inst.Timing(timingSampleRate, "memcached.get", time.Since(start))
	return defaultClient.Get(key)
}

func GetMulti(keys []string) (map[string]*memcache.Item, error) {
	start := time.Now()
	defer inst.Timing(timingSampleRate, "memcached.get-multi", time.Since(start))
	return defaultClient.GetMulti(keys)
}

func Increment(key string, delta uint64) (newValue uint64, err error) {
	start := time.Now()
	defer inst.Timing(timingSampleRate, "memcached.increment", time.Since(start))
	return defaultClient.Increment(key, delta)
}

func Set(item *memcache.Item) error {
	start := time.Now()
	defer inst.Timing(timingSampleRate, "memcached.set", time.Since(start))
	return defaultClient.Set(item)
}
