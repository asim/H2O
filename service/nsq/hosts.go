package nsq

import (
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/service/config"
	"github.com/HailoOSS/service/dns"
)

// diffHosts return hosts which exist in A and do not exist in B.
// Used to disconnect from nsq hosts that are no longer in config.
func diffHosts(hostsA, hostsB []string) []string {
	var delta []string
	for _, host := range hostsA {
		var seen bool
		for _, ahost := range hostsB {
			if ahost == host {
				seen = true
				break
			}
		}
		if !seen {
			delta = append(delta, host)
		}
	}
	return delta
}

func getHosts(port int, path ...string) []string {
	if hosts := config.AtPath(path...).AsHostnameArray(port); len(hosts) > 0 {
		return hosts
	}

	// should we lookup dns?
	if config.AtPath("hailo", "service", "nsq", "disableDnsLookup").AsBool() {
		return []string{}
	}

	// try dns lookup
	cluster := config.AtPath("hailo", "service", "nsq", "cluster").AsString("general")
	hosts, err := dns.Hosts("nsq-" + cluster)
	if err != nil {
		log.Errorf("Failed to load NSQ hosts from dns: %v", err)
		return []string{}
	}

	// append port
	for i, host := range hosts {
		hosts[i] = fmt.Sprintf("%s:%d", host, port)
	}

	return hosts
}
