package dns

import (
	"fmt"
	"sort"

	"github.com/HailoOSS/platform/util"
)

const (
	domain = "HailoOSS.net"
	scope  = "i"
)

var (
	DefaultResolver Resolver = newResolver()
)

func hostName(role string) string {
	region := util.GetAwsRegionName()
	env := util.GetEnvironmentName()
	return fmt.Sprintf("%s.%s.%s.%s.%s", role, region, scope, env, domain)
}

// Hosts returns a list of ip addresses for a particular role.
func Hosts(role string) ([]string, error) {
	name := hostName(role)

	ips, err := DefaultResolver.LookupIP(name)
	if err != nil {
		return nil, err
	}

	var hosts []string

	for _, ip := range ips {
		hosts = append(hosts, ip.String())
	}

	sort.Strings(hosts)

	return hosts, nil
}
