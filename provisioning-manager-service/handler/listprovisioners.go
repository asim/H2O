package handler

import (
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
	"github.com/hailo-platform/H2O/provisioning-manager-service/domain"
	lpproto "github.com/hailo-platform/H2O/provisioning-manager-service/proto/listprovisioners"
	"github.com/hailo-platform/H2O/provisioning-manager-service/registry"
)

func ListProvisioners(req *server.Request) (proto.Message, errors.Error) {
	request := &lpproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest(server.Name+".listprovisioners", err.Error())
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

	rsp := &lpproto.Response{}

	for _, p := range provisioners {
		rsp.Provisioners = append(rsp.Provisioners, p.ToProto(false))
	}

	return rsp, nil
}
