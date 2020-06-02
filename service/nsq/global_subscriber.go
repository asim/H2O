package nsq

import (
	nsqlib "github.com/HailoOSS/go-nsq"
)

const (
	federatedExt = ".federated"
)

// DefaultGlobalSubscriber is the default global subscriber which encapsulates
// local and federated DefaultSubscribers.
type DefaultGlobalSubscriber struct {
	localSubscriber     Subscriber
	federatedSubscriber Subscriber
}

func (s *DefaultGlobalSubscriber) SetMaxInFlight(v int) {
	s.localSubscriber.SetMaxInFlight(v)
	s.federatedSubscriber.SetMaxInFlight(v)
}

func (s *DefaultGlobalSubscriber) AddHandler(handler nsqlib.Handler) {
	s.localSubscriber.AddHandler(handler)
	s.federatedSubscriber.AddHandler(handler)
}

func (s *DefaultGlobalSubscriber) AddHandlers(handler nsqlib.Handler) {
	s.localSubscriber.AddHandlers(handler)
	s.federatedSubscriber.AddHandlers(handler)
}

func (s *DefaultGlobalSubscriber) Connect() error {
	if err := s.localSubscriber.Connect(); err != nil {
		return err
	}
	if err := s.federatedSubscriber.Connect(); err != nil {
		return err
	}
	return nil
}

func (s *DefaultGlobalSubscriber) Disconnect() {
	s.localSubscriber.Disconnect()
	s.federatedSubscriber.Disconnect()
}

func (s *DefaultGlobalSubscriber) IsStarved() bool {
	return s.localSubscriber.IsStarved() || s.federatedSubscriber.IsStarved()
}

func (s *DefaultGlobalSubscriber) SetConfig(option string, value interface{}) error {
	if err := s.localSubscriber.SetConfig(option, value); err != nil {
		return err
	}
	if err := s.federatedSubscriber.SetConfig(option, value); err != nil {
		return err
	}
	return nil
}

// NewDefaultGlobalSubscriber yields a Subscriber which automatically connects
// to the configured (via config service) nodes providing the messages for the
// given topic and its federated counterpart. This allows a client to receive
// messages pubbed within a local region and those federated globally.
func NewDefaultGlobalSubscriber(topic, channel string) (Subscriber, error) {
	// Subscribe to topic
	localSubscriber, err := NewDefaultSubscriber(topic, channel)
	if err != nil {
		return nil, err
	}

	// Subscribe to federated topic
	federatedSubscriber, err := NewDefaultSubscriber(topic+federatedExt, channel)
	if err != nil {
		return nil, err
	}

	return &DefaultGlobalSubscriber{
		localSubscriber:     localSubscriber,
		federatedSubscriber: federatedSubscriber,
	}, nil
}
