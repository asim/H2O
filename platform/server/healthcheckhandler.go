package server

import (
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/healthcheck"
)

// healthHandler handles inbound requests to `health` endpoint
func healthHandler(req *Request) (proto.Message, errors.Error) {
	return healthcheck.Status(), nil
}
