package main

import (
	"github.com/hailocab/h2"
	"github.com/hailocab/h2/examples/hello-world/handler"

	protoHello "github.com/hailocab/h2/examples/hello-world/proto/hello"
)

func main() {
	// Define our new service with an unique identifer and a description
	svc := h2.NewService(
		"com.hailolabs.service.hello",
		"Hello world service",
	)

	// Add an endpoint called "hello"
	svc.AddEndpoints(h2.Endpoint{
		Name: "hello",
		Func: handler.Hello,
		Req:  new(protoHello.Request),
		Resp: new(protoHello.Response),
	})

	// Run app!
	svc.Run()
}
