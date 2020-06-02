package handler

import (
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
	"github.com/HailoOSS/provisioning-manager-service/cache"
	pproto "github.com/HailoOSS/provisioning-manager-service/proto/provisioned"
	"github.com/HailoOSS/provisioning-manager-service/runlevels"
)

func Provisioned(req *server.Request) (proto.Message, errors.Error) {
	request := &pproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest(server.Name+".provisioned", err.Error())
	}

	services, err := cache.Provisioned(request.GetServiceName(), request.GetMachineClass())
	if err != nil {
		return nil, errors.InternalServerError(server.Name+".provisioned", err.Error())
	}

	return &pproto.Response{
		Services: runlevels.Filter(services),
	}, nil
}
