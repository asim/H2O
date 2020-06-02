package raven

// Heartbeat interface
type Heartbeat interface {
	ID() string
	ContentType() string
	Payload() []byte
}
