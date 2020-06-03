package memcache

import (
	"fmt"
	"github.com/hailo-platform/H2O/service/healthcheck"
	"github.com/hailo-platform/H2O/gomemcache/memcache"
)

const (
	HealthCheckId = "com.hailo-platform/H2O.service.memcache"
)

// HealthCheck asserts we can talk to memcache
func HealthCheck() healthcheck.Checker {
	return func() (map[string]string, error) {
		_, err := defaultClient.Get("healthcheck")
		if err != nil && err != memcache.ErrCacheMiss {
			return nil, fmt.Errorf("Memcache operation failed: %v", err)
		}

		return nil, nil
	}
}
