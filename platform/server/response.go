package server

import (
	json "encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/protobuf/proto"
	"github.com/streadway/amqp"
)

// Response wraps an AMQP delivery, with a payload & message type
type Response struct {
	messageType string
	payload     []byte
	delivery    amqp.Delivery
}

// ContentType returns the content type of the delivery
func (self *Response) ContentType() string {
	return self.delivery.ContentType
}

// MessageType returns the message type of the delivery (error, heartbeat etc)
func (self *Response) MessageType() string {
	return self.messageType
}

// Payload returns the payload data
func (self *Response) Payload() []byte {
	return self.payload
}

// ReplyTo returns the queue we should send the response to
func (self *Response) ReplyTo() string {
	return self.delivery.ReplyTo
}

// MessageID returns the message ID of the delivery
func (self *Response) MessageID() string {
	return self.delivery.MessageId
}

// PongResponse sends a PONG message
func PongResponse(replyTo *Request) *Response {
	return &Response{
		messageType: "heartbeat",
		payload:     []byte("PONG"),
		delivery:    replyTo.delivery,
	}
}

// ErrorResponse wraps a normal response with a messageType of "error"
func ErrorResponse(replyTo *Request, err errors.Error) (*Response, error) {
	return response(replyTo, errors.ToProtobuf(err), "error")
}

// ReplyResponse sends a normal response
func ReplyResponse(replyTo *Request, payload proto.Message) (*Response, error) {
	return response(replyTo, payload, "reply")
}

func response(replyTo *Request, payload proto.Message, messageType string) (rsp *Response, err error) {
	rsp = &Response{
		messageType: messageType,
		delivery:    replyTo.delivery,
	}

	switch replyTo.delivery.ContentType {
	case "application/json":
		rsp.payload, err = json.Marshal(payload)
	case "application/octetstream":
		rsp.payload, err = proto.Marshal(payload)
	default:
		err = fmt.Errorf("Unknown content type: %s", replyTo.delivery.ContentType)
	}

	if err != nil {
		rsp = nil
		log.Criticalf("[Server] Failed to marshal payload: %v", err)
	}

	return
}
