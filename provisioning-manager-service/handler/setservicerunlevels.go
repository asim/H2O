package handler

import (
	"fmt"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
	"github.com/HailoOSS/provisioning-manager-service/dao"
	"github.com/HailoOSS/provisioning-manager-service/event"
	srlproto "github.com/HailoOSS/provisioning-manager-service/proto/setservicerunlevels"
	"github.com/HailoOSS/provisioning-manager-service/runlevels"
)

func SetServiceRunLevels(req *server.Request) (proto.Message, errors.Error) {
	request := &srlproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning-manager.setservicerunlevels", fmt.Sprintf("%v", err))
	}

	service := request.GetServiceName()
	levels := request.GetLevels()

	if len(service) == 0 {
		return nil, errors.BadRequest("com.HailoOSS.provisioning-manager.setservicerunlevels", "Service Name cannot be blank")
	}

	var runLevels [6]bool
	var stringLevels []string

	for _, level := range levels {
		if level < runlevels.MinRunLevel || level > runlevels.MaxRunLevel {
			return nil, errors.BadRequest("com.HailoOSS.provisioning-manager.setservicerunlevels", fmt.Sprintf("Invalid run level of %d provided", level))
		}

		runLevels[level] = true
		stringLevels = append(stringLevels, level.String())
	}

	err := dao.SetServiceRunLevels(service, runLevels)
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning-manager.setservicerunlevels", fmt.Sprintf("%v", err))
	}

	event.SetServiceRunLevels(service, stringLevels, req.Auth().AuthUser().Id)

	return &srlproto.Response{}, nil
}
