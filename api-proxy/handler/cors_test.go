package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/HailoOSS/service/config"
)

const CORSConfigJson = `{
    "controlplane": {
        "configVersion": 10001,
        "rules": {
            "test-throttle": {
                "match": {
                    "path": "/throttle-test",
                    "proportion": 1.0
                },
                "action": 3
            }
        },
        "regions": {
            "eu-west-1": {
                "id": "eu-west-1",
                "status": "ONLINE",
                "apps": {
                    "default": {
                        "api": "api-driver-london.elasticride.com"
                    }
                }
            }
        },
        "hobRegions": {
            "LON": "eu-west-1"
        }
    }
}`

func TestCORS_AllowedOrigin(t *testing.T) {
	// Setup config to throttle our request
	buf := bytes.NewBufferString(H2ConfigJson)
	origConfigBuf := bytes.NewBuffer(config.Raw())
	config.Load(buf)
	defer config.Load(origConfigBuf)
	time.Sleep(time.Second)

	// ...and a dummy thin API server
	server, apiClient := SetupTestServerAndClient(t)
	defer TeardownTestServer(t, server)

	request, err := http.NewRequest("OPTIONS", fmt.Sprintf("%s/throttle-test", server.URL), nil)
	request.Header.Set("Origin", "http://hailoweb.com")
	request.Header.Set("Access-Control-Request-Headers", "Foo, Bar, Baz")
	assert.NoError(t, err, "Request construction error")

	response, err := apiClient.Do(request)
	assert.NoError(t, err, "Request error")

	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, "http://hailoweb.com", response.Header.Get("Access-Control-Allow-Origin"),
		"Access-Control-Allow-Origin header is not right")
	assert.Contains(t, "Foo, Bar, Baz", response.Header.Get("Access-Control-Allow-Headers"),
		"Access-Control-Allow-Headers is not reflective of the requested headers")
	assert.Contains(t, response.Header.Get("Access-Control-Allow-Methods"), "DELETE")
	assert.Contains(t, response.Header.Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, response.Header.Get("Access-Control-Allow-Methods"), "HEAD")
	assert.Contains(t, response.Header.Get("Access-Control-Allow-Methods"), "OPTIONS")
	assert.Contains(t, response.Header.Get("Access-Control-Allow-Methods"), "PUT")
	assert.Contains(t, response.Header.Get("Access-Control-Allow-Methods"), "POST")
}

func TestCORS_AllowedOrigin_Subdomain(t *testing.T) {
	// Setup config to throttle our request
	buf := bytes.NewBufferString(H2ConfigJson)
	origConfigBuf := bytes.NewBuffer(config.Raw())
	config.Load(buf)
	defer config.Load(origConfigBuf)
	time.Sleep(time.Second)

	// ...and a dummy thin API server
	server, apiClient := SetupTestServerAndClient(t)
	defer TeardownTestServer(t, server)

	request, err := http.NewRequest("OPTIONS", fmt.Sprintf("%s/throttle-test", server.URL), nil)
	request.Header.Set("Origin", "http://foo.bar-baz.boo.hailoweb.com")
	assert.NoError(t, err, "Request construction error")

	response, err := apiClient.Do(request)
	assert.NoError(t, err, "Request error")

	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, "http://foo.bar-baz.boo.hailoweb.com", response.Header.Get("Access-Control-Allow-Origin"),
		"Access-Control-Allow-Origin header is not right")
}
