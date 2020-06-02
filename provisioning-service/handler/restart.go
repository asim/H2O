package handler

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
	"github.com/HailoOSS/provisioning-service/event"
	"github.com/HailoOSS/provisioning-service/process"
	restart "github.com/HailoOSS/provisioning-service/proto/restart"
)

func Restart(req *server.Request) (proto.Message, errors.Error) {
	log.Infof("Restart... %v", req)

	request := &restart.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning.handler.restart", fmt.Sprintf("%v", err))
	}

	if err := process.Restart(request.GetServiceName(), request.GetServiceVersion(), request.GetAzName()); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning.handler.restart", fmt.Sprintf("%v", err))
	}

	// Pub an event
	event.RestartedToNSQ(request.GetServiceName(), request.GetServiceVersion(), req.Auth().AuthUser().Id)

	return &restart.Response{}, nil
}
