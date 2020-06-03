package handler

import (
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/discovery-service/registry"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"

	unregisterproto "github.com/hailo-platform/H2O/discovery-service/proto/unregister"
)

// Unregister removes a service from the discovery service
func Unregister(req *server.Request) (proto.Message, errors.Error) {
	request := req.Data().(*unregisterproto.Request)
	instanceId := request.GetInstanceId()

	if err := registry.Unregister(instanceId); err != nil {
		log.Warnf("[Discovery] Error unregistering endpoint: %v", err)
		return nil, errors.InternalServerError("com.hailo-platform/H2O.discovery.handler.unregister", fmt.Sprintf("Error unregistering endpoint: %v", err))
	}

	return &unregisterproto.Response{}, nil
}
