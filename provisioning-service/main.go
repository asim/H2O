package main

import (
	service "github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/provisioning-service/config"
	"github.com/HailoOSS/provisioning-service/deps"
	"github.com/HailoOSS/provisioning-service/handler"
	"github.com/HailoOSS/provisioning-service/info"
	"github.com/HailoOSS/provisioning-service/pkgmgr"
	"github.com/HailoOSS/provisioning-service/runner"
)

func main() {
	service.Name = "com.HailoOSS.kernel.provisioning"
	service.Description = "Provisioning service; responsible for provisioning all other services"
	service.Version = ServiceVersion
	service.Source = "github.com/HailoOSS/provisioning-service"

	config.Bootstrap()

	service.Init()

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
	service.Register(&service.Endpoint{
		Name:       "com.HailoOSS.kernel.provisioning.restart",
		Handler:    handler.Restart,
		Authoriser: service.SignInRoleAuthoriser([]string{"ADMIN"}),
		Subscribe:  "com.HailoOSS.kernel.provisioning.restart",
	})
	service.Register(&service.Endpoint{
		Name:       "com.HailoOSS.kernel.provisioning.restartaz",
		Handler:    handler.RestartAZ,
		Authoriser: service.SignInRoleAuthoriser([]string{"ADMIN"}),
		Subscribe:  "com.HailoOSS.kernel.provisioning.restartaz",
	})

	service.RegisterPostConnectHandler(pkgmgr.Setup)
	service.RegisterPostConnectHandler(runner.Run)
	service.RegisterPostConnectHandler(deps.Run)
	service.RegisterPostConnectHandler(info.Run)

	service.RunWithOptions(&service.Options{
		SelfBind: true,
		Die:      false,
	})
}
