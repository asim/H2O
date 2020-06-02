package handler

import (
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
	"github.com/HailoOSS/provisioning-manager-service/domain"
	gpproto "github.com/HailoOSS/provisioning-manager-service/proto/getprovisioner"
	"github.com/HailoOSS/provisioning-manager-service/registry"
)

func GetProvisioner(req *server.Request) (proto.Message, errors.Error) {
	request := &gpproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest(server.Name+".getprovisioner", err.Error())
	}

	if len(request.GetHostname()) == 0 {
		return nil, errors.BadRequest(server.Name+".getprovisioner", "Hostname cannot be blank")
	}

	provisioner, err := registry.Get(&domain.Provisioner{Hostname: request.GetHostname()})
	if err != nil {
		return nil, errors.NotFound(server.Name+".getprovisioner", err.Error())
	}

	return &gpproto.Response{
		Provisioner: provisioner.ToProto(true),
	}, nil
}
