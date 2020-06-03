// +build integration

// This test does end-to-end calls to an environment
// The whole point of the HTTP caller is that it _just works_ *clicks fingers*
// That is, it should work, whether the client request is a JSON request or
// a proto request. Nearly every request internally (that you are likely to be
// using, dear reader), is a proto request - so it's very important that works.
// We make use of the fact the THIN API allows _both types_ of request to be
// made to it.

package multiclient

import (
	"github.com/hailo-platform/H2O/platform/client"

	"github.com/hailo-platform/H2O/protobuf/proto"

	servicesproto "github.com/hailo-platform/H2O/discovery-service/proto/services"
	searchproto "github.com/hailo-platform/H2O/zoning-service/proto/search"
)

import (
	"testing"
)

func TestJsonHttpCall(t *testing.T) {
	// zoning service is OpenToTheWorld and should have _some_ zone surrounding Somerset House
	json := []byte(`{"location":{"lat":51.510761,"lng":-0.1174437}}`)
	req, err := client.NewJsonRequest("com.hailocab.service.zoning", "search", json)
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}
	rsp := &searchproto.Response{}

	caller := HttpCaller("https://api2-staging.elasticride.com")
	perr := caller(req, rsp)

	if perr != nil {
		t.Fatalf("Error executing request: %v", perr)
	}

	if len(rsp.GetZones()) == 0 {
		t.Error("Expecting > 0 zones, got back 0")
	}
}

func TestJsonHttpCallThatFails(t *testing.T) {
	// discovery service is NOT open to the world, because it's a kernel service
	req, err := client.NewJsonRequest("com.hailocab.kernel.discovery", "services", []byte(`{}`))
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}
	rsp := &servicesproto.Response{}

	caller := HttpCaller("https://api2-staging.elasticride.com")
	perr := caller(req, rsp)

	if perr == nil {
		t.Fatal("We are EXPECTING as error executing request")
	}

	if perr.Code() != "com.hailocab.api.rpc.auth" {
		t.Errorf("Expecting code 'com.hailocab.api.rpc.auth' got '%s'", perr.Code())
	}
	if perr.Type() != "FORBIDDEN" {
		t.Errorf("Expecting type 'FORBIDDEN' got '%s'", perr.Type())
	}
}

func TestProtoHttpCall(t *testing.T) {
	// zoning service is OpenToTheWorld and should have _some_ zone surrounding Somerset House
	req, err := client.NewRequest("com.hailocab.service.zoning", "search", &searchproto.Request{
		Location: &searchproto.LatLng{
			Lat: proto.Float64(51.510761),
			Lng: proto.Float64(-0.1174437),
		},
	})
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}
	rsp := &searchproto.Response{}

	caller := HttpCaller("https://api2-staging.elasticride.com")
	perr := caller(req, rsp)

	if perr != nil {
		t.Fatalf("Error executing request: %v", perr)
	}

	if len(rsp.GetZones()) == 0 {
		t.Error("Expecting > 0 zones, got back 0")
	}
}

func TestProtoHttpCallThatFails(t *testing.T) {
	// discovery service is NOT open to the world, because it's a kernel service
	req, err := client.NewRequest("com.hailocab.kernel.discovery", "services", &servicesproto.Request{})
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}
	rsp := &servicesproto.Response{}

	caller := HttpCaller("https://api2-staging.elasticride.com")
	perr := caller(req, rsp)

	if perr == nil {
		t.Fatal("We are EXPECTING as error executing request")
	}

	if perr.Code() != "com.hailocab.api.rpc.auth" {
		t.Errorf("Expecting code 'com.hailocab.api.rpc.auth' got '%s'", perr.Code())
	}
	if perr.Type() != "FORBIDDEN" {
		t.Errorf("Expecting type 'FORBIDDEN' got '%s'", perr.Type())
	}
}
