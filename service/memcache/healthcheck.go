package memcache

import (
	"fmt"
	"github.com/HailoOSS/service/healthcheck"
	"github.com/HailoOSS/gomemcache/memcache"
)

const (
	HealthCheckId = "com.HailoOSS.service.memcache"
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
