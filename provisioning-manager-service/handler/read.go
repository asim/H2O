package handler

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
	"github.com/HailoOSS/provisioning-manager-service/dao"
	read "github.com/HailoOSS/provisioning-manager-service/proto/read"
)

func Read(req *server.Request) (proto.Message, errors.Error) {
	log.Infof("Read... %v", req)

	request := &read.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError(server.Name+".read", fmt.Sprintf("%v", err))
	}

	ps, err := dao.Read(request.GetServiceName(), request.GetServiceVersion(), request.GetMachineClass())
	if err != nil {
		return nil, errors.InternalServerError(server.Name+".read", fmt.Sprintf("%v", err))
	}

	return &read.Response{
		ServiceName:     proto.String(ps.ServiceName),
		ServiceVersion:  proto.Uint64(ps.ServiceVersion),
		MachineClass:    proto.String(ps.MachineClass),
		NoFileSoftLimit: proto.Uint64(ps.NoFileSoftLimit),
		NoFileHardLimit: proto.Uint64(ps.NoFileHardLimit),
		ServiceType:     proto.Uint64(ps.ServiceType),
	}, nil
}
