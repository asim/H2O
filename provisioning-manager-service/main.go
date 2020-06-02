package main

import (
	log "github.com/cihub/seelog"

	service "github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/provisioning-manager-service/cache"
	"github.com/HailoOSS/provisioning-manager-service/handler"
	"github.com/HailoOSS/provisioning-manager-service/instrumenter"
	"github.com/HailoOSS/provisioning-manager-service/runlevels"
	"github.com/HailoOSS/provisioning-manager-service/scheduler"
	"github.com/HailoOSS/provisioning-service/pkgmgr"
)

func main() {
	defer log.Flush()

	service.Name = "com.HailoOSS.kernel.provisioning-manager"
	service.Description = "Responsible for coordinating and caching higher level functionality for provisioning"
	service.Version = ServiceVersion
	service.Source = "github.com/HailoOSS/provisioning-manager-service"
	service.OwnerTeam = "h2o"

	service.Init()

	service.Register(&service.Endpoint{
		Name:       "getresources",
		Mean:       100,
		Upper95:    200,
		Handler:    handler.GetResources,
		Authoriser: service.RoleAuthoriser([]string{"ADMIN"}),
	})

	service.Register(&service.Endpoint{
		Name:       "getprovisioner",
		Mean:       100,
		Upper95:    200,
		Handler:    handler.GetProvisioner,
		Authoriser: service.RoleAuthoriser([]string{"ADMIN"}),
	})

	service.Register(&service.Endpoint{
		Name:       "listprovisioners",
		Mean:       100,
		Upper95:    200,
		Handler:    handler.ListProvisioners,
		Authoriser: service.RoleAuthoriser([]string{"ADMIN"}),
	})

	service.Register(&service.Endpoint{
		Name:       "provisioned",
		Mean:       50,
		Upper95:    100,
		Handler:    handler.Provisioned,
		Authoriser: service.OpenToTheWorldAuthoriser(),
	})

	// CRUD
	service.Register(&service.Endpoint{
		Name:       "search",
		Mean:       100,
		Upper95:    200,
		Handler:    handler.Search,
		Authoriser: service.OpenToTheWorldAuthoriser(),
	})

	service.Register(&service.Endpoint{
		Name:       "create",
		Mean:       100,
		Upper95:    200,
		Handler:    handler.Create,
		Authoriser: service.SignInRoleAuthoriser([]string{"ADMIN"}),
	})

	service.Register(&service.Endpoint{
		Name:       "read",
		Mean:       100,
		Upper95:    200,
		Handler:    handler.Read,
		Authoriser: service.SignInRoleAuthoriser([]string{"ADMIN"}),
	})

	service.Register(&service.Endpoint{
		Name:       "delete",
		Mean:       100,
		Upper95:    200,
		Handler:    handler.Delete,
		Authoriser: service.SignInRoleAuthoriser([]string{"ADMIN"}),
	})

	// Run levels
	service.Register(&service.Endpoint{
		Name:       "describerunlevels",
		Mean:       100,
		Upper95:    200,
		Handler:    handler.DescribeRunLevels,
		Authoriser: service.SignInRoleAuthoriser([]string{"ADMIN"}),
	})

	service.Register(&service.Endpoint{
		Name:       "runlevels",
		Mean:       100,
		Upper95:    200,
		Handler:    handler.RunLevels,
		Authoriser: service.SignInRoleAuthoriser([]string{"ADMIN"}),
	})

	service.Register(&service.Endpoint{
		Name:       "setrunlevel",
		Mean:       100,
		Upper95:    200,
		Handler:    handler.SetRunLevel,
		Authoriser: service.SignInRoleAuthoriser([]string{"ADMIN"}),
	})

	service.Register(&service.Endpoint{
		Name:       "servicerunlevels",
		Mean:       100,
		Upper95:    200,
		Handler:    handler.ServiceRunLevels,
		Authoriser: service.SignInRoleAuthoriser([]string{"ADMIN"}),
	})

	service.Register(&service.Endpoint{
		Name:       "setservicerunlevels",
		Mean:       100,
		Upper95:    200,
		Handler:    handler.SetServiceRunLevels,
		Authoriser: service.SignInRoleAuthoriser([]string{"ADMIN"}),
	})

	// Subscribers
	service.Register(&service.Endpoint{
		Name:       "com.HailoOSS.kernel.provisioning.info",
		Handler:    handler.SubProvisioningInfo,
		Subscribe:  "com.HailoOSS.kernel.provisioning.info",
		Authoriser: service.OpenToTheWorldAuthoriser(),
	})

	service.RegisterPostConnectHandler(pkgmgr.Setup)
	service.RegisterPostConnectHandler(runlevels.Setup)
	service.RegisterPostConnectHandler(scheduler.Run)
	service.RegisterPostConnectHandler(cache.Run)
	service.RegisterPostConnectHandler(instrumenter.Run)
	service.RunWithOptions(&service.Options{
		SelfBind: true,
		Die:      false,
	})
}
