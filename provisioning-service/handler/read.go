package handler

import (
	"fmt"
	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
	readproto "github.com/hailo-platform/H2O/provisioning-manager-service/proto/read"
	read "github.com/hailo-platform/H2O/provisioning-service/proto/read"
)

func Read(req *server.Request) (proto.Message, errors.Error) {
	request := &read.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.provisioning.handler.read", fmt.Sprintf("%v", err))
	}

	readReq := &readproto.Request{
		ServiceName:    request.ServiceName,
		ServiceVersion: request.ServiceVersion,
		MachineClass:   request.MachineClass,
	}

	rrequest, err := req.ScopedRequest("com.hailo-platform/H2O.kernel.provisioning-manager", "read", readReq)
	if err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.provisioning.handler.read", fmt.Sprintf("%v", err))
	}

	response := &readproto.Response{}
	if err := client.Req(rrequest, response); err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.provisioning.handler.read", fmt.Sprintf("%v", err))
	}

	return &read.Response{
		ServiceName:     response.ServiceName,
		ServiceVersion:  response.ServiceVersion,
		MachineClass:    response.MachineClass,
		NoFileSoftLimit: response.NoFileSoftLimit,
		NoFileHardLimit: response.NoFileHardLimit,
	}, nil
}
