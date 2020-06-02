package client

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/HailoOSS/protobuf/proto"
	"github.com/nu7hatch/gouuid"

	"github.com/HailoOSS/service/config"
)

// Request packages data needed to send a request out
type Request struct {
	contentType        string
	payload            []byte
	service            string
	endpoint           string
	messageID          string
	sessionID          string
	traceID            string
	traceShouldPersist bool
	parentMessageID    string
	fromService        string
	fromEndpoint       string
	remoteAddr         string
	options            Options
	authorised         bool
}

// ContentType returns the content type of the request
func (r *Request) ContentType() string {
	return r.contentType
}

// Payload returns the requests payload
func (r *Request) Payload() []byte {
	return r.payload
}

// Service returns the service of the request
func (r *Request) Service() string {
	return r.service
}

// Endpoint returns the endpoint of the request
func (r *Request) Endpoint() string {
	return r.endpoint
}

// MessageID returns the message ID of the request
func (r *Request) MessageID() string {
	return r.messageID
}

// From returns information about which service sent this request
// @todo return something that can be cryptographically verified
func (r *Request) From() string {
	return r.fromService
}

// From returns information about which endpoint sent this request
func (r *Request) FromEndpoint() string {
	return r.fromEndpoint
}

// RemoteAddr returns the remote IP Address of the request
func (r *Request) RemoteAddr() string {
	return r.remoteAddr
}

// SessionID returns the session ID of the request
func (r *Request) SessionID() string {
	return r.sessionID
}

// TraceID returns the trace ID of the request
func (r *Request) TraceID() string {
	return r.traceID
}

// TraceShouldPersist returns if the trace should be stored persistently
func (r *Request) TraceShouldPersist() bool {
	return r.traceShouldPersist
}

// ParentMessageID returns the message ID of a message (if any) that was received and triggered this message
// In other words, we use this to build up the call stack / hierarchy
func (r *Request) ParentMessageID() string {
	return r.parentMessageID
}

// Authorised returns whether the request has already been authorised
func (r *Request) Authorised() bool {
	return r.authorised
}

// SetFrom sets details about which service is making this request
// @todo eventually this should include an async cryptographic signature such that the receiver can verify this to establish trust
func (r *Request) SetFrom(service string) {
	r.fromService = service
}

// SetFromEndpoint sets details about which endpoint is making this request
func (r *Request) SetFromEndpoint(endpoint string) {
	r.fromEndpoint = endpoint
}

// SetRemoteAddr sets the address of the remote client making this request
func (r *Request) SetRemoteAddr(addr string) {
	r.remoteAddr = addr
}

// SetSessionID sets the session ID of the request
func (r *Request) SetSessionID(id string) {
	r.sessionID = id
}

// SetTraceID sets the trace ID of the request
func (r *Request) SetTraceID(id string) {
	r.traceID = id
}

// SetTraceShouldPersist sets whether the request trace should be stored persistently
func (r *Request) SetTraceShouldPersist(val bool) {
	r.traceShouldPersist = val
}

// SetParentMessageID sets the parent message ID, so we can build up the call stack
func (r *Request) SetParentMessageID(id string) {
	r.parentMessageID = id
}

// SetAuthorised sets whether the request has already been authorised
func (r *Request) SetAuthorised(val bool) {
	r.authorised = val
}

// shouldTrace determiens if we should trace this request, when sending
func (r *Request) shouldTrace() bool {
	if r.traceID != "" {
		return true
	}

	pcChance := config.AtPath("hailo", "service", "trace", "pcChance").AsFloat64(0)
	if pcChance <= 0 {
		return false
	}

	if rand.Float64() < pcChance {
		u4, err := uuid.NewV4()
		if err != nil {
			return false
		}

		r.SetTraceID(u4.String())
		return true
	}

	return false
}

func (r *Request) SetOptions(options Options) {
	r.options = options
}

func (r *Request) GetOptions() Options {
	return r.options
}

// Unmarshal the raw bytes payload of this request (into a protobuf)
func (r *Request) Unmarshal(into proto.Message) (err error) {
	if r == nil {
		err = fmt.Errorf("[Client] Cannot unmarshal request from nil Request")
		return
	}
	if into == nil {
		err = fmt.Errorf("[Client] Cannot unmarshal request into nil proto")
		return
	}
	switch r.contentType {
	case "application/json":
		err = json.Unmarshal(r.payload, into)
	case "application/octetstream":
		err = proto.Unmarshal(r.payload, into)
	default:
		err = fmt.Errorf("Unknown content type: %s", r.contentType)
	}

	return
}

func buildRequest(contentType string, payload []byte, service, endpoint string) (*Request, error) {
	if len(service) == 0 {
		return nil, fmt.Errorf("Missing service in request")
	}

	if len(endpoint) == 0 {
		return nil, fmt.Errorf("Missing endpoint in request")
	}

	messageID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("Failed to generate message ID: %v", err)
	}

	return &Request{
		contentType: contentType,
		payload:     payload,
		service:     service,
		endpoint:    endpoint,
		messageID:   messageID.String(),
	}, nil
}

// NewRequest builds a new request object, checking for bad data
func NewRequest(service, endpoint string, payload proto.Message) (*Request, error) {
	payloadData, err := proto.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return NewProtoRequest(
		service,
		endpoint,
		payloadData,
	)
}

// NewJsonRequest builds a new request object for JSON requests
func NewJsonRequest(service, endpoint string, payload []byte) (*Request, error) {
	return buildRequest(
		"application/json",
		payload,
		service,
		endpoint,
	)
}

// NewProtoRequest builds a new request object for raw protobuf requests
func NewProtoRequest(service, endpoint string, payload []byte) (*Request, error) {
	return buildRequest(
		"application/octetstream",
		payload,
		service,
		endpoint,
	)
}
