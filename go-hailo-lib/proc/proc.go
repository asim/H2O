package proc

import (
	"net"
	"sync"
	"time"
)

var (
	cacheInterval = time.Duration(60) * time.Second
	cacheUpdated  time.Time
	mtx           sync.RWMutex
	connCache     map[string]int
	once          sync.Once
)

func setup() {
	loadConnCache(true)
}

func init() {
	once.Do(setup)
}

// connCacheLoader runs a loop which updates the tcp connection count cache.
func connCacheLoader() {
	for {
		loadConnCache(false)
		time.Sleep(cacheInterval)
	}
}

// loadConnCache loads the tcp connection count cache.
func loadConnCache(force bool) {
	mtx.Lock()
	defer mtx.Unlock()

	// If below cache interval just return
	u := time.Now().Sub(cacheUpdated).Seconds()
	if connCache != nil && force == false && u < cacheInterval.Seconds() {
		return
	}

	conns, err := establishedTcpConns()
	if err != nil {
		return
	}

	connCache = conns
	cacheUpdated = time.Now()
}

// numRemoteTcpConns returns the number of tcp establishes connections to a remote host.
// Host argument needs to be in the form host:port
func numRemoteTcpConns(host string) int {
	mtx.RLock()
	defer mtx.RUnlock()

	// sanity check
	if connCache == nil {
		return 0
	}

	// split host port
	hst, port, err := net.SplitHostPort(host)
	if err != nil {
		return 0
	}

	// Resolve the host
	ips, err := net.LookupIP(hst)
	if err != nil {
		return 0
	}

	// Get an ipv4 addr
	var addr string
	for _, ip := range ips {
		if i := ip.To4(); i != nil {
			addr = i.String()
			break
		}
	}

	// Found an ipv4 addr?
	if len(addr) == 0 {
		return 0
	}

	rhost := net.JoinHostPort(addr, port)
	return connCache[rhost]
}

// CachedNumRemoteTcpConns retrieves the number of established tcp connections for a remote host
// from the cache. Host argument should be of the form host:port
func CachedNumRemoteTcpConns(host string) int {
	loadConnCache(false)
	return numRemoteTcpConns(host)
}

// CachedRemoteTcpConns returns a map containing the number of tcp connections for all remote
// hosts connected to by this process.
func CachedRemoteTcpConns() map[string]int {
	loadConnCache(false)
	mtx.RLock()
	defer mtx.RUnlock()
	return connCache
}

// NumRemoteTcpConns forces a cache update and retrieves the number of established tcp connections
// for a remote host. Host argument should be of the form host:port
func NumRemoteTcpConns(host string) int {
	loadConnCache(true)
	return numRemoteTcpConns(host)
}

// RemoteTcpConns forces a cache update and returns a map containing the number of tcp connections
// for all remote hosts connected to by this process.
func RemoteTcpConns() map[string]int {
	loadConnCache(true)
	mtx.RLock()
	defer mtx.RUnlock()
	return connCache
}

// RunCacheLoader starts the cache loader in a background routine
func RunCacheLoader() {
	go connCacheLoader()
}
