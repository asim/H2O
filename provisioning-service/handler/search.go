package handler

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
	pproto "github.com/hailo-platform/H2O/provisioning-manager-service/proto/provisioned"
	dao "github.com/hailo-platform/H2O/provisioning-service/dao"
	search "github.com/hailo-platform/H2O/provisioning-service/proto/search"
)

func provisioned(req *server.Request, serviceName, machineClass string) (dao.ProvisionedServices, error) {
	pReq := &pproto.Request{}

	if len(serviceName) > 0 {
		pReq.ServiceName = proto.String(serviceName)
	}

	if len(machineClass) > 0 {
		pReq.MachineClass = proto.String(machineClass)
	}

	request, err := req.ScopedRequest("com.hailocab.kernel.provisioning-manager", "provisioned", pReq)
	if err != nil {
		return nil, errors.InternalServerError("com.hailocab.provisioning.handler.search", fmt.Sprintf("%v", err))
	}

	response := &pproto.Response{}
	if err := client.Req(request, response); err != nil {
		return nil, errors.InternalServerError("com.hailocab.provisioning.handler.search", fmt.Sprintf("%v", err))
	}

	var provisioned dao.ProvisionedServices
	for _, service := range response.GetServices() {
		provisioned = append(provisioned, &dao.ProvisionedService{
			ServiceName:     service.GetServiceName(),
			ServiceVersion:  service.GetServiceVersion(),
			MachineClass:    service.GetMachineClass(),
			NoFileSoftLimit: service.GetNoFileSoftLimit(),
			NoFileHardLimit: service.GetNoFileHardLimit(),
		})
	}
	return provisioned, nil
}

func Search(req *server.Request) (proto.Message, errors.Error) {
	log.Infof("Search... %v", req)

	request := &search.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.hailocab.provisioning.handler.search", fmt.Sprintf("%v", err))
	}

	rows, err := provisioned(req, request.GetServiceName(), request.GetMachineClass())
	if err != nil {
		return nil, errors.InternalServerError("com.hailocab.provisioning.handler.search", fmt.Sprintf("%v", err))
	}

	results := make([]*search.Result, len(rows))
	for i, row := range rows {
		results[i] = &search.Result{
			ServiceName:     proto.String(row.ServiceName),
			ServiceVersion:  proto.Uint64(row.ServiceVersion),
			MachineClass:    proto.String(row.MachineClass),
			NoFileSoftLimit: proto.Uint64(row.NoFileSoftLimit),
			NoFileHardLimit: proto.Uint64(row.NoFileHardLimit),
		}
	}

	return &search.Response{Results: results}, nil
}
