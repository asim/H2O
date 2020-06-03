package handler

import (
	"fmt"

	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/login-service/dao"
	"github.com/hailo-platform/H2O/login-service/domain"
	revokeproto "github.com/hailo-platform/H2O/login-service/proto/revokeuser"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

// RevokeUser will remove the supplied roles from the supplied user
func RevokeUser(req *server.Request) (proto.Message, errors.Error) {
	request := &revokeproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.hailo-platform/H2O.service.login.revokeuser.unmarshal", err.Error())
	}

	user, err := dao.ReadUser(domain.Application(request.GetApplication()), request.GetUid())
	if err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.service.login.revokeuser.dao.read", err.Error())
	}
	if user == nil {
		return nil, errors.NotFound("com.hailo-platform/H2O.service.login.revokeuser", fmt.Sprintf("No user with ID %s", request.GetUid()))
	}

	user.RevokeRoles(request.GetRoles())
	if errs := userValidator.Validate(user); errs.AnyErrors() {
		return nil, errors.BadRequest("com.hailo-platform/H2O.service.login.revokeuser.validate", errs.Error())
	}

	if err := dao.UpdateUser(user); err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.service.login.revokeuser.dao.update", err.Error())
	}

	return &revokeproto.Response{}, nil
}
