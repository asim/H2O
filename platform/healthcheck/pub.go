package healthcheck

import (
	"github.com/HailoOSS/platform/client"
)

// pubLastSample pings this healthcheck sample out into the ether
func pubLastSample(hc *HealthCheck, ls *Sample) {
	client.Pub("com.HailoOSS.monitor.healthcheck", healthCheckSampleToProto(hc, ls))
}
