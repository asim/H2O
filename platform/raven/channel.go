package raven

import (
	log "github.com/cihub/seelog"
	"github.com/streadway/amqp"
	"sync"
)

// AMQPChannel wraps an AMQP channel with state
type AMQPChannel struct {
	sync.RWMutex
	name    string
	conn    *AMQPConnection
	channel *amqp.Channel
}

func (self *AMQPChannel) connect() (err error) {
	self.Lock()
	defer self.Unlock()

	// get the channel
	if self.channel, err = self.conn.Channel(); err == nil {
		log.Debugf("[Raven] Channel \"%s\" connected", self.name)
	}

	return
}

// NotifyClose allows us to listen for the channel closing
func (self *AMQPChannel) NotifyClose() chan *amqp.Error {
	return self.channel.NotifyClose(make(chan *amqp.Error))
}

// Close allows us to close the channel
func (self *AMQPChannel) Close() error {
	return self.channel.Close()
}
