package handler

import (
	"fmt"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
	"github.com/hailo-platform/H2O/provisioning-manager-service/dao"
	"github.com/hailo-platform/H2O/provisioning-manager-service/event"
	srlproto "github.com/hailo-platform/H2O/provisioning-manager-service/proto/setrunlevel"
	"github.com/hailo-platform/H2O/provisioning-manager-service/runlevels"
)

func SetRunLevel(req *server.Request) (proto.Message, errors.Error) {
	request := &srlproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.provisioning-manager.setrunlevel", fmt.Sprintf("%v", err))
	}

	region := request.GetRegion()
	level := request.GetLevel()

	if len(region) == 0 {
		return nil, errors.BadRequest("com.hailo-platform/H2O.provisioning-manager.setrunlevel", "Region cannot be blank")
	}

	if level < runlevels.MinRunLevel || level > runlevels.MaxRunLevel {
		return nil, errors.BadRequest("com.hailo-platform/H2O.provisioning-manager.setrunlevel", "Invalid run level")
	}

	err := dao.SetRunLevel(region, int64(level))
	if err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.provisioning-manager.setrunlevel", fmt.Sprintf("%v", err))
	}

	event.SetRegionRunLevel(region, level.String(), req.Auth().AuthUser().Id)

	return &srlproto.Response{}, nil
}
