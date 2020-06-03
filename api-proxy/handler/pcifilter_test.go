package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	platformtesting "github.com/hailo-platform/H2O/platform/testing"
	"github.com/hailo-platform/H2O/service/config"
)

const PCIFilterConfigJson = `{
	"pci": {
		"sensitivePaths": ["POST /v1/customer/card"],
		"encryptionProxyUrl": "https://encryption-proxy.com"
	}
}`

type PCIFilterSuite struct {
	platformtesting.Suite
	filter         *PCIFilter
	originalConfig *bytes.Buffer
	srv            *TestServer
}

func TestRunPCIFilterSuite(t *testing.T) {
	platformtesting.RunSuite(t, new(PCIFilterSuite))
}

func (s *PCIFilterSuite) SetupTest() {
	s.Suite.SetupTest()
	s.originalConfig = bytes.NewBuffer(config.Raw())
}

func (s *PCIFilterSuite) setConfigAndInitFilter(configStr string) {
	buf := bytes.NewBufferString(configStr)
	config.Load(buf)
	dummyHandler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {})
	s.srv, _ = SetupTestServerAndClient(s.T())
	s.filter = NewPCIFilter(dummyHandler, s.srv.HailoServer)
}

func (s *PCIFilterSuite) TearDownTest() {
	s.Suite.TearDownTest()
	config.Load(s.originalConfig)
	TeardownTestServer(s.T(), s.srv)
}

func (s *PCIFilterSuite) TestInsensitivePath() {
	s.setConfigAndInitFilter(PCIFilterConfigJson)
	req, err := http.NewRequest("POST", "/foo/bar", nil)
	s.NoError(err)
	s.False(s.filter.IsSensitiveRequest(req))
}

func (s *PCIFilterSuite) TestSensitivePathUnencryptedRequest() {
	s.setConfigAndInitFilter(PCIFilterConfigJson)
	req, err := http.NewRequest("POST", "/v1/customer/card", nil)
	s.NoError(err)
	s.True(s.filter.IsSensitiveRequest(req))
}

func (s *PCIFilterSuite) TestSensitivePathEncryptedRequest() {
	s.setConfigAndInitFilter(PCIFilterConfigJson)
	req, err := http.NewRequest("POST", "/v1/customer/card", nil)
	s.NoError(err)
	req.Header.Set("X-Encrypted", "true")
	s.False(s.filter.IsSensitiveRequest(req))
}

func (s *PCIFilterSuite) TestForward() {
	testServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("X-Foo", "bar")
		rw.WriteHeader(200)
		rw.Write([]byte("baz"))
	}))
	defer testServer.Close()
	s.T().Logf("Server listening on %s", testServer.URL)

	s.setConfigAndInitFilter(fmt.Sprintf(`{
		"pci": {
			"sensitivePaths": ["POST /v1/customer/card"],
			"encryptionProxyUrl": "%s"
		}
	}`, testServer.URL))

	req, err := http.NewRequest("POST", "/v1/customer/card", nil)
	s.NoError(err)

	rw := httptest.NewRecorder()
	s.filter.Forward(rw, req)
	s.Equal(200, rw.Code)
	s.Equal("bar", rw.HeaderMap.Get("X-Foo"))
	s.Equal("baz", rw.Body.String())
}
