package raven

// Publication interface
type Publication interface {
	ContentType() string
	Topic() string
	MessageID() string
	Payload() []byte
	SessionID() string
}
