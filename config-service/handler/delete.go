package handler

import (
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/config-service/domain"
	del "github.com/HailoOSS/config-service/proto/delete"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	gouuid "github.com/nu7hatch/gouuid"
)

// Delete will remove the config from a single ID, optionally at some path
func Delete(req *server.Request) (proto.Message, errors.Error) {
	request := &del.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.config.delete", fmt.Sprintf("%v", err))
	}

	previousConfig, _, err := domain.ReadConfig(request.GetId(), request.GetPath())
	if err != nil {
		log.Warnf("Unable to read previous config on delete: %s", err.Error())
	}

	u4, err := gouuid.NewV4()
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.config.delete.genid", fmt.Sprintf("%v", err))
	}

	err = domain.DeleteConfig(
		u4.String(),
		request.GetId(),
		request.GetPath(),
		req.Auth().AuthUser().Mech,
		req.Auth().AuthUser().Id,
		request.GetMessage(),
	)
	if err == domain.ErrPathNotFound {
		return nil, errors.NotFound("com.HailoOSS.service.config.delete", fmt.Sprintf("%v", err))
	}
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.config.delete", fmt.Sprintf("%v", err))
	}

	broadcastChange(request.GetId())

	// Pub the change to the platform event stream
	pubNSQEvent("DELETED", u4.String(), request.GetId(), request.GetPath(), req.Auth().AuthUser().Mech,
		req.Auth().AuthUser().Id, request.GetMessage(), "", string(previousConfig))

	return &del.Response{}, nil
}
