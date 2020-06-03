package circuitbreaker

// circuitbreaker will stop a client from calling an endpoint if
// that endpoint is failing.  This will stop one service from overwhelming
// another when the callee is in a failed state.
//
// This does not have a write lock in the open and result path unless
// the circuit is open.  Otherwise the write lock is only taken when the
// config is updated or the circuit is open.

import (
	"fmt"
	"strings"
	"sync"

	log "github.com/cihub/seelog"

	"github.com/hailo-platform/H2O/service/config"
)

func init() {
	// Ensure we update config when required
	ch := config.SubscribeChanges()
	go func() {
		for {
			<-ch
			loadFromConfig()
		}
	}()
	loadFromConfig()
}

var (
	circuitBreakers = make(map[string]Circuit) // Maps service/endpoint to Circuit
	lock            = &sync.RWMutex{}          // Protects circuitBreakers

	// The default endpoint config
	defaultOptions = Options{
		Disabled:            false,
		Threshold:           0.95,
		MinSamples:          100,
		Multiplier:          2,
		RandomizationFactor: 0,
		InitialIntervalMs:   100,              // 100 ms
		MaxIntervalMs:       60 * 1000,        // 1 min
		MaxElapsedTimeMs:    24 * 3600 * 1000, // 1 day
	}
)

// Circuit keeps track of whether a connection is open or closed
type Circuit interface {
	Open() bool
	Result(err error)
}

// Open is a helper function to call open on the correct default circuit
func Open(service, endpoint string) bool {
	return getCircuitBreaker(service, endpoint).Open()
}

// Result is a helper function to call Result on the correct default circuit
func Result(service, endpoint string, err error) {
	getCircuitBreaker(service, endpoint).Result(err)
}

func getCircuitBreaker(service, endpoint string) Circuit {
	key := fmt.Sprintf("%s.%s", service, endpoint)

	lock.RLock()
	breaker, ok := circuitBreakers[key]
	lock.RUnlock()

	if ok {
		return breaker
	}

	//Grab a write lock to create the breaker
	lock.Lock()
	defer lock.Unlock()
	// Double check no one else has created it
	if breaker, ok = circuitBreakers[key]; !ok {
		breaker = createCircuit(service, endpoint)
		circuitBreakers[key] = breaker
		return breaker
	}

	return breaker
}

func createCircuit(service, endpoint string) Circuit {
	options := defaultOptions
	config.AtPath("hailo", "platform", "circuitbreaker").AsStruct(&options)
	config.AtPath("hailo", "platform", "circuitbreaker", "endpoints", service, endpoint).AsStruct(&options)

	log.Debugf("Circuitbreaker config for %s.%s: %#v", service, endpoint, options)
	return NewDefaultCircuit(options)
}

type endpointsConfig struct {
	Options
	Endpoint map[string]Options `json:"endpoints"`
}

func loadFromConfig() {
	lock.Lock()
	defer lock.Unlock()

	for key := range circuitBreakers {
		service, endpoint := serviceAndEndpointFromKey(key)
		breaker := createCircuit(service, endpoint)

		circuitBreakers[key] = breaker
	}
}

func serviceAndEndpointFromKey(key string) (string, string) {
	parts := strings.Split(key, ".")
	switch len(parts) {
	case 0:
		return "", ""
	case 1:
		return key, ""
	default:
		return strings.Join(parts[:len(parts)-1], "."), parts[len(parts)-1]
	}
}
