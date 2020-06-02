package multiclient

import (
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/errors"
)

// ErrorCaller is a very simple caller that just returns an error
// If no error provided, defaults to a `NotFound` error with code "errorcaller.notfound"
func ErrorCaller(err errors.Error) Caller {
	return func(req *client.Request, rsp proto.Message) errors.Error {
		if err != nil {
			return err
		}
		return errors.NotFound("errorcaller.notfound", "No error supplied.")
	}
}
