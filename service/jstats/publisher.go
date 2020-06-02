package jstats

import (
	"strconv"
	"time"
)

var defaultInstance Publisher = newDefaultPublisher()

type Publisher interface {
	Publish(event map[string]string) error
}

func Publish(event map[string]string) error {
	return defaultInstance.Publish(event)
}

// PublishEvent will make sure to set the eventtype and sets
// the timestamp to the current time
func PublishEvent(eventType string, payload map[string]string) error {
	emap := make(map[string]string)
	// Copy instead of modifying the map passed in
	for k, v := range payload {
		emap[k] = v
	}
	emap[keyEventType] = eventType
	emap[keyTimestamp] = strconv.Itoa(int(time.Now().Unix()))
	return defaultInstance.Publish(emap)
}
