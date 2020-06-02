package handler

import (
	"fmt"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
	"github.com/HailoOSS/provisioning-manager-service/dao"
	"github.com/HailoOSS/provisioning-manager-service/event"
	srlproto "github.com/HailoOSS/provisioning-manager-service/proto/setrunlevel"
	"github.com/HailoOSS/provisioning-manager-service/runlevels"
)

func SetRunLevel(req *server.Request) (proto.Message, errors.Error) {
	request := &srlproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning-manager.setrunlevel", fmt.Sprintf("%v", err))
	}

	region := request.GetRegion()
	level := request.GetLevel()

	if len(region) == 0 {
		return nil, errors.BadRequest("com.HailoOSS.provisioning-manager.setrunlevel", "Region cannot be blank")
	}

	if level < runlevels.MinRunLevel || level > runlevels.MaxRunLevel {
		return nil, errors.BadRequest("com.HailoOSS.provisioning-manager.setrunlevel", "Invalid run level")
	}

	err := dao.SetRunLevel(region, int64(level))
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning-manager.setrunlevel", fmt.Sprintf("%v", err))
	}

	event.SetRegionRunLevel(region, level.String(), req.Auth().AuthUser().Id)

	return &srlproto.Response{}, nil
}
