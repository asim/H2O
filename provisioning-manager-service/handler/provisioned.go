package handler

import (
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
	"github.com/hailo-platform/H2O/provisioning-manager-service/cache"
	pproto "github.com/hailo-platform/H2O/provisioning-manager-service/proto/provisioned"
	"github.com/hailo-platform/H2O/provisioning-manager-service/runlevels"
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
