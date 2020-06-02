package sessinvalidator

import (
	log "github.com/cihub/seelog"

	nsqlib "github.com/HailoOSS/go-nsq"
	"github.com/HailoOSS/service/auth"
	"github.com/HailoOSS/service/nsq"
)

// Run will connect to NSQ, consume and process session expiry requests
// We can trigger after initialisation via `server.RegisterPostConnectHandler(sessinvalidator.Run)`
func Run() {
	log.Info("Launching session invalidator...")
	subscriber, err := nsq.NewDefaultSubscriber(TopicName, ChannelName)
	if err != nil {
		log.Warnf("Failed to attach to %v topic for processing session expiry: %v", TopicName, err)
	}
	subscriber.SetMaxInFlight(maxInFlight)
	subscriber.AddHandler(&Processor{})
	subscriber.Connect()
}

// Processor exists to strap a HandleMessage to in order to satisfy http://godoc.org/github.com/bitly/go-nsq#Handler
type Processor struct{}

// HandleMessage implements http://godoc.org/github.com/bitly/go-nsq#Handler for processing our jobs
func (p *Processor) HandleMessage(msg *nsqlib.Message) error {
	sessId := string(msg.Body)
	if err := auth.Invalidate(sessId); err != nil {
		log.Warnf("Failed to invalidate session '%v'", sessId)
		return err
	}

	log.Debugf("Session '%v' expired from cache", sessId)

	return nil
}
