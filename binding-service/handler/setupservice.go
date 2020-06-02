package handler

import (
	"github.com/HailoOSS/binding-service/binding"
	"github.com/HailoOSS/binding-service/domain"
	servicedown "github.com/HailoOSS/discovery-service/proto/servicedown"
	serviceup "github.com/HailoOSS/discovery-service/proto/serviceup"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
)

// Create bindings on
// - This broker from h2o -> queue
// - All other clusters/brokers to this broker
func ServiceUpListener(req *server.Request) (proto.Message, errors.Error) {
	request := &serviceup.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.kernel.binding.serviceup", err.Error())
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
		return nil, errors.BadRequest("com.HailoOSS.kernel.binding.servicedown", err.Error())
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
