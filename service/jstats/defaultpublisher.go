package jstats

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/HailoOSS/service/nsq"
)

const (
	jstatsTopic = "event"

	keyEventType = "eventType"
	keyTimestamp = "timestamp"
)

type defaultPublisher struct {
}

func newDefaultPublisher() *defaultPublisher {
	return &defaultPublisher{}

}

func (this *defaultPublisher) Publish(event map[string]string) error {
	// validation
	if event[keyEventType] == "" {
		return errors.New("Jstats event missing eventType")
	}
	ts := event[keyTimestamp]
	if ts == "" {
		return errors.New("Jstats event missing timestamp")
	}
	if _, err := strconv.ParseInt(ts, 10, 64); err != nil {
		return errors.New("Jstats event timestamp is wrong format")
	}

	bytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("Error marshalling event %s", err)
	}
	if err := nsq.Publish(jstatsTopic, bytes); err != nil {
		return fmt.Errorf("Error sending jstats event %s", err)
	}
	return nil
}
