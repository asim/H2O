package handler

import (
	"fmt"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
	"github.com/HailoOSS/provisioning-manager-service/dao"
	levels "github.com/HailoOSS/provisioning-manager-service/proto"
	rlproto "github.com/HailoOSS/provisioning-manager-service/proto/runlevels"
	"github.com/HailoOSS/provisioning-manager-service/runlevels"
)

func RunLevels(req *server.Request) (proto.Message, errors.Error) {
	request := &rlproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning-manager.runlevels", fmt.Sprintf("%v", err))
	}

	runLevels, err := dao.ReadRunLevels()
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning-manager.runlevels", fmt.Sprintf("%v", err))
	}

	defaultLevel := levels.Level(runlevels.DefaultRunLevel)

	rsp := &rlproto.Response{
		DefaultLevel: &defaultLevel,
	}

	for _, runLevel := range runLevels {
		level := levels.Level(runLevel.Level)
		rsp.RunLevels = append(rsp.RunLevels, &rlproto.RunLevel{
			Region: proto.String(runLevel.Region),
			Level:  &level,
		})
	}

	return rsp, nil
}
