package handler

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
	"github.com/hailo-platform/H2O/provisioning-service/process"
	restartaz "github.com/hailo-platform/H2O/provisioning-service/proto/restartaz"
)

func RestartAZ(req *server.Request) (proto.Message, errors.Error) {
	log.Infof("Restart az... %v", req)

	request := &restartaz.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.provisioning.handler.restartaz", fmt.Sprintf("%v", err))
	}

	if err := process.RestartAZ(request.GetAzName()); err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.provisioning.handler.restart", fmt.Sprintf("%v", err))
	}

	return &restartaz.Response{}, nil
}
