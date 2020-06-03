package client

import (
	"encoding/json"
	"fmt"

	"github.com/hailo-platform/H2O/protobuf/proto"
	"github.com/streadway/amqp"
)

// Response wraps an AMQP delivery
type Response struct {
	delivery amqp.Delivery
}

func newResponseFromDelivery(d amqp.Delivery) *Response {
	return &Response{d}
}

// ContentType returns the content type of the delivery
func (self *Response) ContentType() string {
	return self.delivery.ContentType
}

// String returns a string represenation of the response
func (self *Response) String() string {
	return fmt.Sprintf("%v", self.MessageID())
}

// MessageID returns the ID of the underlying message transport system, which
// is unique per delivery, and only really useful for debugging
func (self *Response) MessageID() string {
	return self.delivery.MessageId
}

// CorrelationID returns the MessageId of the original message
func (self *Response) CorrelationID() string {
	return self.delivery.CorrelationId
}

// IsError returns whether this message is an error?
func (self *Response) IsError() bool {
	if val, ok := self.delivery.Headers["messageType"]; ok {
		return val == "error"
	}

	return false
}

// Body of the message
func (self *Response) Body() []byte {
	return self.delivery.Body
}

// Header of the message
func (self *Response) Header() amqp.Table {
	return self.delivery.Headers
}

// Unmarshal the raw bytes payload of this request (into a protobuf)
func (self *Response) Unmarshal(into proto.Message) (err error) {
	if self == nil {
		err = fmt.Errorf("[Client] Cannot unmarshal response from nil Response")
		return
	}
	if into == nil {
		err = fmt.Errorf("[Client] Cannot unmarshal response into nil proto")
		return
	}
	switch self.delivery.ContentType {
	case "application/json":
		err = json.Unmarshal(self.Body(), into)
	case "application/octetstream":
		err = proto.Unmarshal(self.Body(), into)
	default:
		err = fmt.Errorf("Unknown content type: %s", self.delivery.ContentType)
	}

	return
}
