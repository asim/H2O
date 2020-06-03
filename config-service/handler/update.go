package handler

import (
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/config-service/domain"
	update "github.com/hailo-platform/H2O/config-service/proto/update"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	gouuid "github.com/nu7hatch/gouuid"
)

const (
	defaultMech = "s2s"
)

// Update will completely replace configuration at this level in the path with the supplied config (for the given ID)
func Update(req *server.Request) (proto.Message, errors.Error) {
	request := &update.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.hailo-platform/H2O.service.config.update", fmt.Sprintf("%v", err))
	}

	u4, err := gouuid.NewV4()
	if err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.service.config.update.genid", fmt.Sprintf("%v", err))
	}

	var mech, id string
	if user := req.Auth().AuthUser(); user != nil {
		mech = user.Mech
		id = user.Id
	} else {
		mech = defaultMech
		id = req.From()
	}

	previousConfig, _, err := domain.ReadConfig(request.GetId(), request.GetPath())
	if err != nil {
		log.Warnf("Unable to read previous config on update: %s", err.Error())
	}

	err = domain.CreateOrUpdateConfig(
		u4.String(),
		request.GetId(),
		request.GetPath(),
		mech,
		id,
		request.GetMessage(),
		[]byte(request.GetConfig()),
	)
	if err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.service.config.update", fmt.Sprintf("%v", err))
	}

	if !request.GetNoReload() {
		broadcastChange(request.GetId())

		// Pub the change to the platform event stream
		pubNSQEvent("UPDATED", u4.String(), request.GetId(), request.GetPath(), mech, id, request.GetMessage(), request.GetConfig(), string(previousConfig))
	}

	return &update.Response{}, nil
}
