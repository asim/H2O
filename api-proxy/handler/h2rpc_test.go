package handler

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/errors"
	platformtesting "github.com/hailo-platform/H2O/platform/testing"
	"github.com/hailo-platform/H2O/service/auth"
)

type testCaller struct {
	req *client.Request
	rsp *client.Response
	err errors.Error
}

func (c *testCaller) Call(req *client.Request, options ...client.Options) (*client.Response, errors.Error) {
	c.req = req
	if c.err != nil {
		return nil, c.err
	}
	return c.rsp, nil
}

func TestH2RPCHandlerSuite(t *testing.T) {
	suite := new(h2RPCHandlerSuite)
	suite.LogLevel = "off"
	platformtesting.RunSuite(t, suite)
}

type h2RPCHandlerSuite struct {
	platformtesting.Suite
	mockAuthScope *mockAuthScope
}

func (suite *h2RPCHandlerSuite) scopeConstructor() auth.Scope {
	return suite.mockAuthScope
}

func (suite *h2RPCHandlerSuite) SetupTest() {
	suite.Suite.SetupTest()

	suite.mockAuthScope = &mockAuthScope{
		ExpectedSessionId: sessionId,
		T:                 suite.T(),
	}
	authScopeConstructor = suite.scopeConstructor
}

func (suite *h2RPCHandlerSuite) TearDownTest() {
	suite.Suite.TearDownTest()

	authScopeConstructor = auth.New
}

func (suite *h2RPCHandlerSuite) TestRpcHandlerFormEncodedPostReturningError() {
	t := suite.T()

	existingCaller := rpcCaller
	defer func() { rpcCaller = existingCaller }()

	caller := &testCaller{
		err: errors.NotFound("foo.bar.notfound", "Thing not found"),
	}
	rpcCaller = caller.Call

	server, client := SetupTestServerAndClient(t)
	defer TeardownTestServer(t, server)

	values := make(url.Values)
	values.Set("service", "com.hailocab.service.foo")
	values.Set("endpoint", "bar")
	values.Set("request", "{\"baz\":\"bing\"}")
	body := values.Encode()
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/rpc", server.URL), bytes.NewReader([]byte(body)))
	assert.NoError(t, err)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Unexpected problem executing client request: %v", err)
	}

	// double check request

	assert.Equal(t, "com.hailocab.service.foo", caller.req.Service(), "Request has expected service name")
	assert.Equal(t, "bar", caller.req.Endpoint(), "Request has expected endpoint name")
	assert.Equal(t, "application/json", caller.req.ContentType(), "Request has expected content-type")

	// check response

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	contentType := resp.Header.Get("Content-Type")
	assert.Equal(t, "application/json; charset=utf-8", contentType, "Content-type of response should be application/json; charset=utf-8")

	expected := `{"status":false,"payload":"Thing not found","code":11,"dotted_code":"foo.bar.notfound","context":null}`
	if string(b) != expected {
		t.Errorf("Unexpected response body, expecting %v got %v", expected, string(b))
	}
	if resp.StatusCode != 404 {
		t.Errorf("Expecting HTTP status 404 for our NOT_FOUND error, got %v", resp.StatusCode)
	}
}

func TestRpcHandlerPOSTRequired(t *testing.T) {
	server, client := SetupTestServerAndClient(t)
	defer TeardownTestServer(t, server)

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/rpc", server.URL), nil)
	assert.NoError(t, err)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Unexpected problem executing client request: %v", err)
	}

	assert.Equal(t, 405, resp.StatusCode, "Expected Method Not Allowed response")
	assert.Equal(t, "POST", resp.Header.Get("Allow"), "Expected 'Allow: POST' header")
}
