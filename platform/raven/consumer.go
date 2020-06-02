package raven

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/streadway/amqp"
)

// BindService is used when self-binding
func BindService(serviceName, queue string) error {
	log.Tracef("[Raven] Self-binding %v to %v", serviceName, queue)

	if !Connected {
		return fmt.Errorf("[Raven] Error self-binding, raven not connected")
	}

	if err := Consumer.channel.QueueBind(
		queue,       // name of the queue
		serviceName, // bindingKey
		EXCHANGE,    // sourceExchange
		true,        // noWait
		amqp.Table{
			"service": serviceName,
			"x-match": "all",
		}, // arguments
	); err != nil {
		return fmt.Errorf("[Raven] Queue bind failed for \"%s\": %v", queue, err)
	}

	return nil
}

// Consume data from a queue
func Consume(queue string) (deliveries <-chan amqp.Delivery, err error) {
	log.Tracef("[Raven] Attempting to consume from %s", queue)

	if !Connected {
		return nil, fmt.Errorf("[Raven] Error consuming, raven not connected")
	}
	if _, err = Consumer.channel.QueueDeclare(
		queue, // name of the queue
		false, // durable
		true,  // delete when usused
		false, // exclusive
		true,  // noWait
		amqp.Table{"x-message-ttl": int32(5000), "x-expires": int32(30000)}, // arguments
	); err != nil {
		err = fmt.Errorf("[Raven] Queue declare failed \"%s\": %v", queue, err)
		return
	}

	if deliveries, err = Consumer.channel.Consume(
		queue, // queue name
		queue, // consumer tag
		true,  // auto ack
		false, // exclusive
		false, // no local
		true,  // no wait
		nil,   // args
	); err != nil {
		err = fmt.Errorf("[Raven] Failed to consume from queue \"%s\": %v", queue, err)
		return
	}

	// binding should come after consume because temp queues are only deleted once they have had at least once consumer
	if err = Consumer.channel.QueueBind(
		queue,          // name of the queue
		queue,          // bindingKey
		REPLY_EXCHANGE, // sourceExchange
		false,          // noWait
		nil,            // arguments
	); err != nil {
		err = fmt.Errorf("[Raven] Queue bind failed for \"%s\": %v", queue, err)
		return
	}

	log.Tracef("[Raven] Consuming from queue \"%s\"", queue)
	return
}

// ConsumerNotifyClose returns a go channel to notify when the amqp channel closes
func ConsumerNotifyClose() chan *amqp.Error {
	return Consumer.NotifyClose()
}
