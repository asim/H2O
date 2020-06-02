package healthcheck

import (
	hcproto "github.com/HailoOSS/platform/proto/healthcheck"
	"sync"
)

var (
	mtx      sync.RWMutex
	registry map[string]*results = make(map[string]*results)
	runNow   chan bool           = make(chan bool)
)

// Register will add a healthcheck to our registry, causing it to be tested periodically and reported on
func Register(hc *HealthCheck) {
	mtx.Lock()
	defer mtx.Unlock()
	registry[hc.Id] = newResults(hc)
}

// Status
func Status() *hcproto.Response {
	mtx.RLock()
	defer mtx.RUnlock()

	rsp := &hcproto.Response{
		Healthchecks: make([]*hcproto.HealthCheck, 0),
	}
	for _, result := range registry {
		if hc, lastSample := result.getLastSample(); lastSample != nil {
			rsp.Healthchecks = append(rsp.Healthchecks, healthCheckSampleToProto(hc, lastSample))
		}
	}

	// try to kick off a background refresh of the data
	select {
	case runNow <- true:
	default:
	}

	return rsp
}
