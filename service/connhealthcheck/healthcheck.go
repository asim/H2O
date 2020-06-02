package connhealthcheck

import (
	"fmt"
	"github.com/HailoOSS/go-hailo-lib/proc"
	"github.com/HailoOSS/service/healthcheck"
)

const HealthCheckId = "com.HailoOSS.service.tcpconn"

// MaxTcpConnections inspects the number of established TCP connections are being made by this
// process for a list of hosts. If the aggregate exceeds maxconns then an error will be raised.
func MaxTcpConnections(hosts []string, maxconns int) healthcheck.Checker {
	return func() (map[string]string, error) {
		var conns int
		ret := make(map[string]string)

		for _, host := range hosts {
			if _, ok := ret[host]; ok {
				continue
			}

			c := proc.CachedNumRemoteTcpConns(host)
			ret[host] = fmt.Sprintf("%d", c)
			conns += c
		}

		ret["total_conns"] = fmt.Sprintf("%d", conns)

		if conns > maxconns {
			return ret, fmt.Errorf("Number of connections %d exceeds threshold of %d", conns, maxconns)
		}

		return ret, nil
	}
}

// TcpConnections returns all the remote hosts and the number of connections to each.
// If any exceeds the threshold it will raise an error.
func TcpConnections(threshold int) healthcheck.Checker {
	return func() (map[string]string, error) {
		conns := proc.CachedRemoteTcpConns()

		ret := make(map[string]string)
		found := 0
		for host, c := range conns {
			ret[host] = fmt.Sprintf("%d", c)
			if c > threshold {
				found++
			}
		}

		if found > 0 {
			return ret, fmt.Errorf("Found %d remote host connections exceeding threshold of %d", found, threshold)
		}

		return ret, nil
	}
}
