package multiclient

import (
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/errors"
)

// PlatformCaller is the default caller and makes requests via the platform layer
// RPC mechanism (eg: RabbitMQ)
func PlatformCaller() Caller {
	return func(req *client.Request, rsp proto.Message) errors.Error {
		return client.Req(req, rsp)
	}
}
