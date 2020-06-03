package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	ptesting "github.com/hailo-platform/H2O/platform/testing"
	"github.com/hailo-platform/H2O/service/config"
)

func TestHandlerSuite(t *testing.T) {
	suite := new(HandlerSuite)
	suite.LogLevel = "off"
	ptesting.RunSuite(t, suite)
}

type HandlerSuite struct {
	ptesting.Suite
	server *TestServer
	client *http.Client
}

func (suite *HandlerSuite) SetupTest() {
	suite.Suite.SetupTest()
	suite.server, suite.client = SetupTestServerAndClient(suite.T())
}

func (suite *HandlerSuite) TearDownTest() {
	TeardownTestServer(suite.T(), suite.server)
	suite.server, suite.client = nil, nil
	suite.Suite.TearDownTest()
}

// Test normal region pinning to the correct endpoint
func (suite *HandlerSuite) TestRegionPinning() {
	configJson := `{
		"controlplane": {
			"configVersion": 10001,
			"rules": {
				"test-throttle": {
					"match": {
						"path": "/throttle",
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
	buf := bytes.NewBufferString(configJson)
	origConfigBuf := bytes.NewBuffer(config.Raw())
	config.Load(buf)
	defer config.Load(origConfigBuf)
	time.Sleep(time.Second)

	server, client := suite.server, suite.client

	// A request with no HOB info (either as a parameter, or in the Host header), should not be pinned
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/throttle", server.URL), nil)
	suite.Assertions.NoError(err)
	resp, err := client.Do(req)
	suite.Assertions.NoError(err)
	pinnedHostname := resp.Header.Get("X-H-ENDPOINT-API")
	suite.Assertions.Equal("", pinnedHostname, "HOB-less request was pinned (it shouldn't be)")

	// A request made to LON (with the right hostname) should not be redirected
	req, err = http.NewRequest("GET", fmt.Sprintf("%s/throttle", server.URL), nil)
	suite.Assertions.NoError(err)
	req.Host = "api-driver-london.elasticride.com"
	resp, err = client.Do(req)
	suite.Assertions.NoError(err)
	pinnedHostname = resp.Header.Get("X-H-ENDPOINT-API")
	suite.Assertions.Equal("", pinnedHostname, "Correctly directed request was pinned to %s", pinnedHostname)

	// A request made to LON with the wrong hostname, but with city=LON in the query string, should be pinned
	req, err = http.NewRequest("GET", fmt.Sprintf("%s/throttle?city=LON", server.URL), nil)
	suite.Assertions.NoError(err)
	resp, err = client.Do(req)
	suite.Assertions.NoError(err)
	pinnedHostname = resp.Header.Get("X-H-ENDPOINT-API")
	suite.Assertions.Equal("api-driver-london.elasticride.com", pinnedHostname, "Incorrectly directed request is pinned to "+
		"wrong hostname")
	pinningTimestamp := resp.Header.Get("X-H-ENDPOINT-TIMESTAMP")
	suite.Assertions.Equal("10001", pinningTimestamp, "Wrong pinning timestamp")

	// Similarly; city=LON in the POST body, but the wrong hostname should be pinned
	post := url.Values{}
	post.Set("city", "LON")
	req, err = http.NewRequest("POST", fmt.Sprintf("%s/throttle", server.URL),
		bytes.NewBufferString(post.Encode()))
	suite.Assertions.NoError(err)
	resp, err = client.Do(req)
	suite.Assertions.NoError(err)
	pinnedHostname = resp.Header.Get("X-H-ENDPOINT-API")
	suite.Assertions.Equal("api-driver-london.elasticride.com", pinnedHostname, "Request made to offline region has wrong "+
		"pinned endpoint")
	pinningTimestamp = resp.Header.Get("X-H-ENDPOINT-TIMESTAMP")
	suite.Assertions.Equal("10001", pinningTimestamp, "Wrong pinning timestamp")

	// A request made to LON with city=LON in the query string, AND the right hostname, should not be pinned
	req, err = http.NewRequest("GET", fmt.Sprintf("%s/throttle?city=LON", server.URL), nil)
	req.Host = "api-driver-london.elasticride.com"
	suite.Assertions.NoError(err)
	resp, err = client.Do(req)
	suite.Assertions.NoError(err)
	pinnedHostname = resp.Header.Get("X-H-ENDPOINT-API")
	suite.Assertions.Equal("", pinnedHostname, "Correctly directed request was pinned to %s", pinnedHostname)
}

// Test a region outage; where us-east-1 is offline and set to fail-over to eu-west-1.
// Requests made to the NYC HOB should return a pinning response header to the eu-west-1 endpoint.
func (suite *HandlerSuite) TestRegionPinning_RegionOutageFailover() {
	configJson := `{
		"controlplane": {
			"rules": {
				"test-throttle": {
					"match": {
						"path": "/throttle",
						"proportion": 1.0
					},
					"action": 3
				}
			},
			"regions": {
				"eu-west-1": {
					"id": "eu-west-1",
					"status": "ONLINE",
					"failover": ["us-east-1"],
					"apps": {
						"default": {
							"api": "api-driver-london.elasticride.com"
						}
					}
				},
				"us-east-1": {
					"id": "us-east-1",
					"status": "OFFLINE",
					"failover": ["eu-west-1"],
					"apps": {
						"default": {
							"api": "api-driver-nyc.elasticride.com"
						}
					}
				}
			},
			"hobRegions": {
				"LON": "eu-west-1",
				"NYC": "us-east-1"
			}
		}
	}`
	buf := bytes.NewBufferString(configJson)
	origConfigBuf := bytes.NewBuffer(config.Raw())
	config.Load(buf)
	defer config.Load(origConfigBuf)
	time.Sleep(time.Second)

	server, client := suite.server, suite.client

	// A request made to NYC should fail over to eu-west-1
	// Hostname
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/throttle", server.URL), nil)
	suite.Assertions.NoError(err)
	req.Host = "api-driver-nyc.elasticride.com"
	resp, err := client.Do(req)
	suite.Assertions.NoError(err)
	pinnedHostname := resp.Header.Get("X-H-ENDPOINT-API")
	suite.Assertions.Equal("api-driver-london.elasticride.com", pinnedHostname, "Request made to offline region has wrong "+
		"pinned endpoint")

	// GET parameter
	req, err = http.NewRequest("GET", fmt.Sprintf("%s/throttle?city=NYC", server.URL), nil)
	suite.Assertions.NoError(err)
	resp, err = client.Do(req)
	suite.Assertions.NoError(err)
	pinnedHostname = resp.Header.Get("X-H-ENDPOINT-API")
	suite.Assertions.Equal("api-driver-london.elasticride.com", pinnedHostname, "Request made to offline region has wrong "+
		"pinned endpoint")

	// POST parameter
	post := url.Values{}
	post.Set("city", "NYC")
	req, err = http.NewRequest("POST", fmt.Sprintf("%s/throttle", server.URL),
		bytes.NewBufferString(post.Encode()))
	suite.Assertions.NoError(err)
	resp, err = client.Do(req)
	suite.Assertions.NoError(err)
	pinnedHostname = resp.Header.Get("X-H-ENDPOINT-API")
	suite.Assertions.Equal("api-driver-london.elasticride.com", pinnedHostname, "Request made to offline region has wrong "+
		"pinned endpoint")
}

func (suite *HandlerSuite) TestXHailoRoute() {
	configJson := `{
		"controlplane": {
			"configVersion": 10001,
			"rules": {
				"test-throttle": {
					"match": {
						"path": "/throttle",
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
	buf := bytes.NewBufferString(configJson)
	origConfigBuf := bytes.NewBuffer(config.Raw())
	defer config.Load(origConfigBuf)
	config.Load(buf)
	time.Sleep(time.Second)

	server, client := suite.server, suite.client

	// A request with no X-Hailo-Route, but with a matching throttle route
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/throttle", server.URL), nil)
	suite.Assertions.NoError(err)
	resp, err := client.Do(req)
	suite.Assertions.NoError(err)
	usedRoute := resp.Header.Get("X-Hailo-Route")
	suite.Assertions.Equal("Throttle", usedRoute)

	// Specified H2 route
	req, err = http.NewRequest("GET", fmt.Sprintf("%s/throttle", server.URL), nil)
	req.Header.Set("X-Hailo-Route", "H2")
	resp, err = client.Do(req)
	suite.Assertions.NoError(err)
	usedRoute = resp.Header.Get("X-Hailo-Route")
	suite.Assertions.Equal("H2", usedRoute)

	// Specified H1 route
	req, err = http.NewRequest("GET", fmt.Sprintf("%s/throttle", server.URL), nil)
	req.Header.Set("X-Hailo-Route", "H1")
	resp, err = client.Do(req)
	suite.Assertions.NoError(err)
	usedRoute = resp.Header.Get("X-Hailo-Route")
	suite.Assertions.Equal("H1", usedRoute)

	// Specified Deprecate route
	req, err = http.NewRequest("GET", fmt.Sprintf("%s/throttle", server.URL), nil)
	req.Header.Set("X-Hailo-Route", "Deprecate")
	resp, err = client.Do(req)
	suite.Assertions.NoError(err)
	usedRoute = resp.Header.Get("X-Hailo-Route")
	suite.Assertions.Equal("Deprecate", usedRoute)

	// Specified, but unknown (invalid), route
	req, err = http.NewRequest("GET", fmt.Sprintf("%s/throttle", server.URL), nil)
	req.Header.Set("X-Hailo-Route", "foobarbaz")
	resp, err = client.Do(req)
	suite.Assertions.NoError(err)
	usedRoute = resp.Header.Get("X-Hailo-Route")
	suite.Assertions.Equal("Throttle", usedRoute)
}
