package graphite

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/service/config"
	inst "github.com/HailoOSS/service/instrumentation"
	"github.com/mreiferson/go-httpclient"
)

// DefaultConnection is used, by default, to execute Query()
var DefaultConnection *Connection = NewConnection()

// Connection represents a connection to Grapnite
type Connection struct {
	sync.RWMutex

	// connection config
	host, scheme  string
	port          int
	connTimeout   time.Duration
	reqTimeout    time.Duration
	rspHdrTimeout time.Duration
	hash          string

	transport *httpclient.Transport
	client    *http.Client
}

// NewConnection mints a new Graphite connection which listens for config changes
func NewConnection() *Connection {
	conn := &Connection{}
	ch := config.SubscribeChanges()
	go func() {
		for {
			<-ch
			conn.loadConfig()
		}
	}()
	conn.loadConfig()

	return conn
}

// loadConfig gets host, scheme, port and timeouts from config service
func (conn *Connection) loadConfig() {
	// reload host/scheme/port and see if differnet
	host := config.AtPath("hailo", "service", "graphite", "host").AsString("graphite-internal-test.elasticride.com")
	scheme := config.AtPath("hailo", "service", "graphite", "scheme").AsString("http")
	port := config.AtPath("hailo", "service", "graphite", "port").AsInt(-1)
	connTimeout := config.AtPath("hailo", "service", "graphite", "connectTimeout").AsDuration("300ms")
	reqTimeout := config.AtPath("hailo", "service", "graphite", "requestTimeout").AsDuration("2s")
	rspHdrTimeout := config.AtPath("hailo", "service", "graphite", "responseHeaderTimeout").AsDuration("1s")

	// hash all this and compare
	h := md5.New()
	io.WriteString(h, host)
	io.WriteString(h, scheme)
	io.WriteString(h, fmt.Sprintf("%v", port))
	io.WriteString(h, fmt.Sprintf("%v", connTimeout))
	io.WriteString(h, fmt.Sprintf("%v", reqTimeout))
	io.WriteString(h, fmt.Sprintf("%v", rspHdrTimeout))
	reloadHash := fmt.Sprintf("%x", h.Sum(nil))

	if conn.currentHash() == reloadHash {
		// no change
		return
	}

	conn.Lock()
	defer conn.Unlock()

	conn.host, conn.scheme, conn.port = host, scheme, port
	conn.connTimeout, conn.reqTimeout, conn.rspHdrTimeout = connTimeout, reqTimeout, rspHdrTimeout

	// init transport
	if conn.transport != nil {
		conn.transport.Close()
	}
	conn.transport = conn.newTimeoutTransport()
	conn.client = &http.Client{Transport: conn.transport}
}

// hostSchemePort reads these things protected by RLock
func (conn *Connection) hostSchemePort() (string, string, int) {
	conn.RLock()
	defer conn.RUnlock()
	return conn.host, conn.scheme, conn.port
}

// hash reads current config hash protected by RLock
func (conn *Connection) currentHash() string {
	conn.RLock()
	defer conn.RUnlock()
	return conn.hash
}

// newTimeoutTransport mints a transport for HTTP requests which respects timeouts
func (conn *Connection) newTimeoutTransport() *httpclient.Transport {
	t := &httpclient.Transport{}
	t.ConnectTimeout = conn.connTimeout
	t.RequestTimeout = conn.reqTimeout
	t.ResponseHeaderTimeout = conn.rspHdrTimeout
	return t
}

// Query executes a query to fetch data from graphite - returning raw JSON-encoded bytes
// The supplied URL values will be augmented with hostname, port
func (conn *Connection) Query(path string, v url.Values) ([]byte, error) {
	hostname, scheme, port := conn.hostSchemePort()
	if port > 0 {
		hostname = fmt.Sprintf("%v:%v", hostname, port)
	}
	url := &url.URL{Scheme: scheme, Host: hostname, Path: path, RawQuery: v.Encode()}
	res, err := conn.doQuery(url)
	if err != nil {
		inst.Counter(0.1, "query.fail", 1)
		// try to unmangle transport -- this was to avoid an error we were having where
		// we have "failed to send on a closed socket" or somesuch
		// @imsnakes advises us that we should hope to fix this in the http client lib
		// However for now we'll just take this slightly heavy-handed approach
		conn.Lock()
		defer conn.Unlock()
		conn.transport.Close()
		conn.transport = conn.newTimeoutTransport()
		conn.client = &http.Client{Transport: conn.transport}
	} else {
		inst.Counter(0.1, "query.success", 1)
	}
	return res, err
}

// Children wraps DefaultConnection.Children()
func Children(q string) ([]string, error) {
	return DefaultConnection.Children(q)
}

// Children returns a list of target IDs within a specific node in the target tree
// An example q would be `stats.com.HailoOSS` to find all child nodes of this
func (conn *Connection) Children(q string) ([]string, error) {
	v := url.Values{}
	v.Set("format", "treejson")
	v.Set("query", fmt.Sprintf("%v.*", q))
	v.Set("contexts", "1")
	v.Set("path", q)
	v.Set("node", q)

	b, err := DefaultConnection.Query("/metrics/find/", v)
	if err != nil {
		return nil, fmt.Errorf("Query execution error: %v", err)
	}

	r, err := unmarshalChildren(b)
	return r, err

}

type node struct {
	Id string `json:"id"`
}

func unmarshalChildren(b []byte) ([]string, error) {
	var grRes []*node = make([]*node, 0)
	if err := json.Unmarshal(b, &grRes); err != nil {
		return nil, fmt.Errorf("Children result unmarshaling error: %v", err)
	}
	ret := make([]string, len(grRes))
	for i, n := range grRes {
		ret[i] = n.Id
	}
	return ret, nil
}

// doQuery is our low level call to make an HTTP request -- call query() from elsewhere, not this
func (conn *Connection) doQuery(url *url.URL) ([]byte, error) {
	log.Debugf("Querying Graphite: %v", url.String())

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	conn.RLock()
	defer conn.RUnlock()

	t := time.Now()
	rsp, err := conn.client.Do(req)
	inst.Timing(0.1, "query.attempt", time.Since(t))
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	data, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}
