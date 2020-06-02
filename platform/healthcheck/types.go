package healthcheck

import (
	"fmt"
	"sync"
	"time"

	"github.com/HailoOSS/service/config"
	hc "github.com/HailoOSS/service/healthcheck"
)

const (
	StandardInterval = time.Minute
	StandardPriority = hc.Warning
)

// HealthCheck represents a single health check
type HealthCheck struct {
	Id             string        `json:"-"`
	ServiceName    string        `json:"-"`
	ServiceVersion uint64        `json:"-"`
	Hostname       string        `json:"-"`
	InstanceId     string        `json:"-"`
	Checker        hc.Checker    `json:"-"`
	Interval       time.Duration `json:"interval"`
	Priority       hc.Priority   `json:"priority"`
}

// Sample represents the results of a single health check sample
type Sample struct {
	At               time.Time
	IsHealthy        bool
	ErrorDescription string
	Measurements     map[string]string
}

// results represents the health of some HealthCheck, and is an aggregation of the results
type results struct {
	sync.RWMutex
	hc         *HealthCheck
	lastSample *Sample
}

// ---

// newResults mints a result instance which includes the loop to check on health
func newResults(hc *HealthCheck) *results {
	r := &results{
		hc: hc,
	}

	go r.run()

	return r
}

// getLastSample returns lastSample and healthcheck details, with locking
func (r *results) getLastSample() (*HealthCheck, *Sample) {
	r.RLock()
	defer r.RUnlock()
	return r.hc, r.lastSample
}

// run is our main healthcheck loop
func (r *results) run() {
	ch := config.SubscribeChanges()
	for {
		select {
		// Listen for config changes and update the healthcheck when needed
		case <-ch:
			// Allow healthcheck parameters to be overridden in config
			config.AtPath("hailo", "platform", "healthcheck", r.hc.Id).AsStruct(r.hc)
		case <-runNow:
			r.collect()
		case <-time.After(r.hc.Interval):
			r.collect()
		}
	}
}

// collect will go and collect measurements
func (r *results) collect() {
	// go get data
	hc, _ := r.getLastSample()
	measurements, err := r.hc.Checker()

	// record results
	r.Lock()
	defer r.Unlock()
	r.lastSample = &Sample{
		At:           time.Now(),
		IsHealthy:    err == nil,
		Measurements: measurements,
	}
	if err != nil {
		r.lastSample.ErrorDescription = fmt.Sprintf("%v", err)
	}
	pubLastSample(hc, r.lastSample)
}
