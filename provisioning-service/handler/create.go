package handler

import (
	"fmt"
	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
	createproto "github.com/HailoOSS/provisioning-manager-service/proto/create"
	create "github.com/HailoOSS/provisioning-service/proto/create"
)

func Create(req *server.Request) (proto.Message, errors.Error) {
	request := &create.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning.handler.create", fmt.Sprintf("%v", err))
	}

	createReq := &createproto.Request{
		ServiceName:     request.ServiceName,
		ServiceVersion:  request.ServiceVersion,
		MachineClass:    request.MachineClass,
		NoFileSoftLimit: request.NoFileSoftLimit,
		NoFileHardLimit: request.NoFileHardLimit,
	}

	crequest, err := req.ScopedRequest("com.HailoOSS.kernel.provisioning-manager", "create", createReq)
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning.handler.create", fmt.Sprintf("%v", err))
	}

	response := &createproto.Response{}
	if err := client.Req(crequest, response); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning.handler.create", fmt.Sprintf("%v", err))
	}

	return &create.Response{}, nil
}
