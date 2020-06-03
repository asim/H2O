package handler

import (
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/discovery-service/registry"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"

	instancesproto "github.com/hailo-platform/H2O/discovery-service/proto/instances"
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
