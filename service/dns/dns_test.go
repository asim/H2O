package dns

import (
	"fmt"
	"net"
	"strings"
	"testing"

	platformtesting "github.com/HailoOSS/platform/testing"
)

func TestDnsHostSuite(t *testing.T) {
	platformtesting.RunSuite(t, new(DnsHostSuite))
}

type DnsHostSuite struct {
	platformtesting.Suite
	mockResolver *MockResolver
	realResolver Resolver
}

func (s *DnsHostSuite) SetupTest() {
	s.Suite.SetupTest()
	s.mockResolver = &MockResolver{}
	s.realResolver = DefaultResolver
	DefaultResolver = s.mockResolver
}

func (s *DnsHostSuite) TearDownTest() {
	s.Suite.TearDownTest()
	DefaultResolver = s.realResolver
}

func (s *DnsHostSuite) TestHostsKnownRole() {
	mockResolver := &MockResolver{}
	DefaultResolver = mockResolver

	// mock success response
	mockResolver.Register("known-role", []net.IP{
		net.ParseIP("10.0.0.1"),
		net.ParseIP("10.0.0.2"),
		net.ParseIP("10.0.0.3"),
	},
		nil)

	ips, err := Hosts("known-role")
	s.Nil(err)

	expectedIps := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}
	s.Equal(strings.Join(expectedIps, " "), strings.Join(ips, " "))
}

func (s *DnsHostSuite) TestHostsUnknownRole() {
	s.mockResolver.Register("unknown-role", []net.IP{}, fmt.Errorf("no such hosts"))

	ips, err := Hosts("unknown-role")
	s.NotNil(err, "Expected error for non existant dns record got response ips: %v err: %v", ips, err)
}
