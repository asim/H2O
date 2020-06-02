package raven

import (
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/streadway/amqp"
)

const (
	deliveryMode      = amqp.Transient
	defaultPriority   = 0 // 0-9
	heartbeatPriority = 5 // 0-9
	contentEncoding   = ""
)

// SendResponse back via AMQP
func SendResponse(rsp Response, InstanceID string) error {
	messageName := rsp.MessageID()
	if len(messageName) == 0 {
		messageName = rsp.MessageType()
	}

	log.Tracef("[Raven] Sending back response for %s to routing key %s", messageName, rsp.ReplyTo())

	if !Connected {
		return fmt.Errorf("[Raven] Error sending response, raven not connected")
	}

	err := Publisher.channel.Publish(
		REPLY_EXCHANGE, // publish to default exchange for reply-to
		rsp.ReplyTo(),  // replyto becomes our routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			Headers: amqp.Table{
				"messageType": rsp.MessageType(),
			},
			ContentType:     rsp.ContentType(),
			ContentEncoding: contentEncoding,
			Body:            rsp.Payload(),
			DeliveryMode:    deliveryMode,
			Priority:        defaultPriority,
			CorrelationId:   rsp.MessageID(), // original msgid becomes the correlationid
			ReplyTo:         InstanceID,      // incase they need to reply back; we say where it came from
			// a bunch of application/implementation-specific fields
		},
	)

	if err != nil {
		return fmt.Errorf("Error sending response: %v", err)
	}

	return nil
}

// SendRequest via AMQP
func SendRequest(req Request, InstanceID string) error {
	log.Tracef("[Raven] Sending request %s to %s.%s, response back to %s", req.MessageID(), req.Service(), req.Endpoint(), InstanceID)

	if !Connected {
		return fmt.Errorf("[Raven] Error sending request, raven not connected")
	}

	// We only send string headers, so we can't send traceShouldPersistHeader as a bool
	traceShouldPersistHeader := "0"
	if req.TraceShouldPersist() {
		traceShouldPersistHeader = "1"
	}

	authorisedHeader := "0"
	if req.Authorised() {
		authorisedHeader = "1"
	}

	err := Publisher.channel.Publish(
		EXCHANGE, // publish to default exchange for reply-to
		"",       // blank routing key
		false,    // mandatory
		false,    // immediate
		amqp.Publishing{
			Headers: amqp.Table{
				"messageType":        "request",
				"service":            req.Service(),
				"endpoint":           req.Endpoint(),
				"traceID":            req.TraceID(),
				"traceShouldPersist": traceShouldPersistHeader,
				"sessionID":          req.SessionID(),
				"parentMessageID":    req.ParentMessageID(),
				"from":               req.From(),
				"remoteAddr":         req.RemoteAddr(),
				"authorised":         authorisedHeader,
			},
			ContentType:     req.ContentType(),
			ContentEncoding: contentEncoding,
			Body:            req.Payload(),
			DeliveryMode:    deliveryMode,
			Priority:        defaultPriority,
			MessageId:       req.MessageID(),
			ReplyTo:         InstanceID,
			// a bunch of application/implementation-specific fields
		},
	)

	if err != nil {
		return fmt.Errorf("Error sending request: %v", err)
	}

	return nil
}

// SendPublication via AMQP
func SendPublication(pub Publication, InstanceID string) error {
	log.Tracef("[Raven] Sending publication to topic: %s", pub.Topic())

	if !Connected {
		return fmt.Errorf("[Raven] Error sending publication, raven not connected")
	}

	err := Publisher.channel.Publish(
		TOPIC_EXCHANGE, // publish to topic exchange
		pub.Topic(),    // routing key = topic
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			Headers: amqp.Table{
				"messageType": "publication",
				"topic":       pub.Topic(),
				"sessionID":   pub.SessionID(),
			},
			ContentType:     pub.ContentType(),
			ContentEncoding: contentEncoding,
			Body:            pub.Payload(),
			DeliveryMode:    deliveryMode,
			Priority:        defaultPriority,
			MessageId:       pub.MessageID(),
			ReplyTo:         InstanceID,
			// a bunch of application/implementation-specific fields
		})

	if err != nil {
		return fmt.Errorf("Error sending publication: %s", err)
	}

	return nil
}

// SendHeartbeat via AMQP
func SendHeartbeat(hb Heartbeat, InstanceID string) error {
	log.Tracef("[Raven] Sending heartbeat to: %s", hb.ID())

	if !Connected {
		return fmt.Errorf("[Raven] Error sending heartbeat, raven not connected")
	}

	err := Publisher.channel.Publish(
		REPLY_EXCHANGE, // publish to default exchange for reply-to
		hb.ID(),        // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			Headers: amqp.Table{
				"messageType": "heartbeat",
				"heartbeat":   "ping",
			},
			ContentType:     hb.ContentType(),
			ContentEncoding: contentEncoding,
			Body:            hb.Payload(),
			DeliveryMode:    deliveryMode,
			Priority:        heartbeatPriority,
			ReplyTo:         InstanceID,
			// a bunch of application/implementation-specific fields
		},
	)

	if err != nil {
		return fmt.Errorf("[Raven] Error sending heartbeat: %v", err)
	}

	return nil
}
