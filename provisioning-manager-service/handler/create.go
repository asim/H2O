package handler

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
	"github.com/HailoOSS/provisioning-manager-service/dao"
	"github.com/HailoOSS/provisioning-manager-service/event"
	create "github.com/HailoOSS/provisioning-manager-service/proto/create"
	pdao "github.com/HailoOSS/provisioning-service/dao"
	"github.com/HailoOSS/provisioning-service/pkgmgr"
)

func pkgExists(s *dao.Service) errors.Error {
	exists, err := pkgmgr.Exists(&pdao.ProvisionedService{
		ServiceName:     s.ServiceName,
		ServiceVersion:  s.ServiceVersion,
		MachineClass:    s.MachineClass,
		NoFileSoftLimit: s.NoFileSoftLimit,
		NoFileHardLimit: s.NoFileHardLimit,
	})
	if err != nil {
		return errors.InternalServerError(server.Name+".create", fmt.Sprintf("%v", err))
	}
	if !exists {
		return errors.NotFound(server.Name+".create", fmt.Sprintf("%s-%d not found in s3", s.ServiceName, s.ServiceVersion))
	}
	return nil
}

func Create(req *server.Request) (proto.Message, errors.Error) {
	log.Infof("Create... %v", req)

	request := &create.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError(server.Name+".create", fmt.Sprintf("%v", err))
	}

	newRec := &dao.Service{
		ServiceName:     request.GetServiceName(),
		ServiceVersion:  request.GetServiceVersion(),
		MachineClass:    request.GetMachineClass(),
		NoFileSoftLimit: request.GetNoFileSoftLimit(),
		NoFileHardLimit: request.GetNoFileHardLimit(),
		ServiceType:     request.GetServiceType(),
	}

	if err := pkgExists(newRec); err != nil {
		return nil, err
	}

	if err := dao.Create(newRec); err != nil {
		return nil, errors.InternalServerError(server.Name+".create", fmt.Sprintf("%v", err))
	}

	// Pub an event
	event.ProvisionedToNSQ(request.GetServiceName(), request.GetServiceVersion(), request.GetMachineClass(), req.Auth().AuthUser().Id)

	return &create.Response{}, nil
}
