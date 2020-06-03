package handler

import (
	"fmt"

	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/login-service/dao"
	"github.com/hailo-platform/H2O/login-service/domain"
	"github.com/hailo-platform/H2O/login-service/event"
	grantproto "github.com/hailo-platform/H2O/login-service/proto/grantuser"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

// GrantUser will apply the supplied roles to the supplied user, appending them to the user's role set
func GrantUser(req *server.Request) (proto.Message, errors.Error) {
	request := &grantproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.hailo-platform/H2O.service.login.grantuser.unmarshal", err.Error())
	}

	user, err := dao.ReadUser(domain.Application(request.GetApplication()), request.GetUid())
	if err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.service.login.grantuser.dao.read", err.Error())
	}
	if user == nil {
		return nil, errors.NotFound("com.hailo-platform/H2O.service.login.grantuser", fmt.Sprintf("No user with ID %s",
			request.GetUid()))
	}

	user.GrantRoles(request.GetRoles())
	if errs := userValidator.Validate(user); errs.AnyErrors() {
		return nil, errors.BadRequest("com.hailo-platform/H2O.service.login.grantuser.validate", errs.Error())
	}

	if err := dao.UpdateUser(user); err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.service.login.grantuser.dao.update", err.Error())
	}

	if user.ShouldBePublished() {
		e := event.NewUserUpdateEvent(&event.UserEvent{
			Username: user.Uid,
			Roles:    user.Roles,
		})
		e.Publish()
	}

	return &grantproto.Response{}, nil
}
