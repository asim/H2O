package zookeeper

import (
	"bytes"
	"net"
	"testing"

	platformtesting "github.com/HailoOSS/platform/testing"
	"github.com/HailoOSS/service/config"
	"github.com/HailoOSS/service/dns"
)

func TestZkHostsSuite(t *testing.T) {
	platformtesting.RunSuite(t, new(ZkHostsSuite))
}

type ZkHostsSuite struct {
	platformtesting.Suite
	mockResolver *dns.MockResolver
	realResolver dns.Resolver
}

func (s *ZkHostsSuite) SetupTest() {
	s.mockResolver = &dns.MockResolver{}
	s.realResolver = dns.DefaultResolver
	dns.DefaultResolver = s.mockResolver
}

func (s *ZkHostsSuite) TearDownTest() {
	dns.DefaultResolver = s.realResolver
	config.Load(bytes.NewBufferString("{}"))
}

func (s *ZkHostsSuite) TestGetHostsNoConfig() {
	s.mockResolver.Register(
		"zookeeper-general",
		[]net.IP{net.ParseIP("10.0.0.1")},
		nil,
	)
	hosts := getHosts()

	s.Len(hosts, 1)
	s.Equal(hosts[0], "10.0.0.1:2181")
}

func (s *ZkHostsSuite) TestGetHostsServersInConfig() {
	buf := bytes.NewBufferString(`{
		"hailo": {
			"service": {
				"zookeeper":{
					"hosts": ["10.0.0.1:2181"]
				}
			}
		}
	}`)
	config.Load(buf)

	hosts := getHosts()

	s.Len(hosts, 1)
	s.Equal(hosts[0], "10.0.0.1:2181")
}

func (s *ZkHostsSuite) TestGetHostsTierInConfig() {
	buf := bytes.NewBufferString(`{
		"hailo": {
			"service": {
				"zookeeper":{
					"tier": "my-tier"
				}
			}
		}
	}`)
	config.Load(buf)

	s.mockResolver.Register(
		"zookeeper-my-tier",
		[]net.IP{net.ParseIP("10.0.0.1")},
		nil,
	)
	hosts := getHosts()

	s.Len(hosts, 1)
	s.Equal(hosts[0], "10.0.0.1:2181")
}

func (s *ZkHostsSuite) TestGetHostsTierAndServersInConfig() {
	buf := bytes.NewBufferString(`{
		"hailo": {
			"service": {
				"zookeeper":{
					"tier": "my-tier",
					"hosts": {
						"my-tier": ["10.0.0.1:2181"]
					}
				}
			}
		}
	}`)
	config.Load(buf)

	hosts := getHosts()

	s.Len(hosts, 1)
	s.Equal(hosts[0], "10.0.0.1:2181")
}
