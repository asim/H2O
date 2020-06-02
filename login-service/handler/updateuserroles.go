package handler

import (
	"fmt"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/login-service/event"
	updateproto "github.com/HailoOSS/login-service/proto/updateuserroles"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// UpdateUserRoles will update a user's roleset to match the given roles. The roles are ordered.
func UpdateUserRoles(req *server.Request) (proto.Message, errors.Error) {
	request := &updateproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.login.updateuserroles.unmarshal", err.Error())
	}

	// Validate the new roles before we proceed
	newRoleSet := request.GetRoles()
	if err := domain.ValidateRoleSet(newRoleSet); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.login.updateuserroles.validate", err.Error())
	}

	user, err := dao.ReadUser(domain.Application(request.GetApplication()), request.GetUid())
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.updateuserroles.dao.read", err.Error())
	}
	if user == nil {
		return nil, errors.NotFound("com.HailoOSS.service.login.updateuserroles", fmt.Sprintf("No user with ID %s",
			request.GetUid()))
	}

	user.Roles = newRoleSet
	if err := dao.UpdateUser(user); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.updateuserroles.dao.update", err.Error())
	}

	if user.ShouldBePublished() {
		e := event.NewUserUpdateEvent(&event.UserEvent{
			Username: user.Uid,
			Roles:    user.Roles,
		})
		e.Publish()
	}

	return &updateproto.Response{}, nil
}
