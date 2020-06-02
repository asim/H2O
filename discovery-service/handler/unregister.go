package handler

import (
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/discovery-service/registry"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"

	unregisterproto "github.com/HailoOSS/discovery-service/proto/unregister"
)

// Unregister removes a service from the discovery service
func Unregister(req *server.Request) (proto.Message, errors.Error) {
	request := req.Data().(*unregisterproto.Request)
	instanceId := request.GetInstanceId()

	if err := registry.Unregister(instanceId); err != nil {
		log.Warnf("[Discovery] Error unregistering endpoint: %v", err)
		return nil, errors.InternalServerError("com.HailoOSS.discovery.handler.unregister", fmt.Sprintf("Error unregistering endpoint: %v", err))
	}

	return &unregisterproto.Response{}, nil
}
