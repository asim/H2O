package statusmonitor

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

var (
	hosts = []string{"google.com:80", "yahoo.com:80", "bing.com:80"}
)

// Handler returns the AZ health as reported by the StatusMonitor
func (s *StatusMonitor) Handler(rw http.ResponseWriter, r *http.Request) {
	status := s.IsHealthy

	if !status {
		rw.WriteHeader(int(http.StatusInternalServerError))
	}
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")

	fmt.Fprint(rw, JsonResponse{"IsHealthy": status})
}

// StatusHandler synchronously establishes external connections to determine
// the health of the network. If all attempts fail it will return a 500 status.
func StatusHandler(rw http.ResponseWriter, r *http.Request) {
	ok := len(hosts)
	var info string

	for _, host := range hosts {
		c, err := net.DialTimeout("tcp", host, time.Second)
		if err != nil {
			ok--
			info = err.Error()
			continue
		}
		c.Close()
	}

	if ok == 0 {
		errorMsg := fmt.Sprintf("Error establishing external connections: %s", info)
		http.Error(rw, errorMsg, http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	fmt.Fprintf(rw, "OK")
}

// AzStatusChecker checks the status of the default status monitor
func (s *StatusMonitor) AzStatusChecker() (map[string]string, error) {
	ret := make(map[string]string)
	status := s.IsHealthy
	az := s.AZName
	changed := s.LastChanged.Format("2006-01-02 15:04:05")
	ret["azName"] = fmt.Sprintf("%v", az)
	ret["isHealthy"] = fmt.Sprintf("%v", status)
	ret["failureType"] = fmt.Sprintf("%v", s.FailureType.String())
	ret["lastChanged"] = fmt.Sprintf("%v", changed)

	if !status {
		return ret, fmt.Errorf("This Thin API instance is not serving traffic - AZ reported unhealthy")
	}
	if az == "undefined" {
		return ret, fmt.Errorf("Unable to determine the local AZ")
	}
	return ret, nil
}
