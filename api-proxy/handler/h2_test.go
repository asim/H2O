package handler

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"

	"github.com/HailoOSS/protobuf/proto"

	api "github.com/HailoOSS/api-proxy/proto/api"
	"github.com/HailoOSS/platform/client"
	platformtesting "github.com/HailoOSS/platform/testing"
	"github.com/HailoOSS/service/config"
)

const H2ConfigJson = `{
    "controlplane": {
        "configVersion": 10001,
        "rules": {
            "test-h2": {
                "match": {
                    "path": "/h2-test",
                    "proportion": 1.0
                },
                "action": 2
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

type H2HandlerSuite struct {
	platformtesting.Suite
	origConfigBuf     *bytes.Buffer
	server            *TestServer
	apiClient         *http.Client
	mockServiceClient *client.MockClient
}

func TestRunH2HandlerSuite(t *testing.T) {
	suite := new(H2HandlerSuite)
	suite.LogLevel = "off"
	platformtesting.RunSuite(t, suite)
}

func (s *H2HandlerSuite) SetupTest() {
	s.Suite.SetupTest()
	s.mockServiceClient = &client.MockClient{}
	client.ActiveMockClient = s.mockServiceClient
	client.NewClient = client.NewMockClient
	client.DefaultClient = client.NewClient()

	buf := bytes.NewBufferString(H2ConfigJson)
	s.origConfigBuf = bytes.NewBuffer(config.Raw())
	config.Load(buf)

	s.server, s.apiClient = SetupTestServerAndClient(s.T())
}

func (s *H2HandlerSuite) TearDownTest() {
	s.Suite.TearDownTest()
	client.ActiveMockClient = nil
	client.NewClient = client.NewDefaultClient
	client.DefaultClient = client.NewClient()

	config.Load(s.origConfigBuf)

	TeardownTestServer(s.T(), s.server)
}

func (s *H2HandlerSuite) TestNormal() {
	request, err := http.NewRequest("POST", fmt.Sprintf("%s/h2-test/foo", s.server.URL), nil)
	s.NoError(err, "Request construction error")

	// Expect a service-to-service call
	expectedRequestPayload, err := httpRequestToProto(request)
	s.NoError(err)
	expectedRequest, err := client.NewRequest("com.HailoOSS.api.h2-test", "foo", expectedRequestPayload)
	s.NoError(err, "Error constructing expected service request")
	expectedResponse := &api.Response{
		StatusCode: proto.Int(200),
		Header: []string{
			"X-Kludged-By: @obeattie",
			"Location: http://google.com",
			"Set-Cookie: ABC=; path=/",
			"Set-Cookie: DEF=123; path=/",
		},
		Body: proto.String("foobar"),
	}
	s.mockServiceClient.AddRequestExpectation(expectedRequest, expectedResponse)

	response, err := s.apiClient.Do(request)
	s.NoError(err, "Request error")
	bodyBytes, err := ioutil.ReadAll(response.Body)
	s.NoError(err, "Error reading response body")
	body := string(bodyBytes)

	s.Equal(200, response.StatusCode)
	s.Equal("foobar", body)
	s.Equal("application/json; charset=utf-8", response.Header.Get("Content-Type"))
	s.Equal("@obeattie", response.Header.Get("X-Kludged-By"))
	s.Equal("http://google.com", response.Header.Get("Location"))
	s.Equal([]string{"ABC=; path=/", "DEF=123; path=/"}, response.Header["Set-Cookie"])
}

func (s *H2HandlerSuite) TestOverwriteContentType() {
	request, err := http.NewRequest("POST", fmt.Sprintf("%s/h2-test/foo", s.server.URL), nil)
	s.NoError(err, "Request construction error")

	// Expect a service-to-service call
	expectedRequestPayload, err := httpRequestToProto(request)
	s.NoError(err)
	expectedRequest, err := client.NewRequest("com.HailoOSS.api.h2-test", "foo", expectedRequestPayload)
	s.NoError(err, "Error constructing expected service request")
	var _status int32 = 200
	var _body string = "foobar"
	expectedResponse := &api.Response{
		StatusCode: &_status,
		Header: []string{
			"Content-type: text/xml",
		},
		Body: &_body,
	}
	s.mockServiceClient.AddRequestExpectation(expectedRequest, expectedResponse)

	response, err := s.apiClient.Do(request)
	s.NoError(err, "Request error")
	bodyBytes, err := ioutil.ReadAll(response.Body)
	s.NoError(err, "Error reading response body")
	body := string(bodyBytes)

	s.Equal(200, response.StatusCode)
	s.Equal("foobar", body)
	s.Equal("text/xml", response.Header.Get("Content-Type"))
}

func (s *H2HandlerSuite) TestPCIFilter() {
	// Set up fake encryption proxy
	mockProxy := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		r.Header.Set("X-Encrypted", "true")
		proxy := httputil.NewSingleHostReverseProxy(s.server.URL)
		proxy.ServeHTTP(rw, r)
	}))
	defer mockProxy.Close()

	config.Load(bytes.NewBufferString(fmt.Sprintf(`{
		"pci": {
			"sensitivePaths": ["/v1/customer/card"],
			"encryptionProxyUrl": "%s"
		}
	}`, mockProxy.URL)))

	// Construct request
	request, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/customer/card", s.server.URL), nil)
	s.NoError(err, "Request construction error")

	// Expect a service-to-service call
	expectedRequestPayload, err := httpRequestToProto(request)
	s.NoError(err)
	expectedRequest, err := client.NewRequest("com.HailoOSS.api.v1.customer", "card", expectedRequestPayload)
	s.NoError(err, "Error constructing expected service request")
	expectedResponse := &api.Response{
		StatusCode: proto.Int(200),
		Body:       proto.String("foobar"),
	}
	s.mockServiceClient.AddRequestExpectation(expectedRequest, expectedResponse)

	// Perform request and assert
	response, err := http.DefaultClient.Do(request)
	s.NoError(err)
	bodyBytes, err := ioutil.ReadAll(response.Body)
	s.NoError(err, "Error reading response body")
	body := string(bodyBytes)
	s.Equal(200, response.StatusCode)
	s.Equal("foobar", body)
}
