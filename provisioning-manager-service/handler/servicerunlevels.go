package handler

import (
	"fmt"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
	levels "github.com/hailo-platform/H2O/provisioning-manager-service/proto"
	srlproto "github.com/hailo-platform/H2O/provisioning-manager-service/proto/servicerunlevels"
	"github.com/hailo-platform/H2O/provisioning-manager-service/runlevels"
)

func ServiceRunLevels(req *server.Request) (proto.Message, errors.Error) {
	request := &srlproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.provisioning-manager.servicerunlevels", fmt.Sprintf("%v", err))
	}

	rsp := &srlproto.Response{}
	service := request.GetServiceName()

	for name, serviceLevels := range runlevels.Services() {
		if len(service) > 0 && service != name {
			continue
		}

		var runLevels []levels.Level

		for level, active := range serviceLevels {
			if !active {
				continue
			}

			runLevels = append(runLevels, levels.Level(level))
		}

		rsp.ServiceRunLevels = append(rsp.ServiceRunLevels, &srlproto.ServiceRunLevels{
			ServiceName: proto.String(name),
			Levels:      runLevels,
		})
	}

	return rsp, nil
}
