package cassandra

import (
	"fmt"

	"github.com/HailoOSS/service/connhealthcheck"
	"github.com/HailoOSS/service/healthcheck"
)

const (
	HealthCheckId  = "com.HailoOSS.service.cassandra"
	MaxConnCheckId = "com.HailoOSS.service.cassandra.maxconns"
)

// HealthCheck asserts we can connect to C* and
func HealthCheck(keyspace string, cfs []string) healthcheck.Checker {
	return func() (map[string]string, error) {
		pool, err := ConnectionPool(keyspace)
		if err != nil {
			return nil, fmt.Errorf("Failed to get connection pool for keyspace '%v': %v", keyspace, err)
		}
		schema := pool.Schema()
		for _, cf := range cfs {
			if _, exists := schema.ColumnFamilies[cf]; !exists {
				return nil, fmt.Errorf("ColumnFamily '%v' does not exist", cf)
			}
		}

		return nil, nil
	}
}

// MaxConnHealthCheck asserts that the total number of established connections to all cassandra nodes
// falls below a given max threshold.
func MaxConnHealthCheck(maxconns int) healthcheck.Checker {
	return func() (map[string]string, error) {
		return connhealthcheck.MaxTcpConnections(getHosts(), maxconns)()
	}
}
