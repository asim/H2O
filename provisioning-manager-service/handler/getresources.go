package handler

import (
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
	"github.com/hailo-platform/H2O/provisioning-manager-service/domain"
	pproto "github.com/hailo-platform/H2O/provisioning-manager-service/proto"
	grproto "github.com/hailo-platform/H2O/provisioning-manager-service/proto/getresources"
	"github.com/hailo-platform/H2O/provisioning-manager-service/registry"
)

func GetResources(req *server.Request) (proto.Message, errors.Error) {
	request := &grproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest(server.Name+".getresources", err.Error())
	}

	var provisioners []*domain.Provisioner
	var err error

	if machineClass := request.GetMachineClass(); len(machineClass) > 0 {
		filter := registry.FilterMachineClass(machineClass)
		provisioners, err = registry.Filtered(filter)
	} else {
		provisioners, err = registry.List()
	}

	if err != nil {
		return nil, errors.InternalServerError(server.Name+".listprovisioners", err.Error())
	}

	var tcpu, ucpu float64
	var tmem, umem uint64
	var tdisk, udisk uint64

	for _, provisioner := range provisioners {
		fcpu := float64(provisioner.Machine.Cores)
		tcpu += fcpu
		tmem += provisioner.Machine.Memory
		ucpu += (provisioner.Machine.Usage.Cpu * fcpu)
		umem += provisioner.Machine.Usage.Memory
		tdisk += provisioner.Machine.Disk
		udisk += provisioner.Machine.Usage.Disk
	}

	return &grproto.Response{
		Total: &pproto.Resource{
			Cpu:    proto.Float64(tcpu),
			Memory: proto.Uint64(tmem),
			Disk:   proto.Uint64(tdisk),
		},
		Usage: &pproto.Resource{
			Cpu:    proto.Float64(ucpu),
			Memory: proto.Uint64(umem),
			Disk:   proto.Uint64(udisk),
		},
	}, nil
}
