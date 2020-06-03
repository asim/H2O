package handler

import (
	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
	"github.com/hailo-platform/H2O/provisioning-manager-service/cache"
	search "github.com/hailo-platform/H2O/provisioning-manager-service/proto/search"
)

func Search(req *server.Request) (proto.Message, errors.Error) {
	log.Infof("Search... %v", req)

	request := &search.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError(server.Name+".search", err.Error())
	}

	services, err := cache.Provisioned(request.GetServiceName(), request.GetMachineClass())
	if err != nil {
		return nil, errors.InternalServerError(server.Name+".search", err.Error())
	}

	rsp := &search.Response{}

	for _, service := range services {
		rsp.Results = append(rsp.Results, &search.Result{
			ServiceName:     service.ServiceName,
			ServiceVersion:  service.ServiceVersion,
			MachineClass:    service.MachineClass,
			NoFileSoftLimit: service.NoFileSoftLimit,
			NoFileHardLimit: service.NoFileHardLimit,
		})
	}

	return rsp, nil
}
