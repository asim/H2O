package client

import (
	"fmt"

	"github.com/hailo-platform/H2O/protobuf/proto"
	"github.com/nu7hatch/gouuid"
)

// Publication packages data needed to send a publication out
type Publication struct {
	contentType string
	payload     []byte
	topic       string
	messageID   string
	sessionID   string
}

// Payload returns the payload of the publication
func (self *Publication) Payload() []byte {
	return self.payload
}

// Topic returns the topic of the publication
func (self *Publication) Topic() string {
	return self.topic
}

// MessageID returns the message ID of the publication
func (self *Publication) MessageID() string {
	return self.messageID
}

// ContentType return the content type of the publication
func (self *Publication) ContentType() string {
	return self.contentType
}

// SessionID returns the session ID of the publication
func (self *Publication) SessionID() string {
	return self.sessionID
}

// SetSessionID sets the session ID of the publication
func (self *Publication) SetSessionID(id string) {
	self.sessionID = id
}

func buildPublication(contentType string, topic string, payload []byte) (*Publication, error) {
	if len(topic) == 0 {
		return nil, fmt.Errorf("Missing topic in publication")
	}
	messageID, _ := uuid.NewV4()

	return &Publication{
		contentType: contentType,
		payload:     payload,
		topic:       topic,
		messageID:   messageID.String(),
	}, nil
}

// NewPublication returns a new publication based on topic and payload
func NewPublication(topic string, payload proto.Message) (*Publication, error) {
	payloadData, err := proto.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return buildPublication("application/octetstream", topic, payloadData)
}

// NewJsonPublication returns a new publication based on topic and json payload
func NewJsonPublication(topic string, payload []byte) (*Publication, error) {
	return buildPublication("application/json", topic, payload)
}
