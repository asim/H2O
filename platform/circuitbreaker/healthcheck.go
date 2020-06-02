package circuitbreaker

import (
	"fmt"
)

// CircuitHealthCheck will report if a circuit is open
func CircuitHealthCheck() (map[string]string, error) {
	ret := make(map[string]string)
	var err error

	// Check if any circuits are open
	lock.RLock()
	defer lock.RUnlock()
	for key, breaker := range circuitBreakers {
		if breaker.Open() {
			ret[key] = "OPEN"
			err = fmt.Errorf("Open Circuit")
		}
	}

	return ret, err
}
