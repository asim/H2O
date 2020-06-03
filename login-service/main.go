package main

import (
	"time"

	"github.com/hailo-platform/H2O/login-service/dao"
	"github.com/hailo-platform/H2O/login-service/handler"
	authproto "github.com/hailo-platform/H2O/login-service/proto/auth"
	authasproto "github.com/hailo-platform/H2O/login-service/proto/authas"
	changeaccountexpirationdateproto "github.com/hailo-platform/H2O/login-service/proto/changeaccountexpirationdate"
	changeidsproto "github.com/hailo-platform/H2O/login-service/proto/changeids"
	changepasswordproto "github.com/hailo-platform/H2O/login-service/proto/changepassword"
	changeuserstatusproto "github.com/hailo-platform/H2O/login-service/proto/changeuserstatus"
	createuserproto "github.com/hailo-platform/H2O/login-service/proto/createuser"
	deleteindexproto "github.com/hailo-platform/H2O/login-service/proto/deleteindex"
	deletesessionproto "github.com/hailo-platform/H2O/login-service/proto/deletesession"
	deleteuserproto "github.com/hailo-platform/H2O/login-service/proto/deleteuser"
	endpointauthproto "github.com/hailo-platform/H2O/login-service/proto/endpointauth"
	expirepasswordproto "github.com/hailo-platform/H2O/login-service/proto/expirepassword"
	grantserviceproto "github.com/hailo-platform/H2O/login-service/proto/grantservice"
	grantuserproto "github.com/hailo-platform/H2O/login-service/proto/grantuser"
	listsessionsproto "github.com/hailo-platform/H2O/login-service/proto/listsessions"
	listusersproto "github.com/hailo-platform/H2O/login-service/proto/listusers"
	logoutuserproto "github.com/hailo-platform/H2O/login-service/proto/logoutuser"
	readloginproto "github.com/hailo-platform/H2O/login-service/proto/readlogin"
	readsessionproto "github.com/hailo-platform/H2O/login-service/proto/readsession"
	readuserproto "github.com/hailo-platform/H2O/login-service/proto/readuser"
	readusermultiproto "github.com/hailo-platform/H2O/login-service/proto/readusermulti"
	revokeserviceproto "github.com/hailo-platform/H2O/login-service/proto/revokeservice"
	revokeuserproto "github.com/hailo-platform/H2O/login-service/proto/revokeuser"
	setpasswordhashproto "github.com/hailo-platform/H2O/login-service/proto/setpasswordhash"
	updateuserrolesproto "github.com/hailo-platform/H2O/login-service/proto/updateuserroles"
	"github.com/hailo-platform/H2O/login-service/sessinvalidator"
	service "github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/service/cassandra"
	"github.com/hailo-platform/H2O/service/nsq"
	"github.com/hailo-platform/H2O/service/zookeeper"
)

func main() {
	service.Name = "com.hailo-platform/H2O.service.login"
	service.Description = "Responsible for managing authentication credentials and issuing tokens for users knowing these credentials."
	service.Version = ServiceVersion
	service.Source = "github.com/hailo-platform/H2O/login-service"
	service.OwnerEmail = "dg@hailo-platform/H2O.com"
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
