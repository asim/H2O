package gocassa

import (
	"fmt"
	"strings"

	"github.com/HailoOSS/service/connhealthcheck"
	"github.com/HailoOSS/service/healthcheck"
	"github.com/HailoOSS/gocassa"
)

const (
	HealthCheckId  = "com.HailoOSS.service.cassandra-gocassa"
	MaxConnCheckId = "com.HailoOSS.service.cassandra-gocassa.maxconns"
)

// HealthCheck verifies we can connect to the supplied C* keyspace, and verifies the passed column families exist
func HealthCheck(keyspace gocassa.KeySpace, cfs ...string) healthcheck.Checker {
	return func() (map[string]string, error) {
		foundCfs_, err := keyspace.Tables()
		if err != nil {
			return nil, err
		}
		foundCfs := make(map[string]bool, len(foundCfs_))
		for _, cf := range foundCfs_ {
			foundCfs[strings.ToLower(cf)] = true
		}

		var missingCfs []string
		for _, cf := range cfs {
			cf = strings.ToLower(cf)
			if _, ok := foundCfs[cf]; !ok {
				missingCfs = append(missingCfs, cf)
			}
		}

		if len(missingCfs) > 0 {
			return nil, fmt.Errorf("Missing column families: %s", strings.Join(missingCfs, ", "))
		}
		return nil, nil
	}
}

// MaxConnHealthCheck asserts that the total number of established connections to all C* nodes falls below a given max
// threshold.
func MaxConnHealthCheck(maxconns int) healthcheck.Checker {
	return func() (map[string]string, error) {
		return connhealthcheck.MaxTcpConnections(getHosts(), maxconns)()
	}
}
