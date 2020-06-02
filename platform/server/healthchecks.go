package server

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/HailoOSS/platform/circuitbreaker"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/service/config"
)

func initHealthChecks() {
	// add default healthcheck (to check we have config loaded - since we've just tried to load it)
	HealthCheck("com.HailoOSS.kernel.configloaded", func() (map[string]string, error) {
		ret := make(map[string]string)
		h, t := config.LastLoaded()
		ret["hash"] = h
		ret["lastLoaded"] = "[never]"
		if !t.IsZero() {
			ret["lastLoaded"] = fmt.Sprintf("%v", t.Format("2006-01-02 15:04:05"))
		}
		if ret["hash"] == "" || ret["lastLoaded"] == "[never]" {
			return ret, fmt.Errorf("Config not loaded")
		}
		return ret, nil
	})

	// add default healthcheck (to check the service to service errors)
	HealthCheck("com.HailoOSS.kernel.servicetoservice.auth.badrole", func() (map[string]string, error) {
		ret := make(map[string]string)

		var failing []string

		// Get the error counts
		counters := errors.Get("com.HailoOSS.kernel.auth.badrole")

		failed := 0
		for name, count := range counters {
			if count <= 5 {
				continue
			}

			failed += count

			if split := strings.Split(name, ":"); len(split) > 1 {
				name = strings.Join(split[1:], ".")
			}

			ret[name] = fmt.Sprintf("%d", count)
			failing = append(failing, fmt.Sprintf("%s: %d", name, count))
		}

		if cleared := errors.Cleared(); time.Since(cleared).Seconds() > 60 {
			// Clear the errors counts
			errors.Clear("com.HailoOSS.kernel.auth.badrole")
		}

		if len(failing) > 0 {
			return ret, fmt.Errorf("%d failed calls in last minute to %d services: %s", failed, len(failing), strings.Join(failing, ", "))
		}

		return ret, nil
	})

	// add default healthcheck (to check the platform capacity)
	HealthCheck("com.HailoOSS.kernel.resource.capacity", func() (map[string]string, error) {
		capacity := 0
		offendingCallers := []string{}

		tokensMtx.RLock()
		for caller, tokC := range tokens {
			capacity += cap(tokC)
			if len(tokC) == 0 {
				offendingCallers = append(offendingCallers, caller)
			}
		}
		tokensMtx.RUnlock()

		var err error
		if len(offendingCallers) > 0 {
			err = fmt.Errorf("Callers exceeding capacity: %s", strings.Join(offendingCallers, ", "))
		}

		return map[string]string{
			"capacity": fmt.Sprintf("%d", capacity),
			"inflight": fmt.Sprintf("%d", atomic.LoadUint64(&inFlightRequests)),
		}, err
	})

	HealthCheck("com.HailoOSS.kernel.client.circuit", circuitbreaker.CircuitHealthCheck)
}
