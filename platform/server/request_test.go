package server

import (
	"testing"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"

	"github.com/hailo-platform/H2O/service/auth"
)

func TestNewRequestMints(t *testing.T) {
	delivery := amqp.Delivery{
		MessageId: "123456",
	}
	rsp := NewRequestFromDelivery(delivery)
	if rsp.MessageID() != "123456" {
		t.Error("Incorrect message ID")
	}
}

func TestHeartbeatBody(t *testing.T) {
	var req *Request

	heartbeat := amqp.Delivery{
		MessageId: "123456",
		Body:      []byte("PING"),
	}

	req = NewRequestFromDelivery(heartbeat)
	if !req.isHeartbeat() {
		t.Error("PING should be a heartbeat")
	}

	notHeartbeat := amqp.Delivery{
		MessageId: "123456",
		Body:      []byte("TEST"),
	}

	req = NewRequestFromDelivery(notHeartbeat)
	if req.isHeartbeat() {
		t.Error("TEST should not be a heartbeat")
	}
}

func TestHeartbeatHeader(t *testing.T) {
	d := amqp.Delivery{
		Headers: amqp.Table{
			"heartbeat": "ping",
		},
	}

	req := NewRequestFromDelivery(d)

	if !req.isHeartbeat() {
		t.Error("Heartbeat check failed with header")
	}
}

func TestHeartbeatMessageType(t *testing.T) {
	d := amqp.Delivery{
		Headers: amqp.Table{
			"messageType": "heartbeat",
		},
	}

	req := NewRequestFromDelivery(d)

	if !req.isHeartbeat() {
		t.Error("Heartbeat check failed with header")
	}
}

func TestPublication(t *testing.T) {
	d := amqp.Delivery{
		Headers: amqp.Table{
			"topic": "abc",
		},
	}

	req := NewRequestFromDelivery(d)

	if !req.IsPublication() {
		t.Error("Set topic, but not detected pub message")
	}
}

func TestTraceHeaders(t *testing.T) {
	d := amqp.Delivery{
		Headers: amqp.Table{
			"traceID":            "test-trace-id",
			"traceShouldPersist": "0",
			"authorised":         "1",
		},
	}
	req := NewRequestFromDelivery(d)
	assert.True(t, req.shouldTrace(), "shouldTrace() should return true with a traceID set")
	assert.Equal(t, "test-trace-id", req.TraceID(), "traceID mismatch")
	assert.False(t, req.TraceShouldPersist(), "TraceShouldPersist() should return false with AMQP header = 0")
	assert.True(t, req.Authorised(), "Authorised() should return true with AMQP header = 1")

	d = amqp.Delivery{
		Headers: amqp.Table{
			"traceID":            "test-trace-id",
			"traceShouldPersist": "1",
		}}
	req = NewRequestFromDelivery(d)
	assert.True(t, req.TraceShouldPersist(), "TraceShouldPersist() should return true with AMQP header = 1")
}

func TestRemoteAddrHeaders(t *testing.T) {
	d := amqp.Delivery{
		Headers: amqp.Table{
			"remoteAddr": "127.0.0.1",
		},
	}
	req := NewRequestFromDelivery(d)
	assert.Equal(t, "127.0.0.1", req.RemoteAddr(), "remoteAddr mismatch")
}

// Test that the information relating to a trace is passed from the originating request to the scoped request
func TestScopedRequestTracePassthrough(t *testing.T) {
	originatingDelivery := amqp.Delivery{
		Headers: amqp.Table{
			"traceID":            "test-trace-id",
			"traceShouldPersist": "1",
		},
	}
	originatingReq := NewRequestFromDelivery(originatingDelivery)

	scopedReq, err := originatingReq.ScopedRequest("com.hailocab.service.helloworld", "sayhello", &TestPayload{})
	assert.Nil(t, err, "error constructing scoped request: %v", err)
	assert.Equal(t, "test-trace-id", scopedReq.TraceID(), `TraceID() should return "test-trace-id"`)
	assert.True(t, scopedReq.TraceShouldPersist(), "TraceShouldPersist() should return true")
}

func TestScopedRequestAuthorised(t *testing.T) {
	req, err := ScopedRequest("com.hailocab.service.hellowork", "sayhello", &TestPayload{})
	assert.Nil(t, err, "error constructing scoped request: %v", err)
	assert.True(t, req.Authorised(), "S2S requests should be authorised")
}

func TestRequestAuthorisedPassthrough(t *testing.T) {
	originatingDelivery := amqp.Delivery{
		Headers: amqp.Table{
			"authorised": "1",
		},
	}
	originatingReq := NewRequestFromDelivery(originatingDelivery)
	assert.True(t, originatingReq.Auth().Authorised(), "scope of authorised request should be authorised")

	scopedReq, err := originatingReq.ScopedRequest("com.hailocab.service.helloworld", "sayhello", &TestPayload{})
	assert.Nil(t, err, "error constructing scoped request: %v", err)
	assert.True(t, scopedReq.Authorised(), "Scoped requests should inherit authorised-flag")
}

func TestMockedScope(t *testing.T) {
	d := amqp.Delivery{}
	/*
		Headers: amqp.Table{
			"traceID":            "test-trace-id",
			"traceShouldPersist": "0",
		},
	}*/
	req := NewRequestFromDelivery(d)

	s := &auth.MockScope{MockUid: "111", MockRoles: []string{"CUSTOMER"}}
	req.SetAuth(s)

	assert.False(t, req.Auth().HasAccess("ADMIN"))
	assert.True(t, req.Auth().HasAccess("CUSTOMER"))
	assert.Equal(t, "111", req.Auth().AuthUser().Id)
}
