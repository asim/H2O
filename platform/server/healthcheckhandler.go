package server

import (
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/healthcheck"
)

// healthHandler handles inbound requests to `health` endpoint
func healthHandler(req *Request) (proto.Message, errors.Error) {
	return healthcheck.Status(), nil
}
