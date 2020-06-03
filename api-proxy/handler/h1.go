package handler

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httputil"
	"regexp"
	"time"

	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/api-proxy/hostmapping"
	"github.com/hailo-platform/H2O/api-proxy/stats"
	inst "github.com/hailo-platform/H2O/service/instrumentation"
)

const (
	dialTimeout     = 5 * time.Second
	responseTimeout = 30 * time.Second
	idleConnections = 5
)

var (
	// The proxy is cached and shared among all requests
	v1Proxy http.Handler
	// Error message sent to the client when proxying fails
	proxyErrorPayload = []byte(`{"status":false,"payload":"Internal low-level service failure, cannot complete request","debug":{"errorCode":"proxy error"},"code":11}`)
	// The TLS configuration to use when creating a new proxy instance
	proxyTransportTLSConfig *tls.Config
	// Regular expression to match a CORS response header key
	corsResponseHeaderRegex *regexp.Regexp = regexp.MustCompile("^Access-Control-")
)

// ResponseWriter used for sending the response back from the H1 reverse proxy to the client. This both records the
// bytes written (so we can write a default failure response if there is no response written by the reverse proxy), and
// filters any CORS response headers that H1 tries to send to the client
type h1ProxyResponseWriter struct {
	http.ResponseWriter
	header  http.Header
	status  int
	written int
}

func (rw *h1ProxyResponseWriter) Write(data []byte) (int, error) {
	written, err := rw.ResponseWriter.Write(data)
	rw.written = rw.written + written
	return written, err
}

func (rw *h1ProxyResponseWriter) WriteHeader(status int) {
	// Add all headers to the ResponseWriter that aren't CORS headers
	for key, values := range rw.header {
		if !corsResponseHeaderRegex.MatchString(key) {
			for _, value := range values {
				rw.ResponseWriter.Header().Add(key, value)
			}
		}
	}

	rw.ResponseWriter.WriteHeader(status)
	rw.status = status
}

// isError tests if the result of serving was a 5xx error indicating something is wrong
func (rw *h1ProxyResponseWriter) isError() bool {
	return rw.status >= 500 && rw.status < 600
}

func (rw *h1ProxyResponseWriter) Header() http.Header {
	if rw.header == nil {
		rw.header = make(http.Header)
	}

	return rw.header
}

// h1Handler is responsible for proxying requests to H1
func h1Handler(rw http.ResponseWriter, r *http.Request) {
	start := time.Now()
	proxyRw := &h1ProxyResponseWriter{
		ResponseWriter: rw,
	}

	defer func() {
		if proxyRw.isError() {
			inst.Timing(1.0, h1_failure, time.Since(start))
			inst.Counter(1.0, h1_failure, 1)
			inst.Timing(1.0, h1_azFailure, time.Since(start))
			inst.Counter(1.0, h1_azFailure, 1)
		} else {
			inst.Timing(1.0, h1_success, time.Since(start))
			inst.Counter(1.0, h1_success, 1)
			inst.Timing(1.0, h1_azSuccess, time.Since(start))
			inst.Counter(1.0, h1_azSuccess, 1)
		}

		if r.URL.Path != "/" {
			stats.Record("/", !proxyRw.isError(), time.Since(start))
		}

		stats.Record(r.URL.Path, !proxyRw.isError(), time.Since(start))
		inst.Counter(1.0, "access.h1."+sanitizeKey(extractHobFromRequest(r))+"."+sanitizeKey(r.URL.Path), 1)
		instrumentHob(r, "")
	}()

	v1Proxy.ServeHTTP(proxyRw, r)

	if proxyRw.status == http.StatusInternalServerError && proxyRw.written == 0 {
		// most likely a proxy error since all V1 services should have written some response data
		_, err := proxyRw.Write(proxyErrorPayload)
		if err != nil {
			log.Errorf("Error writing proxy error payload: %v", err)
		}
	}
}

// setupProxy initiates our H1 proxy
func setupProxy() {
	director := func(req *http.Request) {
		req.URL.Scheme = "https"
		req.URL.Host = hostmapping.Map(req.Host)
		log.Tracef("[H1 proxy] Proxying request to %s://%s", req.URL.Scheme, req.URL.Host)
	}

	timeoutDialer := func(network, address string) (net.Conn, error) {
		return net.DialTimeout(network, address, dialTimeout)
	}

	v1Proxy = &httputil.ReverseProxy{
		Director: director,
		Transport: &http.Transport{
			Dial: timeoutDialer,
			ResponseHeaderTimeout: responseTimeout,
			MaxIdleConnsPerHost:   idleConnections,
			TLSClientConfig:       proxyTransportTLSConfig,
		},
	}
}
