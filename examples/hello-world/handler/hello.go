package handler

import (
	"fmt"

	"github.com/hailocab/h2"
	"github.com/hailocab/platform-layer/errors"
	"github.com/hailocab/protobuf/proto"

	protoHello "github.com/hailocab/h2/examples/hello-world/proto/hello"
)

const (
	// Define the full path of the endpoint for the sake of logging or errors.
	HelloEndpoint = "com.hailolabs.service.hello.Hello"
)

// Hello provides a func to process an incoming 'hello' request.
// A handler requires a request parameter and will return either a response or an error
func Hello(req h2.Request) (proto.Message, h2.Error) {
	request := req.Data().(*protoHello.Request)

	// Error example
	if len(request.GetName()) == 0 {
		// We specify multi error types in the platform layer repository in the 'errors'
		// package. See: https://godoc.org/github.com/hailocab/platform-layer/errors
		return nil, errors.BadRequest(HelloEndpoint, "You didn't specify a name in the request")
	}

	// Return response with the message set to `Hello <name>`
	return &protoHello.Response{
		Message: proto.String(fmt.Sprintf("Hello %q", request.GetName())),
	}, nil
}
