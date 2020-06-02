package handler

import (
	"fmt"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/login-service/event"
	grantproto "github.com/HailoOSS/login-service/proto/grantuser"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// GrantUser will apply the supplied roles to the supplied user, appending them to the user's role set
func GrantUser(req *server.Request) (proto.Message, errors.Error) {
	request := &grantproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.login.grantuser.unmarshal", err.Error())
	}

	user, err := dao.ReadUser(domain.Application(request.GetApplication()), request.GetUid())
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.grantuser.dao.read", err.Error())
	}
	if user == nil {
		return nil, errors.NotFound("com.HailoOSS.service.login.grantuser", fmt.Sprintf("No user with ID %s",
			request.GetUid()))
	}

	user.GrantRoles(request.GetRoles())
	if errs := userValidator.Validate(user); errs.AnyErrors() {
		return nil, errors.BadRequest("com.HailoOSS.service.login.grantuser.validate", errs.Error())
	}

	if err := dao.UpdateUser(user); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.grantuser.dao.update", err.Error())
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
