package event

import (
	"encoding/json"

	log "github.com/cihub/seelog"
	nsq "github.com/HailoOSS/service/nsq"
)

const (
	userEventTopicName = "login.userevent"
)

type UserEvent struct {
	Action                string   `json:"action,omitempty"`
	FirstName             string   `json:"firstName,omitempty"`
	LastName              string   `json:"lastName,omitempty"`
	AlternateNames        []string `json:"alternateNames,omitempty"`
	CreatedAt             string   `json:"createdAt,omitempty"`
	FullName              string   `json:"fullName,omitempty"`
	Roles                 []string `json:"roles,omitempty"`
	Username              string   `json:"username,omitempty"`
	AccountExpirationDate string   `json:"accountExpirationDate,omitempty"`
	Status                string   `json:"status,omitempty"`
	LastLoginAt           string   `json:"lastLoginAt,omitempty"`
}

func NewUserCreateEvent(e *UserEvent) *UserEvent {
	e.Action = "createuser"
	return e
}

func NewUserUpdateEvent(e *UserEvent) *UserEvent {
	e.Action = "updateuser"
	return e
}

func NewUserDeleteEvent(e *UserEvent) *UserEvent {
	e.Action = "deleteuser"
	return e
}

func (e *UserEvent) Publish() {
	bytes, err := json.Marshal(e)

	if err != nil {
		log.Errorf("[UserEvent] Cannot marshal event %v - Error: %v", e, err)
		return
	}

	if err := nsq.Publish(userEventTopicName, bytes); err != nil {
		log.Errorf("[UserEvent] Unable to publish to nsq: %v", err)
	}
}
