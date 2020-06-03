package handler

import (
	"github.com/hailo-platform/H2O/binding-service/binding"
	"github.com/hailo-platform/H2O/binding-service/domain"
	servicedown "github.com/hailo-platform/H2O/discovery-service/proto/servicedown"
	serviceup "github.com/hailo-platform/H2O/discovery-service/proto/serviceup"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
)

// Create bindings on
// - This broker from h2o -> queue
// - All other clusters/brokers to this broker
func ServiceUpListener(req *server.Request) (proto.Message, errors.Error) {
	request := &serviceup.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.hailocab.kernel.binding.serviceup", err.Error())
	}
	errObj := binding.SetupService(domain.ServiceFromServiceupProto(request))
	if errObj != nil {
		return nil, errObj
	}
	return &serviceup.Response{}, nil
}

// Remove bindings on
// - This cluster, h2o ->  instance id
// - All other clusters/brokers to this broker. Only needed if we have no more instances of that service in this AZ
func ServiceDownListener(req *server.Request) (proto.Message, errors.Error) {
	request := &servicedown.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.hailocab.kernel.binding.servicedown", err.Error())
	}
	queue := request.GetInstanceId()
	service := request.GetServiceName()
	azname := request.GetAzName()
	errObj := binding.TeardownService(service, queue, azname)
	if errObj != nil {
		return nil, errObj
	}
	return &servicedown.Response{}, nil

}
