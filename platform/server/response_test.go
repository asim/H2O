package server

import (
	"github.com/HailoOSS/platform/errors"
	"github.com/streadway/amqp"
	"testing"
)

type TestPayload struct{}

func (*TestPayload) Reset()         {}
func (*TestPayload) String() string { return "" }
func (*TestPayload) ProtoMessage()  {}

func TestReplyResponse(t *testing.T) {
	request := &Request{
		delivery: amqp.Delivery{
			ContentType: "application/octetstream",
		},
	}

	payload := &TestPayload{}
	response, err := ReplyResponse(request, payload)

	if err != nil {
		t.Fatalf("Failed to create reply response: %v", err)
	}

	if response.MessageType() != "reply" {
		t.Errorf(`Wrong message type "%v", expecting "reply"`, response.MessageType())
	}
}

func TestPongResponse(t *testing.T) {
	request := &Request{
		delivery: amqp.Delivery{
			ContentType: "application/octetstream",
		},
	}
	response := PongResponse(request)

	if string(response.payload) != "PONG" {
		t.Errorf("Invalid payload: %s", response.payload)
	}

	if response.MessageType() != "heartbeat" {
		t.Errorf(`Wrong message type "%v", expecting "heartbeat"`, response.MessageType())
	}
}

func TestErrorResponse(t *testing.T) {
	request := &Request{
		delivery: amqp.Delivery{
			ContentType: "application/octetstream",
		},
	}

	e := errors.Timeout("com.something", "Something timed out")
	response, err := ErrorResponse(request, e)

	if err != nil {
		t.Fatalf("Failed to create error response: %v", err)
	}

	if response.MessageType() != "error" {
		t.Errorf(`Wrong message type "%v", expecting "error"`, response.MessageType())
	}
}

func TestInvalidContentType(t *testing.T) {
	request := &Request{
		delivery: amqp.Delivery{
			ContentType: "a/b",
		},
	}

	payload := &TestPayload{}
	_, err := ReplyResponse(request, payload)

	if err == nil {
		t.Fatalf("Expected error due to invalid content type")
	}

	if err.Error() != "Unknown content type: a/b" {
		t.Errorf("Wrong error message: %v", err)
	}
}
