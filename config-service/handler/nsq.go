package handler

import (
	"encoding/json"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/service/nsq"
)

const broadcastTopic = "config.reload"
const platformTopicName = "platform.events"

type NSQEvent struct {
	Id        string
	Type      string
	Timestamp string
	Details   map[string]string
}

func broadcastChange(id string) {
	if err := nsq.Publish(broadcastTopic, []byte(id)); err != nil {
		log.Warnf("Failed to broadcast change via NSQ: %v", err)
	}
}

func pubNSQEvent(action, changeId, id, path, mech, user, message, config, previousConfig string) {
	event := changeToNSQ(action, changeId, id, path, mech, user, message, config, previousConfig)
	bytes, err := json.Marshal(event)
	if err != nil {
		log.Errorf("Error marshaling nsq event message for %v:%v", changeId, err)
		return
	}
	err = nsq.Publish(platformTopicName, bytes)
	if err != nil {
		log.Errorf("Error publishing message to NSQ: %v", err)
		return
	}
}
