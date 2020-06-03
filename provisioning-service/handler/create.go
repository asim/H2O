package handler

import (
	"fmt"
	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
	createproto "github.com/hailo-platform/H2O/provisioning-manager-service/proto/create"
	create "github.com/hailo-platform/H2O/provisioning-service/proto/create"
)

func Create(req *server.Request) (proto.Message, errors.Error) {
	request := &create.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.hailocab.provisioning.handler.create", fmt.Sprintf("%v", err))
	}

	createReq := &createproto.Request{
		ServiceName:     request.ServiceName,
		ServiceVersion:  request.ServiceVersion,
		MachineClass:    request.MachineClass,
		NoFileSoftLimit: request.NoFileSoftLimit,
		NoFileHardLimit: request.NoFileHardLimit,
	}

	crequest, err := req.ScopedRequest("com.hailocab.kernel.provisioning-manager", "create", createReq)
	if err != nil {
		return nil, errors.InternalServerError("com.hailocab.provisioning.handler.create", fmt.Sprintf("%v", err))
	}

	response := &createproto.Response{}
	if err := client.Req(crequest, response); err != nil {
		return nil, errors.InternalServerError("com.hailocab.provisioning.handler.create", fmt.Sprintf("%v", err))
	}

	return &create.Response{}, nil
}
