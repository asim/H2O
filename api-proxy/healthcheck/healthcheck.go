package healthcheck

import (
	"fmt"
	"os"
	"time"

	"github.com/hailo-platform/H2O/api-proxy/stats"
	"github.com/hailo-platform/H2O/api-proxy/statusmonitor"
	"github.com/hailo-platform/H2O/platform/healthcheck"
	"github.com/hailo-platform/H2O/platform/server"
	hc "github.com/hailo-platform/H2O/service/healthcheck"
)

var (
	Hostname string
)

func Init(monitor *statusmonitor.StatusMonitor) {
	Hostname, _ = os.Hostname()

	// Add azstatus healthcheck that exposes the status of this api instance
	healthcheck.Register(&healthcheck.HealthCheck{
		Id:             "com.hailo-platform/H2O.kernel.azstatus",
		ServiceName:    server.Name,
		ServiceVersion: server.Version,
		Hostname:       Hostname,
		InstanceId:     server.InstanceID,
		Interval:       time.Minute,
		Checker:        monitor.AzStatusChecker,
		Priority:       hc.Pager,
	})
}

func RegisterErrorRateCheck(id, ep string, errRate float32) {
	Register(id, func() (map[string]string, error) {
		srate, erate := stats.Rate1(ep)
		ret := make(map[string]string)
		ret["errorRate1"] = fmt.Sprintf("%.2f%%", erate*100.0)
		ret["successRate1"] = fmt.Sprintf("%.2f%%", srate*100.0)

		if erate > errRate {
			return ret, fmt.Errorf("1 minute error rate %.2f exceeds threshold %.2f%%", erate*100.00, errRate*100.00)
		}

		return ret, nil
	})
}

func RegisterDropOffCheck(id, ep string, dropOff float32) {
	Register(id, func() (map[string]string, error) {
		oldRate := stats.GetMetric(ep + ".totalRate1")
		newRate := stats.Rate1Total(ep)
		stats.SaveMetric(ep+".totalRate1", newRate)

		var dev float32

		if newRate != 0.0 && oldRate != 0.0 {
			dev = newRate / oldRate
		}

		ret := make(map[string]string)

		if newRate > 0.0 {
			newRate = newRate / 100.0
		}

		ret["rate1Total"] = fmt.Sprintf("%.2f", newRate)

		switch {
		case dev == 0.0, dev == 1.0:
			// No deviation
			ret["rate1DropOff"] = "0.0%"
			ret["rate1Increase"] = "0.0%"

			// Receiving no requests
			if newRate == 0.0 {
				return ret, fmt.Errorf("Request rate is 0.0%%")
			}
		case dev < 1.0:
			// Request rate has dropped
			ret["rate1DropOff"] = fmt.Sprintf("%.2f%%", (1.0-dev)*100.0)
			ret["rate1Increase"] = "0.0%"
		default:
			// Request rate has increased
			ret["rate1DropOff"] = "0.0%"
			ret["rate1Increase"] = fmt.Sprintf("%.2f%%", (dev-1.0)*100.0)
		}

		if drop := (1.0 - dev); dev < 1.0 && drop > dropOff {
			return ret, fmt.Errorf("1 minute dropoff rate %.2f exceeds threshold %.2f%%", drop*100.0, dropOff*100.0)
		}

		return ret, nil
	})
}

func Register(id string, checker hc.Checker) {
	healthcheck.Register(&healthcheck.HealthCheck{
		Id:             id,
		ServiceName:    server.Name,
		ServiceVersion: server.Version,
		Hostname:       Hostname,
		InstanceId:     server.InstanceID,
		Interval:       time.Minute,
		Checker:        checker,
		Priority:       hc.Warning,
	})
}
