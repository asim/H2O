package main

import (
	"time"

	handler "github.com/HailoOSS/discovery-service/handler"
	"github.com/HailoOSS/discovery-service/registry"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/service/zookeeper"

	endpointsproto "github.com/HailoOSS/discovery-service/proto/endpoints"
	instancesproto "github.com/HailoOSS/discovery-service/proto/instances"
	registerproto "github.com/HailoOSS/discovery-service/proto/register"
	servicesproto "github.com/HailoOSS/discovery-service/proto/services"
	unregisterproto "github.com/HailoOSS/discovery-service/proto/unregister"
)

func main() {
	server.Name = "com.HailoOSS.kernel.discovery"
	server.Description = "Discovery service; responsible for knowing which services are currently running on which boxes"
	server.Version = ServiceVersion
	server.Source = "github.com/HailoOSS/discovery-service"
	server.OwnerEmail = "dg@HailoOSS.com"
	server.OwnerMobile = "+447921465358"

	server.Init()

	server.Register(
		&server.Endpoint{
			Name:             "multiregister",
			Mean:             50,
			Upper95:          100,
			Handler:          handler.MultiRegister,
			Authoriser:       server.OpenToTheWorldAuthoriser(),
			RequestProtocol:  new(registerproto.MultiRequest),
			ResponseProtocol: new(registerproto.Response),
		},
		&server.Endpoint{
			Name:             "unregister",
			Mean:             50,
			Upper95:          100,
			Handler:          handler.Unregister,
			Authoriser:       server.OpenToTheWorldAuthoriser(),
			RequestProtocol:  new(unregisterproto.Request),
			ResponseProtocol: new(unregisterproto.Response),
		},
		&server.Endpoint{
			Name:             "services",
			Mean:             1500,
			Upper95:          3500,
			Handler:          handler.Services,
			Authoriser:       server.OpenToTheWorldAuthoriser(),
			RequestProtocol:  new(servicesproto.Request),
			ResponseProtocol: new(servicesproto.Response),
		},
		&server.Endpoint{
			Name:             "endpoints",
			Mean:             50,
			Upper95:          100,
			Handler:          handler.Endpoints,
			Authoriser:       server.OpenToTheWorldAuthoriser(),
			RequestProtocol:  new(endpointsproto.Request),
			ResponseProtocol: new(endpointsproto.Response),
		},
		&server.Endpoint{
			Name:             "instances",
			Mean:             1000,
			Upper95:          5000,
			Handler:          handler.Instances,
			Authoriser:       server.OpenToTheWorldAuthoriser(),
			RequestProtocol:  new(instancesproto.Request),
			ResponseProtocol: new(instancesproto.Response),
		})

	registry.Init()
	server.HealthCheck(zookeeper.HealthCheckId, zookeeper.HealthCheck())
	zookeeper.WaitForConnect(time.Second)
	server.BindAndRun()
}
