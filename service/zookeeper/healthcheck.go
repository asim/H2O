package zookeeper

import (
	"fmt"
	"github.com/hailo-platform/H2O/service/config"
	"github.com/hailo-platform/H2O/service/connhealthcheck"
	"github.com/hailo-platform/H2O/service/healthcheck"
)

const (
	HealthCheckId  = "com.hailocab.service.zookeeper"
	MaxConnCheckId = "com.hailocab.service.zookeeper.maxconns"
)

// HealthCheck asserts we can talk to ZK
func HealthCheck() healthcheck.Checker {
	return func() (map[string]string, error) {
		_, _, err := Exists("/healthcheck")
		if err != nil {
			return nil, fmt.Errorf("Zookeeper operation failed: %v", err)
		}
		return nil, nil
	}
}

// MaxConnHealthCheck asserts that the total number of established connections to all zookeeper nodes
// falls below a given max threshold.
func MaxConnHealthCheck(maxconns int) healthcheck.Checker {
	return func() (map[string]string, error) {
		nodes := config.AtPath("hailo", "service", "zookeeper", "hosts").AsHostnameArray(2181)
		return connhealthcheck.MaxTcpConnections(nodes, maxconns)()
	}
}
