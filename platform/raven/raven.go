package raven

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	log "github.com/cihub/seelog"
	"github.com/streadway/amqp"
)

var (
	// EXCHANGE is the default exchange
	EXCHANGE = "h2o"
	// REPLY_EXCHANGE is the exchange where replies go to
	REPLY_EXCHANGE = "h2o.direct"
	// TOPIC_EXCHANGE is the topic exchange
	TOPIC_EXCHANGE = "h2o.topic"
	// HOSTNAME is the hostname of rabbitmq
	HOSTNAME = "localhost"
	// USERNAME is the username to connect to rabbitmq with
	USERNAME = "hailo"
	// PASSWORD is the password to connect to rabbitmq with
	PASSWORD = "hailo"
	// PORT is the rabbitmq port
	PORT = flag.Int("amqp-port", 5672, "<port> to connect to for AMQP")
	// ADMINPORT is the port of the admin interface
	ADMINPORT = flag.Int("amqp-admin-port", 15672, "<port> to connect to for AMQP")
)

var (
	// AmqpUri is the URI to connect to rabbit, made up of username, host etc
	AmqpUri string
	// Connection wraps a standard AMQP connection
	Connection *AMQPConnection
	// Publisher represents the channel we send messages on
	Publisher *AMQPChannel
	// Consumer represents the channel we consume message on
	Consumer  *AMQPChannel
	Connected bool

	quitChan chan struct{}
)

func init() {
	// get boxen AMQP config from environment
	if boxenAMQP := os.Getenv("BOXEN_RABBITMQ_URL"); boxenAMQP != "" {
		AmqpUri = boxenAMQP

		if config, err := amqp.ParseURI(boxenAMQP); err == nil {
			fmt.Printf("Using custom boxen config: %v", boxenAMQP)
			fmt.Println()
			HOSTNAME = config.Host
			USERNAME = config.Username
			PASSWORD = config.Password
			PORT = &config.Port
		}
	} else {
		AmqpUri = fmt.Sprintf("amqp://%s:%s@%s:%s/", USERNAME, PASSWORD, HOSTNAME, strconv.Itoa(*PORT))
	}

	if mgmtPort := os.Getenv("BOXEN_RABBITMQ_MGMT_PORT"); mgmtPort != "" {
		if p, err := strconv.Atoi(mgmtPort); err == nil {
			ADMINPORT = &p
		}
	}

	// connect AMQP
	Connection = &AMQPConnection{}

	// connect Send channel
	Publisher = &AMQPChannel{
		name: "publisher",
		conn: Connection,
	}

	// connect Consume channel
	Consumer = &AMQPChannel{
		name: "consumer",
		conn: Connection,
	}
}

// Connect to AMQP + channels
func Connect() chan bool {
	quitChan = make(chan struct{})

	ch := make(chan bool)
	go keepalive(ch)
	return ch
}

func Disconnect() {
	quitChan <- struct{}{}
}

func keepalive(ch chan bool) {
	var (
		conn  chan *amqp.Error
		pubCh chan *amqp.Error
		conCh chan *amqp.Error
	)

	for {
		// Attempt to connect if not already
		if !Connected {
			if err := connect(); err != nil {
				log.Criticalf("[Raven] Failed to reconnect: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Connection succesful
			Connected = true

			// Attempt to handle the various connection errors
			conn = Connection.NotifyClose()
			pubCh = Publisher.NotifyClose()
			conCh = Consumer.NotifyClose()

			// Notify our listeners that we are good to go
			select {
			case ch <- true:
			case <-time.After(1 * time.Second):
			}
		}

		select {
		case <-quitChan:
			log.Criticalf("[Raven] Manually disconnected")
			ch <- false
			return
		case err := <-conn:
			log.Criticalf("[Raven] Disconnected: %v, recoverable: %v", err, err.Recover)
			Connected = false
			// Notify our listeners
			select {
			case ch <- false:
			case <-time.After(1 * time.Second):
			}
			// brute force for now until we work out why deliveries channel on consumer isn't being closed
			die(err)

		case err, ok := <-conCh:
			if ok {
				log.Warnf("[Raven] Consumer channel closed: %v", err)
			}
			log.Debugf("[Raven] Re-establishing consumer channel...")
			// Hard close
			if err := Consumer.Close(); err != nil {
				log.Warnf("[Raven] Unable to forcefully close the consumer chan: %v", err)
			}
			// Attempt to reconnect if possible
			if err := Consumer.connect(); err != nil {
				die(err)
			}
			// We are good again
			conCh = Consumer.NotifyClose()

		case err, ok := <-pubCh:
			if ok {
				log.Warnf("[Raven] Publisher channel closed: %v", err)
			}
			log.Debugf("[Raven] Re-establishing publisher channel...")
			// Hard close
			if err := Publisher.Close(); err != nil {
				log.Warnf("[Raven] Unable to forcefully close the publisher chan: %v", err)
			}
			// Attempt to reconnect if possible
			if err := Publisher.connect(); err != nil {
				die(err)
			}
			// We are good again
			pubCh = Publisher.NotifyClose()
		}

	}

}

// die would bruteforcefully kill the binary
func die(err error) {
	log.Criticalf("[Raven] Terminating due to connection error: %v", err)
	log.Flush()
	os.Exit(8)
}

func connect() error {
	if err := Connection.connect(); err != nil {
		return err
	}

	if err := Publisher.connect(); err != nil {
		Connection.Close()
		return err
	}

	if err := Consumer.connect(); err != nil {
		Publisher.Close()
		Connection.Close()
		return err
	}

	return nil
}

// IsConnected returns the status of the raven connection
// Todo: we should probably write a proper conn manager with
// locking and all.
func IsConnected() bool {
	return Connected
}
