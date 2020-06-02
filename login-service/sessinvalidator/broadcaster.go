package sessinvalidator

import (
	log "github.com/cihub/seelog"
	"github.com/HailoOSS/service/nsq"
)

// BroadcastSessionExpiry sends a message to NSQ saying that a session is now invalid
func BroadcastSessionExpiry(sessId string) {
	// try to publish invalidation
	if err := nsq.Publish(TopicName, []byte(sessId)); err != nil {
		log.Warnf("Failed to PUB session invalidation broadcast message to NSQ: %v", err)
	}
}
