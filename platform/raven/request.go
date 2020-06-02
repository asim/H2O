package raven

// Request interface
type Request interface {
	ContentType() string
	Service() string
	Endpoint() string
	MessageID() string
	From() string
	RemoteAddr() string
	TraceID() string
	TraceShouldPersist() bool
	SessionID() string
	ParentMessageID() string
	Payload() []byte
	Authorised() bool
}
