package server

import (
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/platform/errors"
)

// Handler interface
type Handler func(req *Request) (proto.Message, errors.Error)

// Middleware wraps a handler to provide additional features
type Middleware func(*Endpoint, Handler) Handler

// PostConnectHandler repesents a function we run after connecting to RabbitMQ
type PostConnectHandler func()

// CleanupHandler repesents a function we run after the service has been interrupted
type CleanupHandler func()

// RegisterPostConnectHandler adds a post connect handler to the map so we can run it
func RegisterPostConnectHandler(pch PostConnectHandler) {
	// Don't see the need in locking this. Feel free to add if you think it's needed
	postConnHdlrs = append(postConnHdlrs, pch)
}

// RegisterCleanupHandler adds a cleanup handler to the map so we can run it
func RegisterCleanupHandler(ch CleanupHandler) {
	// Don't see the need in locking this. Feel free to add if you think it's needed
	cleanupHdlrs = append(cleanupHdlrs, ch)
}
