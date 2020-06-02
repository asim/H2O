package nsq

import (
	"crypto/md5"
	"fmt"
	"io"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	nsqlib "github.com/HailoOSS/go-nsq"
	"github.com/HailoOSS/service/config"
)

type Subscriber interface {
	// AddHandler registers something to handle inbound messages
	AddHandler(handler nsqlib.Handler)

	// AddHandlers registers something to handle inbound messages. This function uses
	// config to determine how many handlers should be started.
	AddHandlers(handler nsqlib.Handler)

	// SetMaxInFlight defines how many messages NSQ should punt our way at a time
	SetMaxInFlight(int)

	// IsStarved indicates whether any connection will reach max in flight
	IsStarved() bool

	// Connect initiates the NSQ config-driven connection loop
	// NOTE: this should be the last thing you do
	Connect() error

	// Disconnect stops the consumer and the config loop
	Disconnect()

	// SetConfig sets a config value on the underlying NSQ consumer
	SetConfig(option string, value interface{}) error
}

type DefaultSubscriber struct {
	topic        string
	channel      string
	consumer     *nsqlib.Consumer
	cfg          *nsqlib.Config
	handlers     []nsqlib.Handler
	stop         chan struct{}
	configHash   string
	subHosts     []string
	lookupdHosts []string
}

// NewDefaultSubscriber yields a DefaultSubscriber that automatically connects to
// the configured (via config service) nsqlookupds to find nodes hosting the
// messages for the given topic
func NewDefaultSubscriber(topic string, channel string) (Subscriber, error) {
	return &DefaultSubscriber{
		cfg:     nsqlib.NewConfig(),
		topic:   topic,
		channel: channel,
		stop:    make(chan struct{}),
	}, nil
}

func (s *DefaultSubscriber) SetConfig(option string, value interface{}) error {
	return s.cfg.Set(option, value)
}

func (s *DefaultSubscriber) SetMaxInFlight(v int) {
	s.cfg.MaxInFlight = v
}

func (s *DefaultSubscriber) AddHandler(handler nsqlib.Handler) {
	s.handlers = append(s.handlers, handler)
}

func (s *DefaultSubscriber) AddHandlers(handler nsqlib.Handler) {
	subHandlers := config.AtPath("hailo", "service", "nsq", "subHandlers").AsInt(6)
	log.Infof("Adding %d handlers", subHandlers)
	for i := 0; i < subHandlers; i++ {
		s.AddHandler(handler)
	}
}

func (s *DefaultSubscriber) IsStarved() bool {
	return s.consumer.IsStarved()
}

func (s *DefaultSubscriber) Connect() error {
	consumer, err := nsqlib.NewConsumer(s.topic, s.channel, s.cfg)
	if err != nil {
		return err
	}
	consumer.SetLogger(&logBridge{}, nsqlib.LogLevelInfo)
	for _, handler := range s.handlers {
		consumer.AddHandler(handler)
	}
	s.consumer = consumer
	go s.configLoop()
	return nil
}

func (s *DefaultSubscriber) Disconnect() {
	close(s.stop)
	s.consumer.Stop()
	<-s.consumer.StopChan
}

func (s *DefaultSubscriber) configLoop() {
	// Wait 5 mins for config to load. If we cannot load config by the
	// then there's most likely a major issue and we should panic.
	if !config.WaitUntilLoaded(5 * time.Minute) {
		panic("Waiting 5 mins for config to load")
	}

	s.loadFromConfig()
	ch := config.SubscribeChanges()
	for {
		select {
		case <-ch:
			s.loadFromConfig()
		case <-s.stop:
			return
		}
	}
}

func (s *DefaultSubscriber) loadFromConfig() {
	for {
		err := s.doLoad()
		if err == nil {
			break
		}
		log.Warn(err)
		time.Sleep(time.Second)
	}
}

func (s *DefaultSubscriber) doLoad() error {
	subHosts := config.AtPath("hailo", "service", "nsq", "subHosts").AsHostnameArray(4150)
	disableLookupd := config.AtPath("hailo", "service", "nsq", "disableLookupd").AsBool()
	lookupdHosts := getHosts(4161, "hailo", "service", "nsq", "nsqlookupdSeeds")

	h := md5.New()
	io.WriteString(h, strings.Join(subHosts, ","))
	io.WriteString(h, fmt.Sprintf("%v", disableLookupd))
	io.WriteString(h, strings.Join(lookupdHosts, ","))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	if s.configHash == hash {
		return nil // don't bother as nothing interesting has changed
	}

	if disableLookupd {
		log.Infof("Connecting to NSQ directly: %v", subHosts)
		var hostList []string
		for _, addr := range subHosts {
			// support comma separated host lists too
			hostList = append(hostList, strings.Split(addr, ",")...)
		}
		err := s.consumer.ConnectToNSQDs(hostList)
		if err != nil && err != nsqlib.ErrAlreadyConnected {
			return fmt.Errorf("Error connecting to nsqd(s): %v", err)
		}
	} else {
		if len(lookupdHosts) > 0 {
			log.Infof("Connecting to NSQ via lookupd hosts: %v", lookupdHosts)
			err := s.consumer.ConnectToNSQLookupds(lookupdHosts)
			if err != nil {
				return fmt.Errorf("Error connecting to nsqlookupd(s): %v", err)
			}
		}
	}

	// Disconnect from old hosts
	if hosts := diffHosts(s.subHosts, subHosts); len(hosts) > 0 {
		log.Infof("Disconnecting from NSQ hosts: %v", hosts)
		for _, host := range hosts {
			err := s.consumer.DisconnectFromNSQD(host)
			if err != nil && err != nsqlib.ErrNotConnected {
				log.Warnf("Error disconnecting from NSQ host %s: %v", host, err)
			}
		}
	}

	// Disconnect from old lookupds
	if hosts := diffHosts(s.lookupdHosts, lookupdHosts); len(hosts) > 0 {
		log.Infof("Disconnecting from NSQ lookupds: %v", hosts)
		for _, host := range hosts {
			err := s.consumer.DisconnectFromNSQLookupd(host)
			if err != nil && err != nsqlib.ErrNotConnected {
				log.Warnf("Error disconnecting from NSQ host %s: %v", host, err)
			}
		}
	}

	// save state on success
	s.configHash = hash
	s.subHosts = subHosts
	s.lookupdHosts = lookupdHosts

	return nil
}
