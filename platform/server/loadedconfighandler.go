package server

import (
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/service/config"

	loadedconfigproto "github.com/HailoOSS/platform/proto/loadedconfig"
)

// loadedConfigHandler handles inbound requests to `loadedconfig` endpoint
func loadedConfigHandler(req *Request) (proto.Message, errors.Error) {
	configJson := string(config.Raw())
	return &loadedconfigproto.Response{
		Config: proto.String(configJson),
	}, nil
}
