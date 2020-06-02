package httphealthcheck

import (
	"fmt"
	"net/http"
	"time"

	"github.com/HailoOSS/service/healthcheck"
	"github.com/mreiferson/go-httpclient"
)

const HealthCheckId = "com.HailoOSS.service.http"

var (
	defaultHttpClient *http.Client
)

type UrlBuilder func() string

// HealthCheck asserts we can connect to some HTTP server and execute a GET and get back a 200 OK
func HealthCheck(urlBuilder UrlBuilder, timeout time.Duration) healthcheck.Checker {
	return func() (map[string]string, error) {
		ret := make(map[string]string)

		url := urlBuilder()
		ret["url"] = url

		if defaultHttpClient == nil {
			defaultHttpClient = newHttpClient(timeout)
		}

		rsp, err := defaultHttpClient.Get(url)
		if err != nil {
			// If there's an error we replace the connection
			defaultHttpClient = newHttpClient(timeout)
			return ret, err
		}
		ret["statusCode"] = fmt.Sprintf("%v", rsp.StatusCode)
		defer rsp.Body.Close()
		if rsp.StatusCode != 200 {
			return ret, fmt.Errorf("Did not return 200 OK")
		}
		return ret, nil
	}
}

func newHttpClient(timeout time.Duration) *http.Client {
	transport := &httpclient.Transport{}
	transport.ConnectTimeout = timeout
	transport.RequestTimeout = timeout
	transport.ResponseHeaderTimeout = timeout
	client := &http.Client{Transport: transport}
	return client
}
