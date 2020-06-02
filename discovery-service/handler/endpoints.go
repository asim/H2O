package handler

import (
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/discovery-service/registry"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"

	endpointsproto "github.com/HailoOSS/discovery-service/proto/endpoints"
)

// Endpoints returns all endpoints discovered (optionally matching a given service name),
// for all versions of the service
// Keep in mind that there can be upwards of a few thousand of these, so use this
// sparingly if you aren't supplying a service
func Endpoints(req *server.Request) (proto.Message, errors.Error) {
	request := req.Data().(*endpointsproto.Request)

	instances := registry.AllInstances()
	if service := request.GetService(); service != "" {
		instances = instances.Filter(registry.MatchingService(service))
	}

	return &endpointsproto.Response{
		Endpoints: instancesToEndpointsProto(instances),
	}, nil
}
