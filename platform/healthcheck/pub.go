package healthcheck

import (
	"github.com/hailo-platform/H2O/platform/client"
)

// pubLastSample pings this healthcheck sample out into the ether
func pubLastSample(hc *HealthCheck, ls *Sample) {
	client.Pub("com.hailocab.monitor.healthcheck", healthCheckSampleToProto(hc, ls))
}
