package handler

import (
	"fmt"
	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
	delproto "github.com/HailoOSS/provisioning-manager-service/proto/delete"
	delete "github.com/HailoOSS/provisioning-service/proto/delete"
)

func Delete(req *server.Request) (proto.Message, errors.Error) {
	request := &delete.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning.handler.delete", fmt.Sprintf("%v", err))
	}

	deleteReq := &delproto.Request{
		ServiceName:    request.ServiceName,
		ServiceVersion: request.ServiceVersion,
		MachineClass:   request.MachineClass,
	}

	drequest, err := req.ScopedRequest("com.HailoOSS.kernel.provisioning-manager", "delete", deleteReq)
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning.handler.delete", fmt.Sprintf("%v", err))
	}

	response := &delproto.Response{}
	if err := client.Req(drequest, response); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning.handler.delete", fmt.Sprintf("%v", err))
	}

	return &delete.Response{}, nil
}
