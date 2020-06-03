package handler

import (
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
	"github.com/hailo-platform/H2O/provisioning-manager-service/domain"
	gpproto "github.com/hailo-platform/H2O/provisioning-manager-service/proto/getprovisioner"
	"github.com/hailo-platform/H2O/provisioning-manager-service/registry"
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
