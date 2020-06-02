package handler

import (
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/discovery-service/registry"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"

	servicesproto "github.com/HailoOSS/discovery-service/proto/services"
)

// Services returns a list of services running in the region
func Services(req *server.Request) (proto.Message, errors.Error) {
	request := req.Data().(*servicesproto.Request)

	instances := registry.AllInstances()
	if service := request.GetService(); service != "" {
		instances = instances.Filter(registry.MatchingService(service))
	}

	return &servicesproto.Response{
		Services: instancesToServicesProto(instances),
	}, nil
}
