package region

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"launchpad.net/tomb"

	log "github.com/cihub/seelog"
	nsqlib "github.com/HailoOSS/go-nsq"
	"github.com/HailoOSS/service/nsq"
)

const (
	topicName = "region.reload"
)

// DefaultInstance is the default config instance
var DefaultInstance *Regions = New()

// Regions represents a bunch of config settings
type Regions struct {
	tomb.Tomb
	observers    []chan bool
	observersMtx sync.RWMutex
}

func New() *Regions {
	r := &Regions{
		observers: make([]chan bool, 0),
	}

	go r.listenChanges()
	return r
}

// SubscribeChanges is a wrapper around DefaultInstance.SubscribeChanges
func SubscribeChanges() <-chan bool {
	return DefaultInstance.SubscribeChanges()
}

func (r *Regions) listenChanges() {
	for {
		// look out for region changes PUBbed via NSQ -- subscribe via a random ephemeral channel
		channel := fmt.Sprintf("g%v#ephemeral", rand.Uint32())
		subscriber, err := nsq.NewDefaultGlobalSubscriber(topicName, channel)
		if err != nil {
			log.Warnf("[Ingester] Failed to attach to topic '%v' topic for ingesting events: %v", topicName, err)
			time.Sleep(time.Second * 5)
			continue
		}
		subscriber.AddHandler(nsqlib.HandlerFunc(func(m *nsqlib.Message) error {
			// Notify observers
			r.observersMtx.RLock()
			defer r.observersMtx.RUnlock()
			for _, ch := range r.observers {
				// Non-blocking send
				select {
				case ch <- true:
				default:
				}
			}

			return nil
		}))

		log.Infof("[Config Load] Subscribing to region.reload")
		if err := subscriber.Connect(); err != nil {
			log.Warnf("[Config Load] Failed to connect to NSQ for region changes: %v", err)
			time.Sleep(time.Second * 5)
			continue
		}

		// Block until killed
		r.Wait()
		subscriber.Disconnect()

		return
	}
}

// SubscribeChanges will yield a channel which will then receive a boolean whenever
// the loaded configuration changes (depending on the exact loader used)
func (r *Regions) SubscribeChanges() <-chan bool {
	r.observersMtx.Lock()
	defer r.observersMtx.Unlock()

	ch := make(chan bool, 1)
	r.observers = append(r.observers, ch)

	return (<-chan bool)(ch)
}
