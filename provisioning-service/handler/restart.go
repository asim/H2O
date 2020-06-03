package handler

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
	"github.com/hailo-platform/H2O/provisioning-service/event"
	"github.com/hailo-platform/H2O/provisioning-service/process"
	restart "github.com/hailo-platform/H2O/provisioning-service/proto/restart"
)

func Restart(req *server.Request) (proto.Message, errors.Error) {
	log.Infof("Restart... %v", req)

	request := &restart.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.provisioning.handler.restart", fmt.Sprintf("%v", err))
	}

	if err := process.Restart(request.GetServiceName(), request.GetServiceVersion(), request.GetAzName()); err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.provisioning.handler.restart", fmt.Sprintf("%v", err))
	}

	// Pub an event
	event.RestartedToNSQ(request.GetServiceName(), request.GetServiceVersion(), req.Auth().AuthUser().Id)

	return &restart.Response{}, nil
}
