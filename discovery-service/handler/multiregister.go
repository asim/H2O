package handler

import (
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/protobuf/proto"

	registerproto "github.com/HailoOSS/discovery-service/proto/register"
	"github.com/HailoOSS/discovery-service/registry"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// MultiRegister registers a bunch of endpoints for a service in one hit
func MultiRegister(req *server.Request) (proto.Message, errors.Error) {
	request := req.Data().(*registerproto.MultiRequest)

	inst := multiRegToInstance(request)
	if err := registry.Register(inst); err != nil {
		log.Warnf("[Discovery] Error registering endpoint: %s", err.Error())
		return nil, errors.InternalServerError("com.HailoOSS.kernel.discovery.multiregister", fmt.Sprintf("Error registering: %v", err))
	}

	return &registerproto.Response{}, nil
}
