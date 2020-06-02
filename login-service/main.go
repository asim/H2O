package main

import (
	"time"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/handler"
	authproto "github.com/HailoOSS/login-service/proto/auth"
	authasproto "github.com/HailoOSS/login-service/proto/authas"
	changeaccountexpirationdateproto "github.com/HailoOSS/login-service/proto/changeaccountexpirationdate"
	changeidsproto "github.com/HailoOSS/login-service/proto/changeids"
	changepasswordproto "github.com/HailoOSS/login-service/proto/changepassword"
	changeuserstatusproto "github.com/HailoOSS/login-service/proto/changeuserstatus"
	createuserproto "github.com/HailoOSS/login-service/proto/createuser"
	deleteindexproto "github.com/HailoOSS/login-service/proto/deleteindex"
	deletesessionproto "github.com/HailoOSS/login-service/proto/deletesession"
	deleteuserproto "github.com/HailoOSS/login-service/proto/deleteuser"
	endpointauthproto "github.com/HailoOSS/login-service/proto/endpointauth"
	expirepasswordproto "github.com/HailoOSS/login-service/proto/expirepassword"
	grantserviceproto "github.com/HailoOSS/login-service/proto/grantservice"
	grantuserproto "github.com/HailoOSS/login-service/proto/grantuser"
	listsessionsproto "github.com/HailoOSS/login-service/proto/listsessions"
	listusersproto "github.com/HailoOSS/login-service/proto/listusers"
	logoutuserproto "github.com/HailoOSS/login-service/proto/logoutuser"
	readloginproto "github.com/HailoOSS/login-service/proto/readlogin"
	readsessionproto "github.com/HailoOSS/login-service/proto/readsession"
	readuserproto "github.com/HailoOSS/login-service/proto/readuser"
	readusermultiproto "github.com/HailoOSS/login-service/proto/readusermulti"
	revokeserviceproto "github.com/HailoOSS/login-service/proto/revokeservice"
	revokeuserproto "github.com/HailoOSS/login-service/proto/revokeuser"
	setpasswordhashproto "github.com/HailoOSS/login-service/proto/setpasswordhash"
	updateuserrolesproto "github.com/HailoOSS/login-service/proto/updateuserroles"
	"github.com/HailoOSS/login-service/sessinvalidator"
	service "github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/service/cassandra"
	"github.com/HailoOSS/service/nsq"
	"github.com/HailoOSS/service/zookeeper"
)

func main() {
	service.Name = "com.HailoOSS.service.login"
	service.Description = "Responsible for managing authentication credentials and issuing tokens for users knowing these credentials."
	service.Version = ServiceVersion
	service.Source = "github.com/HailoOSS/login-service"
	service.OwnerEmail = "dg@HailoOSS.com"
	service.OwnerMobile = "+447921465358"

	service.Init()

	service.Register(&service.Endpoint{
		Name:             "auth",
		Mean:             500,
		Upper95:          2000,
		Handler:          handler.Auth,
		Authoriser:       service.OpenToTheWorldAuthoriser(),
		RequestProtocol:  new(authproto.Request),
		ResponseProtocol: new(authproto.Response),
	})

	service.Register(&service.Endpoint{
		Name:             "authas",
		Mean:             500,
		Upper95:          2000,
		Handler:          handler.AuthAs,
		Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
		RequestProtocol:  new(authasproto.Request),
		ResponseProtocol: new(authasproto.Response),
	})

	// for backwards compat -- we still need sessionread
	service.Register(
		&service.Endpoint{
			Name:             "sessionread",
			Mean:             50,
			Upper95:          200,
			Handler:          handler.ReadSession,
			Authoriser:       service.OpenToTheWorldAuthoriser(),
			RequestProtocol:  new(readsessionproto.Request),
			ResponseProtocol: new(readsessionproto.Response),
		},
		&service.Endpoint{
			Name:             "listsessions",
			Mean:             50,
			Upper95:          250,
			Handler:          handler.ListSessions,
			Authoriser:       service.SignInAuthoriser(),
			RequestProtocol:  new(listsessionsproto.Request),
			ResponseProtocol: new(listsessionsproto.Response),
		},
		&service.Endpoint{
			Name:             "readsession",
			Mean:             50,
			Upper95:          200,
			Handler:          handler.ReadSession,
			Authoriser:       service.OpenToTheWorldAuthoriser(),
			RequestProtocol:  new(readsessionproto.Request),
			ResponseProtocol: new(readsessionproto.Response),
		},
		&service.Endpoint{
			Name:    "deletesession",
			Mean:    50,
			Upper95: 200,
			Handler: handler.DeleteSession,
			// we add additional checks to make sure you're ADMIN or deleting your own session
			Authoriser:       service.OpenToTheWorldAuthoriser(),
			RequestProtocol:  new(deletesessionproto.Request),
			ResponseProtocol: new(deletesessionproto.Response),
		},
		&service.Endpoint{
			Name:    "createuser",
			Mean:    1500,
			Upper95: 2000,
			Handler: handler.CreateUser,
			// we add context-based checks depending on what user is being created
			Authoriser:       service.OpenToTheWorldAuthoriser(),
			RequestProtocol:  new(createuserproto.Request),
			ResponseProtocol: new(createuserproto.Response),
		},
		&service.Endpoint{
			Name:             "readuser",
			Mean:             50,
			Upper95:          200,
			Handler:          handler.ReadUser,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(readuserproto.Request),
			ResponseProtocol: new(readuserproto.Response),
		},
		&service.Endpoint{
			Name:             "readusermulti",
			Mean:             50,
			Upper95:          200,
			Handler:          handler.ReadUserMulti,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(readusermultiproto.Request),
			ResponseProtocol: new(readusermultiproto.Response),
		},
		&service.Endpoint{
			Name:             "listusers",
			Mean:             200,
			Upper95:          400,
			Handler:          handler.ListUsers,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(listusersproto.Request),
			ResponseProtocol: new(listusersproto.Response),
		},
		&service.Endpoint{
			Name:             "deleteuser",
			Mean:             50,
			Upper95:          200,
			Handler:          handler.DeleteUser,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(deleteuserproto.Request),
			ResponseProtocol: new(deleteuserproto.Response),
		},
		&service.Endpoint{
			Name:             "grantuser",
			Mean:             50,
			Upper95:          200,
			Handler:          handler.GrantUser,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(grantuserproto.Request),
			ResponseProtocol: new(grantuserproto.Response),
		},
		&service.Endpoint{
			Name:             "revokeuser",
			Mean:             50,
			Upper95:          200,
			Handler:          handler.RevokeUser,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(revokeuserproto.Request),
			ResponseProtocol: new(revokeuserproto.Response),
		},
		&service.Endpoint{
			Name:       "reindex",
			Mean:       50,
			Upper95:    200,
			Handler:    handler.ReindexUsers,
			Authoriser: service.RoleAuthoriser([]string{"ADMIN"}),
		},
		&service.Endpoint{
			Name:    "endpointauth",
			Mean:    100,
			Upper95: 300,
			Handler: handler.EndpointAuth,
			// we add additional checks to make sure it's the service in question calling this, or ADMIN
			Authoriser:       service.OpenToTheWorldAuthoriser(),
			RequestProtocol:  new(endpointauthproto.Request),
			ResponseProtocol: new(endpointauthproto.Response),
		},
		&service.Endpoint{
			Name:             "grantservice",
			Mean:             50,
			Upper95:          200,
			Handler:          handler.GrantService,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(grantserviceproto.Request),
			ResponseProtocol: new(grantserviceproto.Response),
		},
		&service.Endpoint{
			Name:             "revokeservice",
			Mean:             50,
			Upper95:          200,
			Handler:          handler.RevokeService,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(revokeserviceproto.Request),
			ResponseProtocol: new(revokeserviceproto.Response),
		},
		&service.Endpoint{
			Name:             "readlogin",
			Mean:             50,
			Upper95:          200,
			Handler:          handler.ReadLogin,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(readloginproto.Request),
			ResponseProtocol: new(readloginproto.Response),
		},
		&service.Endpoint{
			Name:             "changeids",
			Mean:             100,
			Upper95:          300,
			Handler:          handler.ChangeIds,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(changeidsproto.Request),
			ResponseProtocol: new(changeidsproto.Response),
		},
		&service.Endpoint{
			Name:             "changepassword",
			Mean:             150,
			Upper95:          500,
			Handler:          handler.ChangePassword,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(changepasswordproto.Request),
			ResponseProtocol: new(changepasswordproto.Response),
		},
		&service.Endpoint{
			Name:             "expirepassword",
			Mean:             100,
			Upper95:          300,
			Handler:          handler.ExpirePassword,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(expirepasswordproto.Request),
			ResponseProtocol: new(expirepasswordproto.Response),
		},
		&service.Endpoint{
			Name:             "setpasswordhash",
			Mean:             150,
			Upper95:          500,
			Handler:          handler.SetPasswordHash,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(setpasswordhashproto.Request),
			ResponseProtocol: new(setpasswordhashproto.Response),
		},
		&service.Endpoint{
			Name:             "logoutuser",
			Mean:             150,
			Upper95:          500,
			Handler:          handler.LogoutUser,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(logoutuserproto.Request),
			ResponseProtocol: new(logoutuserproto.Response),
		},
		&service.Endpoint{
			Name:             "updateuserroles",
			Mean:             150,
			Upper95:          500,
			Handler:          handler.UpdateUserRoles,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(updateuserrolesproto.Request),
			ResponseProtocol: new(updateuserrolesproto.Response),
		},
		&service.Endpoint{
			Name:             "changestatus",
			Mean:             150,
			Upper95:          500,
			Handler:          handler.ChangeUserStatus,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(changeuserstatusproto.Request),
			ResponseProtocol: new(changeuserstatusproto.Response),
		},
		&service.Endpoint{
			Name:             "changeaccountexpirationdate",
			Mean:             150,
			Upper95:          500,
			Handler:          handler.ChangeAccountExpirationDate,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(changeaccountexpirationdateproto.Request),
			ResponseProtocol: new(changeaccountexpirationdateproto.Response),
		},
		&service.Endpoint{
			Name:             "deleteindex",
			Mean:             50,
			Upper95:          200,
			Handler:          handler.DeleteIndex,
			Authoriser:       service.RoleAuthoriser([]string{"ADMIN"}),
			RequestProtocol:  new(deleteindexproto.Request),
			ResponseProtocol: new(deleteindexproto.Response),
		},
	)

	// run our session expirer
	service.RegisterPostConnectHandler(sessinvalidator.Run)

	// add healthchecks
	service.HealthCheck(cassandra.HealthCheckId, cassandra.HealthCheck(dao.Keyspace, dao.Cfs))
	service.HealthCheck(zookeeper.HealthCheckId, zookeeper.HealthCheck())
	service.HealthCheck(nsq.HealthCheckId, nsq.HealthCheck())
	service.HealthCheck(nsq.HighWatermarkId, nsq.HighWatermark(sessinvalidator.TopicName, sessinvalidator.ChannelName, 50))

	// Connection Setup (avoids a race)
	zookeeper.WaitForConnect(time.Second)

	service.BindAndRun()
}
