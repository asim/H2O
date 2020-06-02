package dns

import (
	"net"

	"github.com/stretchr/testify/mock"
)

type MockResolver struct {
	mock.Mock
}

func (mr *MockResolver) Register(role string, ips []net.IP, err error) {
	name := hostName(role)
	mr.Mock.On("LookupIP", name).Return(ips, err)
}

func (mr *MockResolver) LookupIP(name string) ([]net.IP, error) {
	args := mr.Mock.Called(name)
	return args.Get(0).([]net.IP), args.Error(1)
}
