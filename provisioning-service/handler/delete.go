package handler

import (
	"fmt"
	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
	delproto "github.com/hailo-platform/H2O/provisioning-manager-service/proto/delete"
	delete "github.com/hailo-platform/H2O/provisioning-service/proto/delete"
)

func Delete(req *server.Request) (proto.Message, errors.Error) {
	request := &delete.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.provisioning.handler.delete", fmt.Sprintf("%v", err))
	}

	deleteReq := &delproto.Request{
		ServiceName:    request.ServiceName,
		ServiceVersion: request.ServiceVersion,
		MachineClass:   request.MachineClass,
	}

	drequest, err := req.ScopedRequest("com.hailo-platform/H2O.kernel.provisioning-manager", "delete", deleteReq)
	if err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.provisioning.handler.delete", fmt.Sprintf("%v", err))
	}

	response := &delproto.Response{}
	if err := client.Req(drequest, response); err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.provisioning.handler.delete", fmt.Sprintf("%v", err))
	}

	return &delete.Response{}, nil
}
