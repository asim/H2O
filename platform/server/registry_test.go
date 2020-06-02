package server

import "testing"

func TestWeCannotAddEmptyEndpoint(t *testing.T) {
	reg := newRegistry()
	ep := &Endpoint{}
	err := reg.add(ep)

	if err == nil {
		t.Error("Should not be allowed to add empty endpoint")
	}
}

func TestEndpointNamesShouldBeLowercase(t *testing.T) {
	reg := newRegistry()
	ep := &Endpoint{Name: "MixedCase"}
	err := reg.add(ep)

	if err == nil {
		t.Error("Should not be allowed to add endpoints with uppercase letters")
	}
}

func TestRegister(t *testing.T) {
	reg := newRegistry()

	if len(reg.endpoints) > 0 {
		t.Error("Intial registry shound be empty")
	}

	ep := &Endpoint{
		Name: "test",
	}
	reg.add(ep)

	if len(reg.endpoints) != 1 {
		t.Error("Should be 1 entry in the registry")
	}
}

func TestFind(t *testing.T) {
	reg := newRegistry()
	ep := &Endpoint{
		Name: "test",
	}
	reg.add(ep)

	if _, ok := reg.find(ep.Name); !ok {
		t.Error("Unable to lookup endpoint")
	}

	if _, ok := reg.find("Unknown"); ok {
		t.Error("Found invalid entry")
	}
}

func TestIterate(t *testing.T) {
	endpoints := make(map[string]*Endpoint)

	reg := newRegistry()
	ep1 := &Endpoint{
		Name: "test 1",
	}
	reg.add(ep1)
	ep2 := &Endpoint{
		Name: "test 2",
	}
	reg.add(ep2)

	for _, ep := range reg.iterate() {
		endpoints[ep.Name] = ep
	}

	if len(endpoints) != 2 {
		t.Error("Too many items returned in interator")
	}

	if ep, ok := endpoints[ep1.Name]; !ok || ep != ep1 {
		t.Error("Missing ep1")
	}

	if ep, ok := endpoints[ep2.Name]; !ok || ep != ep2 {
		t.Error("Missing ep2")
	}
}
