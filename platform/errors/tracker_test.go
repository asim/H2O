package errors

import (
	"fmt"
	"testing"
)

func TestTracker(t *testing.T) {
	for i := 0; i < 10; i++ {
		code := fmt.Sprintf("errorCode%d", i)
		from := fmt.Sprintf("fromService%d", i)
		service := fmt.Sprintf("toService%d", i)
		endpoint := fmt.Sprintf("toEndpoint%d", i)

		// Track some errors
		Track(code, from, service, endpoint)

		// Check the counts after track
		if count := Count(code, from, service, endpoint); count != 1 {
			t.Errorf("Got unexpected count for %s:%s:%s:%s. Expected %d Got %d", code, from, service, endpoint, 1, count)
		}

		// Clear the errors
		Clear(code, from, service, endpoint)

		// Check the counts after clear
		if count := Count(code, from, service, endpoint); count != 0 {
			t.Errorf("Got unexpected count for %s:%s:%s:%s. Expected %d Got %d", code, from, service, endpoint, 0, count)
		}

		// Track multiple errors
		for j := 0; j < 10; j++ {
			Track(code, from, service, endpoint)

			// Track a new service
			newService := fmt.Sprintf("%s%d", service, j)
			Track(code, from, newService, endpoint)

			// Check the counts for the new service
			if count := Count(code, from, newService, endpoint); count != 1 {
				t.Errorf("Got unexpected count for %s:%s:%s:%s. Expected %d Got %d", code, from, newService, endpoint, 1, count)
			}
		}

		// Check the counts after track
		if count := Count(code, from, service, endpoint); count != 10 {
			t.Errorf("Got unexpected count for %s:%s:%s:%s. Expected %d Got %d", code, from, service, endpoint, 10, count)
		}

		// Check the counters match
		counters := Get(code)
		name := counterName(from, service, endpoint)
		if _, ok := counters[name]; !ok {
			t.Errorf("Could not find counter %s in counters %v", name, counters)
		}

		// Clear
		Clear(code)
	}

}
