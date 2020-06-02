package handler

import (
	"fmt"
	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
	readproto "github.com/HailoOSS/provisioning-manager-service/proto/read"
	read "github.com/HailoOSS/provisioning-service/proto/read"
)

func Read(req *server.Request) (proto.Message, errors.Error) {
	request := &read.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning.handler.read", fmt.Sprintf("%v", err))
	}

	readReq := &readproto.Request{
		ServiceName:    request.ServiceName,
		ServiceVersion: request.ServiceVersion,
		MachineClass:   request.MachineClass,
	}

	rrequest, err := req.ScopedRequest("com.HailoOSS.kernel.provisioning-manager", "read", readReq)
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning.handler.read", fmt.Sprintf("%v", err))
	}

	response := &readproto.Response{}
	if err := client.Req(rrequest, response); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning.handler.read", fmt.Sprintf("%v", err))
	}

	return &read.Response{
		ServiceName:     response.ServiceName,
		ServiceVersion:  response.ServiceVersion,
		MachineClass:    response.MachineClass,
		NoFileSoftLimit: response.NoFileSoftLimit,
		NoFileHardLimit: response.NoFileHardLimit,
	}, nil
}
