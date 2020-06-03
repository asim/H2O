package server

import (
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/service/config"

	loadedconfigproto "github.com/hailo-platform/H2O/platform/proto/loadedconfig"
)

// loadedConfigHandler handles inbound requests to `loadedconfig` endpoint
func loadedConfigHandler(req *Request) (proto.Message, errors.Error) {
	configJson := string(config.Raw())
	return &loadedconfigproto.Response{
		Config: proto.String(configJson),
	}, nil
}
