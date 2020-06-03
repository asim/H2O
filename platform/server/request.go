package server

import (
	json "encoding/json"
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/protobuf/proto"
	"github.com/streadway/amqp"

	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/service/auth"
)

// Request just wraps an AMQP delivery
type Request struct {
	delivery        amqp.Delivery
	scope           auth.Scope
	unmarshaledData proto.Message
}

// NewRequestFromDelivery creates the Request object based on an AMQP delivery object
func NewRequestFromDelivery(d amqp.Delivery) *Request {
	return &Request{delivery: d}
}

func NewRequestFromProto(req proto.Message) *Request {
	protoBytes := make([]byte, 0)
	if req != nil {
		protoBytes, _ = proto.Marshal(req)
	}

	result := NewRequestFromDelivery(amqp.Delivery{
		Body:        protoBytes,
		ContentType: "application/octetstream",
		Headers:     amqp.Table(make(map[string]interface{})),
	})
	result.unmarshaledData = req
	return result
}

func (self *Request) String() string {
	return fmt.Sprintf("%v %v", self.MessageID(), self.Destination())
}

// MessageID returns the ID of the underlying message transport system, which
// is unique per delivery, and only really useful for debugging
func (self *Request) MessageID() string {
	return self.delivery.MessageId
}

// Destination returns the name of the server and endpoint that the request was directed
// at, for example: com.hailo-platform/H2O.service.customer.retrieve
func (self *Request) Destination() string {
	return fmt.Sprintf("%s.%s", self.Service(), self.Endpoint())
}

func (self *Request) getHeader(key string) string {
	if self == nil {
		log.Error("[Server] Cannot extract header - no delivery to use (self is nil)")
		return ""
	}
	if value, ok := self.delivery.Headers[key]; ok {
		if v, ok := value.(string); ok {
			return v
		}
		log.Warnf("[Server] Failed to cast AMQP header to string: %#v", value)
	}

	return ""
}

// Endpoint returns the name of the endpoint part of the destination
func (self *Request) Endpoint() string {
	return self.getHeader("endpoint")
}

// Service returns the name of the service part of the destination
func (self *Request) Service() string {
	return self.getHeader("service")
}

// Context returns some string that is designed for us to base error messages from
func (self *Request) Context() string {
	return fmt.Sprintf("%s.%s", self.getHeader("service"), self.getHeader("endpoint"))
}

// Topic returns the topic that a Pub/Sub request goes to
func (self *Request) Topic() string {
	return self.getHeader("topic")
}

// From returns information about which service sent this message
// @todo make this such that we can cryptographically verify this
func (self *Request) From() string {
	return self.getHeader("from")
}

// SessionID returns the security context session ID, if there is one
func (self *Request) SessionID() string {
	return self.getHeader("sessionID")
}

// SetSessionID allows changing of the session ID if it changes during a request
func (self *Request) SetSessionID(sessionID string) *Request {
	self.delivery.Headers["sessionID"] = sessionID
	return self
}

// TraceID returns the trace ID, if there is one
func (self *Request) TraceID() string {
	return self.getHeader("traceID")
}

// TraceShouldPersist returns if the trace should be stored persistently
func (self *Request) TraceShouldPersist() bool {
	return self.getHeader("traceShouldPersist") == "1"
}

// ParentMessageID returns the message ID of a message (if any) that was received and triggered this message
// In other words, we use this to build up the call stack / hierarchy
func (self *Request) ParentMessageID() string {
	return self.getHeader("parentMessageID")
}

// Authorised returns the whether the request has already been authorised
func (self *Request) Authorised() bool {
	return self.getHeader("authorised") == "1"
}

// shouldTrace determiens if we should trace this request, when handling
func (self *Request) shouldTrace() bool {
	return self.TraceID() != ""
}

// IsPublication returns whether this is a publication, i.e. has a topic
func (self *Request) IsPublication() bool {
	return len(self.Topic()) > 0
}

func (self *Request) MessageType() string {
	return self.getHeader("messageType")
}

// RemoteAddr returns the IP Address of the client, this may not be set if the
// request did not originate from the thinapi.
func (self *Request) RemoteAddr() string {
	return self.getHeader("remoteAddr")
}

func (self *Request) Payload() []byte {
	return self.delivery.Body
}

// Unmarshal the raw bytes payload of this request (into a protobuf)
func (self *Request) Unmarshal(into proto.Message) (err error) {
	switch self.delivery.ContentType {
	case "application/json":
		err = json.Unmarshal(self.delivery.Body, into)
	case "application/octetstream":
		err = proto.Unmarshal(self.delivery.Body, into)
	default:
		err = fmt.Errorf("Unknown content type: %s", self.delivery.ContentType)
	}

	return
}

// Auth returns a fully-initialised authentication scope, from which you
// can determine if anyone is authenticated and who they are
func (self *Request) Auth() auth.Scope {
	if self.scope == nil {
		// init
		self.scope = auth.New()
		self.scope.RpcScope(defaultScoper)
		if s := self.SessionID(); s != "" {
			if err := self.scope.RecoverSession(s); err != nil {
				log.Warnf("[Server] Session recovery failure: %v", err)
			}
		}
		if s := self.From(); s != "" {
			self.scope.RecoverService(self.Endpoint(), s)
		}

		self.scope.SetAuthorised(self.Authorised())
	}

	return self.scope
}

// SetAuth is useful for mocking/testing
func (self *Request) SetAuth(s auth.Scope) {
	self.scope = s
}

// ScopedRequest returns a client request, prepared with any scoping information from _this_ inbound server request (in
// other words we are forwarding all the scope information such as trace ID, session ID etc.. with our new client
// request). This includes the scope of the service _making_ the call.
func (self *Request) ScopedRequest(service, endpoint string, payload proto.Message) (*client.Request, error) {
	if self == nil {
		return nil, fmt.Errorf("Cannot build scoped request from nil Request")
	}
	r, err := client.NewRequest(service, endpoint, payload)
	if err != nil {
		return nil, err
	}
	// load in scope
	if self.SessionID() != "" {
		r.SetSessionID(self.SessionID())
	} else {
		// double check Auth() scope
		if self.Auth().IsAuth() {
			r.SetSessionID(self.Auth().AuthUser().SessId)
		}
	}
	r.SetTraceID(self.TraceID())
	r.SetTraceShouldPersist(self.TraceShouldPersist())
	r.SetParentMessageID(self.MessageID())
	r.SetRemoteAddr(self.RemoteAddr())

	// scope -- who WE are (not who sent it to us)
	r.SetFrom(Name)
	r.SetFromEndpoint(self.Endpoint())

	// set whether the request has already been authorised
	r.SetAuthorised(self.Auth().Authorised())

	return r, nil
}

// ScopedJsonRequest does just the same as ScopedRequest but with JSON payload
func (self *Request) ScopedJsonRequest(service, endpoint string, payload []byte) (*client.Request, error) {
	r, err := client.NewJsonRequest(service, endpoint, payload)
	if err != nil {
		return nil, err
	}
	// load in scope
	if self.SessionID() != "" {
		r.SetSessionID(self.SessionID())
	} else {
		// double check Auth() scope
		if self.Auth().IsAuth() {
			r.SetSessionID(self.Auth().AuthUser().SessId)
		}
	}
	r.SetTraceID(self.TraceID())
	r.SetParentMessageID(self.MessageID())
	r.SetRemoteAddr(self.RemoteAddr())

	// scope -- who WE are (not who sent it to us)
	r.SetFrom(Name)
	r.SetFromEndpoint(self.Endpoint())

	return r, nil
}

// check if inbound request is a heartbeat
func (self *Request) isHeartbeat() bool {
	// Check message type
	if self.MessageType() == "heartbeat" {
		return true
	}

	// Check header
	if self.getHeader("heartbeat") == "ping" {
		return true
	}

	// fallback to checking body
	return string(self.delivery.Body) == "PING"
}

// ReplyTo gets the AMQP channel that we should be replying to
func (self *Request) ReplyTo() string {
	return self.delivery.ReplyTo
}

// Data returns the unmarshalled inbound payload
func (self *Request) Data() proto.Message {
	if self.unmarshaledData == nil {
		panic("Data() cannot be called without a registered request protocol")
	}
	return self.unmarshaledData
}

// Sets what the expected unmarshaled data should be
func (self *Request) SetData(data proto.Message) {
	self.unmarshaledData = data
}
