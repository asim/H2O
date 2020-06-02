package main

import (
	log "github.com/cihub/seelog"
	"github.com/HailoOSS/binding-service/binding"
	"github.com/HailoOSS/binding-service/handler"
	bindinghealth "github.com/HailoOSS/binding-service/healthcheck"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/service/zookeeper"
	"time"
)

func main() {
	defer log.Flush()

	// register service + endpoints
	server.Name = "com.HailoOSS.kernel.binding"
	server.Description = "Binding service; responsible for binding brokers and services"
	server.Version = ServiceVersion
	server.Source = "github.com/HailoOSS/binding-service"
	server.OwnerEmail = "dominic@HailoOSS.com"
	server.OwnerMobile = "+447867524496"

	server.Init()

	server.Register(&server.Endpoint{
		Name:       "subscribetopic",
		Handler:    handler.SubscribeTopicHandler,
		Authoriser: server.OpenToTheWorldAuthoriser(),
	})

	server.Register(&server.Endpoint{
		Name:       "createrule",
		Handler:    handler.CreateBindingRuleHandler,
		Authoriser: server.OpenToTheWorldAuthoriser(),
	})

	server.Register(&server.Endpoint{
		Name:       "deleterule",
		Handler:    handler.DeleteBindingRuleHandler,
		Authoriser: server.OpenToTheWorldAuthoriser(),
	})

	server.Register(&server.Endpoint{
		Name:       "listrules",
		Handler:    handler.ListBindingRulesHandler,
		Authoriser: server.OpenToTheWorldAuthoriser(),
	})

	// only register, don't bind. We'll manually do it in the init() call
	server.Register(&server.Endpoint{
		Name:       "com.HailoOSS.kernel.discovery.serviceup",
		Handler:    handler.ServiceUpListener,
		Authoriser: server.OpenToTheWorldAuthoriser(),
	})

	// only register, don't bind. We'll manually do it in the init() call
	server.Register(&server.Endpoint{
		Name:       "com.HailoOSS.kernel.discovery.servicedown",
		Handler:    handler.ServiceDownListener,
		Authoriser: server.OpenToTheWorldAuthoriser(),
	})

	binding.Init()
	server.RegisterPostConnectHandler(binding.PostConnectHandler)

	server.HealthCheck(bindinghealth.HealthCheckId, bindinghealth.BindingHealthCheck())
	server.HealthCheck(zookeeper.HealthCheckId, zookeeper.HealthCheck())

	zookeeper.WaitForConnect(time.Second)

	// run!
	server.BindAndRun()
}
