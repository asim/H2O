package handler

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	ptesting "github.com/HailoOSS/platform/testing"
)

func TestRealIP(t *testing.T) {
	suite.Run(t, new(RealIPTestSuite))
}

type RealIPTestSuite struct {
	ptesting.Suite
}

type dummyHandler struct{}

func (h *dummyHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	io.WriteString(rw, r.RemoteAddr)
}

func (r *RealIPTestSuite) TestXForwardedFor() {
	t := r.T()

	handler := &RealIPHandler{&dummyHandler{}}

	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	req.Header.Set("X-Forwarded-For", "20.20.20.20")
	req.RemoteAddr = "127.0.0.1:1337"
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)
	responseBody, err := ioutil.ReadAll(rw.Body)
	assert.NoError(t, err)
	assert.Equal(t, "20.20.20.20:1337", string(responseBody))

	req, err = http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	req.Header.Set("X-Forwarded-For", "20.20.20.20")
	req.Header.Set("X-Forwarded-Port", "80")
	req.RemoteAddr = "127.0.0.1:1337"
	rw = httptest.NewRecorder()
	handler.ServeHTTP(rw, req)
	responseBody, err = ioutil.ReadAll(rw.Body)
	assert.NoError(t, err)
	assert.Equal(t, "20.20.20.20:80", string(responseBody))
}
