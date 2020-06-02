package client

import (
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/davegardnerisme/deephash"
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/service/config"

	eps "github.com/HailoOSS/discovery-service/proto/endpoints"
)

var (
	defaultTimeout    = time.Millisecond * 2000
	defaultMin        = "10ms"
	defaultMax        = "60s"
	defaultMultiplier = 1.0
)

// Timeout stores state information about services in order to make decisions on what timeout to use when contacting
// other services
type Timeout struct {
	sync.RWMutex
	// endpoints is a map of service+endpoint -> timeout which we load on-demand
	endpoints map[string]map[string]time.Duration
	// Client to use for making requests
	client Client
	// "dial" settings
	min, max   time.Duration
	multiplier float64
}

// NewTimeout mints a blank timeout container from which we can calculate timeouts to use for requests
func NewTimeout(c Client) *Timeout {
	t := &Timeout{
		endpoints: make(map[string]map[string]time.Duration),
		client:    c,
	}

	// trigger occasional background reloads of timeouts
	go func() {
		ticker := time.NewTicker(loadTimeoutsInterval)
		for {
			<-ticker.C
			t.reloadSlas()
		}
	}()

	// keep watch on config updates
	ch := config.SubscribeChanges()
	go func() {
		for {
			<-ch
			t.loadFromConfig()
		}
	}()
	t.loadFromConfig()

	return t
}

// Get timeout to use for an attempt made calling some service
// our strategy is to always return a timeout immediately, and if we don't have
// any knowledge of what a good timeout is, pick a default and trigger a background
// load from the discovery service
func (t *Timeout) Get(service, endpoint string, attempt int) time.Duration {
	d, exists := t.fetchSla(service, endpoint)
	if !exists {
		// need to trigger timeout loads from discovery service
		t.add(service, endpoint)
		go t.reloadSlas()
	}

	// apply controls
	d *= time.Duration(t.multiplier)

	// apply relaxation backoff -- this is a little weird, but we have to cast attempt to duration to multiply
	d *= time.Duration(attempt)

	// apply configured min/max bounds
	if d > t.max {
		d = t.max
	} else if d < t.min {
		d = t.min
	}

	return d
}

// loadFromConfig will grab the configurable settings from config service
func (t *Timeout) loadFromConfig() {
	min := config.AtPath("hailo", "platform", "timeout", "min").AsDuration(defaultMin)
	max := config.AtPath("hailo", "platform", "timeout", "max").AsDuration(defaultMax)
	multiplier := config.AtPath("hailo", "platform", "timeout", "multiplier").AsFloat64(defaultMultiplier)

	// any difference?
	if hashTimeouts(min, max, multiplier) == t.hashTimeouts() {
		return
	}

	t.Lock()
	defer t.Unlock()
	t.min = min
	t.max = max
	t.multiplier = multiplier

	log.Infof("[Client] Loaded timeout configuration from config service [min=%v, max=%v, multiplier=%v]", min, max, multiplier)
}

// fetchSla will grab timeout from our map, and let us know if it was found
func (t *Timeout) fetchSla(service, endpoint string) (time.Duration, bool) {
	t.RLock()
	defer t.RUnlock()

	if serviceEndpoints, ok := t.endpoints[service]; ok {
		if sla, ok := serviceEndpoints[endpoint]; ok {
			return sla, true
		}
	}

	return defaultTimeout, false
}

// add some service + endpoint to our list
func (t *Timeout) add(service, endpoint string) {
	t.Lock()
	defer t.Unlock()

	if _, ok := t.endpoints[service]; !ok {
		t.endpoints[service] = make(map[string]time.Duration)
	}
	t.endpoints[service][endpoint] = defaultTimeout
}

// reloadSlas loads timeouts from discovery service for all services we know about (have tried to call)
func (t *Timeout) reloadSlas() {
	replacement := make(map[string]map[string]time.Duration)

	t.RLock()
	for service := range t.endpoints {
		// load from discovery service
		log.Debugf("[Client] Loading SLAs from discovery service for %v...", service)
		req, err := NewRequest("com.HailoOSS.kernel.discovery", "endpoints", &eps.Request{
			Service: proto.String(service),
		})
		if err != nil {
			log.Warnf("[Client] Failed to create proto request to get endpoints for service: %s", service)
			continue
		}
		rsp := &eps.Response{}
		// explicitly define timeout since we're in no rush
		if err := t.client.Req(req, rsp, Options{"retries": 0, "timeout": time.Second * 5}); err != nil {
			log.Warnf("[Client] Trouble getting endpoint response back from discovery-service for service: %s", service)
			continue
		}

		for _, ep := range rsp.GetEndpoints() {
			endpoint := strings.TrimLeft(strings.TrimPrefix(ep.GetFqName(), service), ".")
			if _, ok := replacement[service]; !ok {
				replacement[service] = make(map[string]time.Duration)
			}
			replacement[service][endpoint] = msToDuration(ep.GetUpper95())
		}
	}

	// double check we have all the things we started with -- if not, but back the "last known" (probably defaults)
	for service, serviceEndpoints := range t.endpoints {
		for endpoint, timeout := range serviceEndpoints {
			if _, ok := replacement[service]; !ok {
				replacement[service] = make(map[string]time.Duration)
			}
			if _, ok := replacement[service][endpoint]; !ok {
				log.Debugf("[Client] Failed to find SLA for %s.%s, falling back to %v", service, endpoint, timeout)
				replacement[service][endpoint] = timeout
			}
		}
	}
	t.RUnlock()

	// SLAs changed? if not, don't bother switching+logging
	if hashSlas(replacement) == t.hashEndpoints() {
		return
	}

	t.Lock()
	defer t.Unlock()
	t.endpoints = replacement

	log.Infof("[Client] Loaded new SLAs from discovery service: %v", t.endpoints)
}

func (t *Timeout) hashEndpoints() string {
	t.RLock()
	defer t.RUnlock()

	return hashSlas(t.endpoints)
}

func (t *Timeout) hashTimeouts() string {
	t.RLock()
	defer t.RUnlock()

	return hashTimeouts(t.min, t.max, t.multiplier)
}

func hashSlas(m map[string]map[string]time.Duration) string {
	return fmt.Sprintf("%x", deephash.Hash(m))
}

func hashTimeouts(min, max time.Duration, multiplier float64) string {
	return fmt.Sprintf("%x", deephash.Hash(struct {
		Min, Max   time.Duration
		Multiplier float64
	}{
		Min:        min,
		Max:        max,
		Multiplier: multiplier,
	}))
}

func msToDuration(ms uint32) time.Duration {
	return time.Duration(int64(time.Millisecond) * int64(ms))
}
