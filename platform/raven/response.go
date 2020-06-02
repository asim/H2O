package raven

// Response interface
type Response interface {
	ContentType() string
	MessageType() string
	Payload() []byte
	ReplyTo() string
	MessageID() string
}
