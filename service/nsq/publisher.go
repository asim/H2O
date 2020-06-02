package nsq

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/stretchr/testify/mock"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/go-hostpool"
	nsqlib "github.com/HailoOSS/go-nsq"
	"github.com/HailoOSS/service/config"
	inst "github.com/HailoOSS/service/instrumentation"
)

const (
	cl_ONE = iota
	cl_TWO
	cl_QUORUM
	maxQueueLength = 32
)

var (
	configRetryDelay = time.Millisecond * 100

	ErrEmptyBody = fmt.Errorf("Attempted to publish empty body")
)

// getDLQName returns the name of a deadletter queue for a topic/channel.
func getDLQName(topic, channel string) string {
	deadletter := fmt.Sprintf("%s.%s.dl", topic, channel)
	if len(deadletter) > maxQueueLength {
		topicLen := len(topic)
		if topicLen > 14 {
			topicLen = 14
		}
		channLen := len(channel)
		if channLen > maxQueueLength-topicLen-4 {
			channLen = maxQueueLength - topicLen - 4
		}

		deadletter = fmt.Sprintf("%s.%s.dl", topic[:topicLen], channel[:channLen])
	}
	return deadletter
}

// Publisher is our wrapper round NSQ PUB for auto-config
type Publisher interface {
	MultiPublish(topic string, body [][]byte) error
	Publish(topic string, body []byte) error
}

// MultiPublish wraps DefaultPublisher.MultiPublish
func MultiPublish(topic string, body [][]byte) error {
	return DefaultPublisher.MultiPublish(topic, body)
}

// Publish wraps DefaultPublisher.Publish
func Publish(topic string, body []byte) error {
	return DefaultPublisher.Publish(topic, body)
}

// PublishDeadLetter puts messages on the deadletter queue for a topic/channel.
func PublishDeadLetter(topic, channel string, body []byte) error {
	deadletter := getDLQName(topic, channel)
	log.Errorf("Failed to process message. Sending to deadletter %s", deadletter)
	var err error
	for i := 0; i < 3; i++ {
		if err = Publish(deadletter, body); err == nil {
			return nil
		}
	}
	log.Errorf("Failed to publish message to deadletter nsq %s %s %s", deadletter, err, body)
	return err
}

// HostpoolPublisher is our default publisher which gets N hosts from config and then
// allows us to PUB to M of them
type HostpoolPublisher struct {
	sync.RWMutex
	once sync.Once
	// consistency-level (borrowed from c* - how many we need to PUB to for success)
	cl int
	// a producer per configued NSQ pub host
	hostpool  hostpool.HostPool
	producers map[string]*nsqlib.Producer
	// how many of these we have (count) and how many we need to write to (n)
	count, n int
	// sample rate for NSQ instrumentation
	instrumentationSampleRate float32
	// hash of current config, to avoid locking and updating if we don't have to
	configHash string
}

// DefaultPublisher is a default implementation of the Publisher interface.
// It is an instance of a HostpoolPublisher
var DefaultPublisher Publisher = &HostpoolPublisher{
	producers: make(map[string]*nsqlib.Producer),
	hostpool:  hostpool.New([]string{}),
}

// MultiPublish pubs X messages at once, synchronously, to N of M NSQs
func (p *HostpoolPublisher) MultiPublish(topic string, body [][]byte) error {

	if len(body) <= 0 {
		// This is a fatal error in the server that causes a disconnect.
		// Let's not send it
		return ErrEmptyBody
	}

	p.once.Do(p.setup)

	p.RLock()
	defer p.RUnlock()

	// must have some hosts configured
	if p.count == 0 {
		return fmt.Errorf("No NSQ pubHosts configured (hailo.service.nsq.pubHosts)")
	}

	// need to achieve N successful results; we'll round-robin from random starting point
	seen := make(map[string]bool)
	maxTries := p.count + p.count
	errarr := make([]error, 0)
	for try := 0; try < maxTries; try++ {
		// Get new host from pool
		hpr := p.hostpool.Get()
		host := hpr.Host()

		// have we already PUBbed to this?
		if _, exists := seen[host]; exists {
			continue
		}

		// Instrument payload body size
		inst.Timing(p.instrumentationSampleRate, "nsq.publish."+topic+".size", time.Duration(len(body)))

		// try this one
		err := p.producers[host].MultiPublish(topic, body)
		hpr.Mark(err) // Update hostpool
		if err != nil {
			errarr = append(errarr, err)
			continue
		}

		seen[host] = true
		// have we PUBbed to enough yet?
		if len(seen) >= p.n {
			break
		}
	}

	if len(seen) < p.n {
		errStr := "Errors"
		for _, anErr := range errarr {
			errStr += fmt.Sprintf(" :: %v", anErr)
		}
		return fmt.Errorf("Could not PUB to enough NSQs to consider this a success. Did PUB to %v out of %v attempts. %v", len(seen), maxTries, errStr)
	}

	return nil
}

// Publish will PUB a message to N of M NSQs
func (publisher *HostpoolPublisher) Publish(topic string, body []byte) error {
	return publisher.MultiPublish(topic, [][]byte{body})
}

// setup is a one-time action that loads PUB hosts from config and sets up a config subscriber
func (p *HostpoolPublisher) setup() {
	// Wait 5 mins for config to load. If we cannot load config by the
	// then there's most likely a major issue and we should panic.
	if !config.WaitUntilLoaded(5 * time.Minute) {
		panic("Waiting 5 mins for config to load")
	}

	ch := config.SubscribeChanges()
	p.loadFromConfig()
	go func() {
		for {
			<-ch
			for {
				if err := p.loadFromConfig(); err != nil {
					log.Warnf("Failed to load NSQ PUB config: %v", err)
					time.Sleep(configRetryDelay)
				} else {
					break
				}
			}
		}
	}()
}

// loadFromConfig grabs latest config, then diffs against currently loaded
func (p *HostpoolPublisher) loadFromConfig() error {
	cl := config.AtPath("hailo", "service", "nsq", "writeCl").AsString("ONE")
	pubHosts := getHosts(4150, "hailo", "service", "nsq", "pubHosts")
	hbInterval := config.AtPath("hailo", "service", "nsq", "pubHeartbeatInterval").AsDuration("30s")

	// hash and test
	hash, _ := config.LastLoaded()
	if p.configHash == hash {
		return nil
	}
	p.configHash = hash

	// lock now and then update everything
	p.Lock()
	defer p.Unlock()

	canonicalHosts := make(map[string]bool)
	for _, host := range pubHosts {
		canonicalHosts[host] = true
		// do we have a producer for this host?
		if _, ok := p.producers[host]; !ok {
			cfg := nsqlib.NewConfig()
			cfg.HeartbeatInterval = hbInterval
			prod, err := nsqlib.NewProducer(host, cfg)
			if err != nil {
				return err
			}
			prod.SetLogger(&logBridge{}, nsqlib.LogLevelDebug)
			p.producers[host] = prod
		}
	}

	// now remove any removed ones
	for host, prod := range p.producers {
		if !canonicalHosts[host] {
			delete(p.producers, host)
			prod.Stop()
		}
	}

	// add hosts to hostpool
	p.hostpool.SetHosts(pubHosts)

	log.Infof("Initialized NSQ publisher with hosts %v", strings.Join(pubHosts, ", "))

	// setup the other meta data
	p.count = len(p.producers)
	switch cl {
	case "TWO":
		p.cl = cl_TWO
		p.n = 2
	case "QUORUM":
		p.cl = cl_QUORUM
		p.n = p.count/2 + 1
	default:
		p.cl = cl_ONE // our default
		p.n = 1
	}

	p.configHash = hash

	return nil
}

type MockPublisher struct {
	mock.Mock
}

func (p *MockPublisher) MultiPublish(topic string, body [][]byte) error {
	ret := p.Mock.Called(topic, body)

	return ret.Error(0)
}

func (p *MockPublisher) Publish(topic string, body []byte) error {
	ret := p.Mock.Called(topic, body)

	return ret.Error(0)
}
