package multiclient

import (
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/errors"
)

// PlatformCaller is the default caller and makes requests via the platform layer
// RPC mechanism (eg: RabbitMQ)
func PlatformCaller() Caller {
	return func(req *client.Request, rsp proto.Message) errors.Error {
		return client.Req(req, rsp)
	}
}
