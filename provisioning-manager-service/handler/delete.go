package handler

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
	"github.com/HailoOSS/provisioning-manager-service/dao"
	"github.com/HailoOSS/provisioning-manager-service/event"
	delete "github.com/HailoOSS/provisioning-manager-service/proto/delete"
)

func Delete(req *server.Request) (proto.Message, errors.Error) {
	log.Infof("Delete... %v", req)

	request := &delete.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError(server.Name+".delete", fmt.Sprintf("%v", err))
	}

	recToDel := &dao.Service{
		ServiceName:    request.GetServiceName(),
		ServiceVersion: request.GetServiceVersion(),
		MachineClass:   request.GetMachineClass(),
	}

	if err := dao.Delete(recToDel); err != nil {
		return nil, errors.InternalServerError(server.Name+".delete", fmt.Sprintf("%v", err))
	}

	// Pub an event
	event.DeprovisionedToNSQ(request.GetServiceName(), request.GetServiceVersion(), request.GetMachineClass(), req.Auth().AuthUser().Id)

	return &delete.Response{}, nil
}
