package dns

import (
	"net"
)

type Resolver interface {
	LookupIP(string) ([]net.IP, error)
}

type resolver struct{}

func (r *resolver) LookupIP(name string) ([]net.IP, error) {
	return net.LookupIP(name)
}

func newResolver() *resolver {
	return &resolver{}
}
