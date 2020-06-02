package main

import (
	"time"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/config-service/config"
	"github.com/HailoOSS/config-service/dao"
	"github.com/HailoOSS/config-service/domain"
	"github.com/HailoOSS/config-service/handler"
	"github.com/HailoOSS/config-service/httpserver"
	service "github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/service/cassandra"
	"github.com/HailoOSS/service/healthcheck"
	"github.com/HailoOSS/service/nsq"
	"github.com/HailoOSS/service/zookeeper"
)

func main() {
	service.Name = "com.HailoOSS.service.config"
	service.Description = "Responsible for storing configuration data for applications."
	service.Version = ServiceVersion
	service.Source = "github.com/HailoOSS/config-service"
	service.OwnerEmail = "dg@HailoOSS.com"
	service.OwnerMobile = "+447921465358"

	// to avoid chicken and egg, manually load c* settings we need to access the config
	config.Bootstrap()

	// DefaultRepository is the default implementation of the data source
	domain.DefaultRepository = &dao.CassandraRepository{}

	// fire off HTTP handler
	go httpserver.Serve(service.Name, service.Source, service.Version)

	service.Init()

	service.Register(&service.Endpoint{
		Name:       "read",
		Mean:       50,
		Upper95:    100,
		Handler:    handler.Read,
		Authoriser: service.RoleAuthoriser([]string{"ADMIN"}),
	})
	service.Register(&service.Endpoint{
		Name:       "compile",
		Mean:       100,
		Upper95:    200,
		Handler:    handler.Compile,
		Authoriser: service.RoleAuthoriser([]string{"ADMIN"}),
	})
	service.Register(&service.Endpoint{
		Name:       "multicompile",
		Mean:       300,
		Upper95:    500,
		Handler:    handler.MultiCompile,
		Authoriser: service.RoleAuthoriser([]string{"ADMIN"}),
	})

	service.Register(&service.Endpoint{
		Name:       "update",
		Mean:       300,
		Upper95:    500,
		Handler:    handler.Update,
		Authoriser: service.RoleAuthoriser([]string{"ADMIN"}),
	})
	service.Register(&service.Endpoint{
		Name:       "delete",
		Mean:       100,
		Upper95:    200,
		Handler:    handler.Delete,
		Authoriser: service.SignInRoleAuthoriser([]string{"ADMIN"}),
	})
	service.Register(&service.Endpoint{
		Name:       "changelog",
		Mean:       100,
		Upper95:    200,
		Handler:    handler.ChangeLog,
		Authoriser: service.SignInRoleAuthoriser([]string{"ADMIN"}),
	})
	service.Register(&service.Endpoint{
		Name:       "explain",
		Mean:       100,
		Upper95:    200,
		Handler:    handler.Explain,
		Authoriser: service.SignInRoleAuthoriser([]string{"ADMIN"}),
	})

	service.Register(&service.Endpoint{
		Name:       "diff",
		Mean:       1000,
		Upper95:    2000,
		Handler:    handler.Diff,
		Authoriser: service.SignInRoleAuthoriser([]string{"ADMIN"}),
	})

	// add healthchecks
	service.HealthCheck(cassandra.HealthCheckId, cassandra.HealthCheck(dao.Keyspace, dao.Cfs))
	service.HealthCheck(nsq.HealthCheckId, nsq.HealthCheck())
	service.PriorityHealthCheck(httpserver.HealthCheckId, httpserver.HttpConnectHealthCheck(), healthcheck.Email)

	if err := zookeeper.WaitForConnect(2 * time.Second); err != nil {
		log.Criticalf("Failed to connect to ZooKeeper")
	}

	service.BindAndRun()
}
