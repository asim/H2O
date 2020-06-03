package main

import (
	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/binding-service/binding"
	"github.com/hailo-platform/H2O/binding-service/handler"
	bindinghealth "github.com/hailo-platform/H2O/binding-service/healthcheck"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/service/zookeeper"
	"time"
)

func main() {
	defer log.Flush()

	// register service + endpoints
	server.Name = "com.hailo-platform/H2O.kernel.binding"
	server.Description = "Binding service; responsible for binding brokers and services"
	server.Version = ServiceVersion
	server.Source = "github.com/hailo-platform/H2O/binding-service"
	server.OwnerEmail = "dominic@hailo-platform/H2O.com"
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
		Name:       "com.hailo-platform/H2O.kernel.discovery.serviceup",
		Handler:    handler.ServiceUpListener,
		Authoriser: server.OpenToTheWorldAuthoriser(),
	})

	// only register, don't bind. We'll manually do it in the init() call
	server.Register(&server.Endpoint{
		Name:       "com.hailo-platform/H2O.kernel.discovery.servicedown",
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
