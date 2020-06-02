package circuitbreaker

import (
	"bytes"
	"testing"
	"time"

	"github.com/facebookgo/clock"

	"github.com/HailoOSS/platform/errors"
	ptesting "github.com/HailoOSS/platform/testing"
	"github.com/HailoOSS/service/config"
)

type CircuitBreakerTestSuit struct {
	ptesting.Suite
	circuit Circuit
	clock   *clock.Mock
}

func TestRunCircuitBreakerTestSuite(t *testing.T) {
	ptesting.RunSuite(t, new(CircuitBreakerTestSuit))
}

func (s *CircuitBreakerTestSuit) SetupTest() {
	s.Suite.SetupTest()
	s.clock = clock.NewMock()
	defaultClock = s.clock
}

func (s *CircuitBreakerTestSuit) TestOpenCircuit() {
	s.circuit = NewDefaultCircuit(defaultOptions)

	// This should trip the circuit breaker
	for i := 0; i < 100; i++ {
		s.circuit.Result(errors.Timeout("code", "description"))
	}

	s.True(s.circuit.Open())
}

func (s *CircuitBreakerTestSuit) TestClosedCircuit() {
	s.circuit = NewDefaultCircuit(defaultOptions)
	s.False(s.circuit.Open())
}

func (s *CircuitBreakerTestSuit) TestOpenToClosedCircuit() {
	s.circuit = NewDefaultCircuit(defaultOptions)

	// Trip the circuit breaker
	for i := 0; i < 100; i++ {
		s.circuit.Result(errors.Timeout("code", "description"))
	}
	s.True(s.circuit.Open())

	// Wait and do a successful call. Circuit should be closed.
	s.clock.Add(101 * time.Millisecond)
	s.False(s.circuit.Open())
	s.circuit.Result(nil)
	s.False(s.circuit.Open())
}

func (s *CircuitBreakerTestSuit) TestCircuitConfig() {
	service := "com.HailoOSS.test.cruft"
	endpoint := "testendpoint"

	// Set default timeout to 100 ms
	config.Load(bytes.NewBuffer([]byte(`{
		"hailo": {
			"platform": {
				"circuitbreaker": {
					"initialIntervalMs": 100
				}
			}
		}
	}`)))

	// Let config propagate -- crufty :(
	time.Sleep(50 * time.Millisecond)

	// Circuit is initially closed
	s.False(Open(service, endpoint))

	// Trip circuit and check
	for i := 0; i < 100; i++ {
		Result(service, endpoint, errors.Timeout("code", "description"))
	}
	s.True(Open(service, endpoint))

	// Wait for circuit to half-open
	s.clock.Add(51 * time.Millisecond)
	s.True(Open(service, endpoint), "Circuit should be open after 51ms")
	s.clock.Add(50 * time.Millisecond)
	s.False(Open(service, endpoint), "Circuit should be closed after 101ms")

	// Set new interval
	s.NoError(config.Load(bytes.NewBuffer([]byte(`{
		"hailo": {
			"platform": {
				"circuitbreaker": {
					"initialIntervalMs": 50,
					"endpoints": {
						"com.HailoOSS.test.cruft": {
							"testendpoint": {
								"initialIntervalMs": 90
							}
						}
					}
				}
			}
		}
	}`))))

	// Let config propagate -- crufty :(
	time.Sleep(50 * time.Millisecond)

	// Check circuit is closed again
	s.False(Open(service, endpoint))

	// Trip circuit and check
	for i := 0; i < 100; i++ {
		Result(service, endpoint, errors.Timeout("code", "description"))
	}
	s.True(Open(service, endpoint))

	// Wait for circuit to half-open
	s.clock.Add(51 * time.Millisecond)
	s.True(Open(service, endpoint), "Circuit should be open after 51ms")
	s.clock.Add(40 * time.Millisecond)
	s.False(Open(service, endpoint), "Circuit should be closed after 91ms")
}
