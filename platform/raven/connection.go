package raven

import (
	log "github.com/cihub/seelog"
	"github.com/streadway/amqp"
)

// AMQPConnection wraps an AMQP connection with state
type AMQPConnection struct {
	conn *amqp.Connection
}

func (self *AMQPConnection) connect() (err error) {
	log.Tracef("[Raven] Connecting to %s", AmqpUri)

	if self.conn, err = amqp.Dial(AmqpUri); err == nil {
		log.Infof("[Raven] Online, connected to %s", AmqpUri)
	}

	return
}

// NotifyClose allows us to listen for the connection closing
func (self *AMQPConnection) NotifyClose() chan *amqp.Error {
	return self.conn.NotifyClose(make(chan *amqp.Error))
}

// Channel makes sure we are connected and gets a new channel
func (self *AMQPConnection) Channel() (*amqp.Channel, error) {
	return self.conn.Channel()
}

// Close allows us to close the connection
func (self *AMQPConnection) Close() error {
	return self.conn.Close()
}
