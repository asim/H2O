package client

import (
	"bytes"
	"testing"

	"github.com/hailo-platform/H2O/service/config"
	"github.com/stretchr/testify/assert"
)

type TestPayload struct{}

func (*TestPayload) Reset()         {}
func (*TestPayload) String() string { return "" }
func (*TestPayload) ProtoMessage()  {}

func TestNewRequest(t *testing.T) {
	testService := "com.hailo-platform/H2O.service.helloworld"
	testEndpoint := "sayhello"

	payload := &TestPayload{}
	req, err := NewRequest(testService, testEndpoint, payload)

	if err != nil {
		t.Fatalf("Error not expected: %v", err)
	}

	if req.Service() != testService {
		t.Errorf(`Wrong service, expecting "%s" got "%s"`, testService, req.Service())
	}

	if req.Endpoint() != testEndpoint {
		t.Errorf(`Wrong endpoint, expecting "%s" got "%s"`, testEndpoint, req.Endpoint())
	}

	if req.MessageID() == "" {
		t.Error("Message ID blank, should have been minted with a nice UUID")
	}

	if req.ContentType() != "application/octetstream" {
		t.Errorf(`Wrong content type, expecting "application/octetstream" got "%s"`, req.ContentType())
	}
}

func TestNewRequestErrorOnEmptyService(t *testing.T) {
	payload := &TestPayload{}
	req, err := NewRequest("", "", payload)

	if err == nil {
		t.Error("Missing error")
	}

	if req != nil {
		t.Errorf("Request found: %+v", req)
	}
}

func TestNewJsonRequest(t *testing.T) {
	payload := []byte(`{"hello":"world"}`)
	req, _ := NewJsonRequest("abc", "def", payload)

	if req.ContentType() != "application/json" {
		t.Errorf(`Wrong content type, expecting "application/json" got "%s"`, req.ContentType())
	}
}

func TestShouldTrace(t *testing.T) {
	testService := "com.hailo-platform/H2O.service.helloworld"
	testEndpoint := "sayhello"
	payload := &TestPayload{}

	// Without a traceID and with a 0 pcChance, we shouldn't trace
	buf := bytes.NewBufferString(`{"hailo":{"service":{"trace":{"pcChance":0.0}}}}`)
	config.Load(buf)
	req, _ := NewRequest(testService, testEndpoint, payload)
	req.SetTraceID("")
	assert.False(t, req.shouldTrace(), `shouldTrace() should return false with traceID="" and pcChance=0`)

	// With a traceID present, we should always trace
	req, _ = NewRequest(testService, testEndpoint, payload)
	req.SetTraceID("test-trace-id")
	assert.True(t, req.shouldTrace(), "shouldTrace() should return true with a traceID set")

	// Without a traceID, and with a 1 pcChance (ie. always), we should trace
	buf = bytes.NewBufferString(`{"hailo":{"service":{"trace":{"pcChance":1.0}}}}`)
	config.Load(buf)
	req, _ = NewRequest(testService, testEndpoint, payload)
	req.SetTraceID("")
	assert.True(t, req.shouldTrace(), `shouldTrace() should return true with traceId="" and pcChance=1`)
}
