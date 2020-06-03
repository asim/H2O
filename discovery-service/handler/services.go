package handler

import (
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/discovery-service/registry"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"

	servicesproto "github.com/hailo-platform/H2O/discovery-service/proto/services"
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
