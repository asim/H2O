package handler

import (
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/protobuf/proto"

	registerproto "github.com/hailo-platform/H2O/discovery-service/proto/register"
	"github.com/hailo-platform/H2O/discovery-service/registry"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

// MultiRegister registers a bunch of endpoints for a service in one hit
func MultiRegister(req *server.Request) (proto.Message, errors.Error) {
	request := req.Data().(*registerproto.MultiRequest)

	inst := multiRegToInstance(request)
	if err := registry.Register(inst); err != nil {
		log.Warnf("[Discovery] Error registering endpoint: %s", err.Error())
		return nil, errors.InternalServerError("com.hailo-platform/H2O.kernel.discovery.multiregister", fmt.Sprintf("Error registering: %v", err))
	}

	return &registerproto.Response{}, nil
}
