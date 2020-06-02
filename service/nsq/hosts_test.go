package nsq

import (
	"bytes"
	"net"
	"strings"
	"testing"

	platformtesting "github.com/HailoOSS/platform/testing"
	"github.com/HailoOSS/service/config"
	"github.com/HailoOSS/service/dns"
)

func TestRunNsqHostsSuite(t *testing.T) {
	platformtesting.RunSuite(t, new(NsqHostsSuite))
}

type NsqHostsSuite struct {
	platformtesting.Suite
	mockResolver *dns.MockResolver
	realResolver dns.Resolver
}

func (s *NsqHostsSuite) SetupTest() {
	s.Suite.SetupTest()
	s.mockResolver = &dns.MockResolver{}
	s.realResolver = dns.DefaultResolver
	dns.DefaultResolver = s.mockResolver
}

func (s *NsqHostsSuite) TearDownTest() {
	s.Suite.TearDownTest()
	dns.DefaultResolver = s.realResolver
	config.Load(bytes.NewBufferString("{}"))
}

func (s *NsqHostsSuite) TestGetHostsNoConfig() {
	s.mockResolver.Register(
		"nsq-general",
		[]net.IP{net.ParseIP("10.0.0.1")},
		nil,
	)
	hosts := getHosts(4150)

	expectedHosts := []string{"10.0.0.1:4150"}
	s.Equal(strings.Join(expectedHosts, " "), strings.Join(hosts, " "))
}

func (s *NsqHostsSuite) TestGetHostsClusterInConfig() {
	buf := bytes.NewBufferString(`{
		"hailo": {
			"service": {
				"nsq":{
					"cluster": "my-cluster"
				}
			}
		}
	}`)
	config.Load(buf)

	s.mockResolver.Register(
		"nsq-my-cluster",
		[]net.IP{net.ParseIP("10.0.0.1")},
		nil,
	)
	hosts := getHosts(4150)

	expectedHosts := []string{"10.0.0.1:4150"}
	s.Equal(strings.Join(expectedHosts, " "), strings.Join(hosts, " "))
}

func (s *NsqHostsSuite) TestDiffHosts() {
	testCases := []struct {
		a, b, expect []string
	}{
		{
			[]string{},
			[]string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
			[]string{},
		},
		{
			[]string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
			[]string{},
			[]string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
		},
		{
			[]string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
			[]string{"10.0.0.1"},
			[]string{"10.0.0.2", "10.0.0.3"},
		},
		{
			[]string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
			[]string{"10.0.0.1", "10.0.0.2", "10.0.0.4", "10.0.0.5"},
			[]string{"10.0.0.3"},
		},
	}

	for _, test := range testCases {
		hosts := diffHosts(test.a, test.b)
		s.Equal(strings.Join(test.expect, " "), strings.Join(hosts, " "))
	}
}
