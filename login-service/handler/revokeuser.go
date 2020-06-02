package handler

import (
	"fmt"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	revokeproto "github.com/HailoOSS/login-service/proto/revokeuser"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// RevokeUser will remove the supplied roles from the supplied user
func RevokeUser(req *server.Request) (proto.Message, errors.Error) {
	request := &revokeproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.login.revokeuser.unmarshal", err.Error())
	}

	user, err := dao.ReadUser(domain.Application(request.GetApplication()), request.GetUid())
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.revokeuser.dao.read", err.Error())
	}
	if user == nil {
		return nil, errors.NotFound("com.HailoOSS.service.login.revokeuser", fmt.Sprintf("No user with ID %s", request.GetUid()))
	}

	user.RevokeRoles(request.GetRoles())
	if errs := userValidator.Validate(user); errs.AnyErrors() {
		return nil, errors.BadRequest("com.HailoOSS.service.login.revokeuser.validate", errs.Error())
	}

	if err := dao.UpdateUser(user); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.revokeuser.dao.update", err.Error())
	}

	return &revokeproto.Response{}, nil
}
