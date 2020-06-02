package handler

import (
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/discovery-service/registry"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"

	instancesproto "github.com/HailoOSS/discovery-service/proto/instances"
)

// Instances returns instances, optionally just matching an AZ name
func Instances(req *server.Request) (proto.Message, errors.Error) {
	request := req.Data().(*instancesproto.Request)

	instances := registry.AllInstances()
	if az := request.GetAzName(); az != "" {
		instances = instances.Filter(registry.MatchingAz(az))
	}
	if service := request.GetServiceName(); service != "" {
		instances = instances.Filter(registry.MatchingService(service))
	}

	return &instancesproto.Response{
		Instances: instancesToProto(instances),
	}, nil
}
