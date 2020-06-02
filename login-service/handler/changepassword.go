package handler

import (
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/auther"
	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	cpwd "github.com/HailoOSS/login-service/proto/changepassword"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"

	"github.com/HailoOSS/service/instrumentation"
)

func ChangePassword(req *server.Request) (proto.Message, errors.Error) {
	request := &cpwd.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.login.changepassword.unmarshal", err.Error())
	}

	authMech := request.GetMech()

	// Only support h2 auth mech for now
	if authMech != "h2" {
		return nil, errors.BadRequest("com.HailoOSS.service.login.changepassword.authmech",
			fmt.Sprintf("Unhandled auth mech: %s", authMech))
	}

	app := domain.Application(request.GetApplication())

	// 1. Check User exists
	user, err := dao.ReadUser(app, request.GetUsername())
	if err != nil {
		instrumentation.Counter(1.0, "handler.change_password.error.readuser", 1)
		return nil, errors.InternalServerError("com.HailoOSS.service.login.changepassword.readuser",
			fmt.Sprintf("Error reading user: %v", err))
	}
	if user == nil {
		instrumentation.Counter(1.0, "handler.change_password.error.readuser", 1)
		return nil, errors.NotFound("com.HailoOSS.service.login.changepassword.readuser",
			fmt.Sprintf("Could not find user with username %s", request.GetUsername()))
	}

	// 2. Update password
	var session *domain.Session
	if req.SessionID() != "" {
		if session, err = dao.ReadSession(req.SessionID()); err != nil {
			log.Warnf("Session %s not found", req.SessionID())
		}
	}

	// Validate old password
	if request.GetOldPassword() != "" {
		if err = auther.ValidateAuth(app, request.GetUsername(), []byte(request.GetOldPassword())); err != nil {
			instrumentation.Counter(1.0, "handler.change_password.error.old_password", 1)
			return nil, errors.InternalServerError("com.HailoOSS.service.login.changepassword.validateauth",
				fmt.Sprintf("Unable to validate old password: %v", err))
		}
	}

	// Change it
	if err = auther.ChangePassword(user, request.GetNewPassword(), session); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.changepassword.updatepassword",
			fmt.Sprintf("Unable to update password: %v", err))
	}

	instrumentation.Counter(1.0, "handler.change_password.success", 1)

	log.Infof("Successfully modified password for user [user-name=%s]", request.GetUsername())
	return &cpwd.Response{}, nil
}
