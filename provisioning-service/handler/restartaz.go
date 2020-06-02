package handler

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
	"github.com/HailoOSS/provisioning-service/process"
	restartaz "github.com/HailoOSS/provisioning-service/proto/restartaz"
)

func RestartAZ(req *server.Request) (proto.Message, errors.Error) {
	log.Infof("Restart az... %v", req)

	request := &restartaz.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning.handler.restartaz", fmt.Sprintf("%v", err))
	}

	if err := process.RestartAZ(request.GetAzName()); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning.handler.restart", fmt.Sprintf("%v", err))
	}

	return &restartaz.Response{}, nil
}
