package handler

import (
	"fmt"
	"time"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	setpwd "github.com/HailoOSS/login-service/proto/setpasswordhash"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

func SetPasswordHash(req *server.Request) (proto.Message, errors.Error) {
	request := &setpwd.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest(server.Name+".setpasswordhash.unmarshal", err.Error())
	}

	if len(request.GetPasswordHash()) == 0 {
		return nil, errors.BadRequest(server.Name+".setpasswordhash.nopassword", "No password hash given")
	}

	app := domain.Application(request.GetApplication())

	// 1. Check User exists
	user, err := dao.ReadUser(app, request.GetUid())
	if err != nil {
		return nil, errors.InternalServerError(server.Name+".setpasswordhash.readuser", fmt.Sprintf("Error reading user: %v", err))
	}
	if user == nil {
		return nil, errors.NotFound(server.Name+".setpasswordhash.readuser", fmt.Sprintf("Could not find user with id %s", request.GetUid()))
	}

	if string(user.Password) == request.GetPasswordHash() {
		// already got password hash, ignore
		return &setpwd.Response{}, nil
	}

	user.Password = []byte(request.GetPasswordHash())
	user.PasswordHistory = append(user.PasswordHistory, user.Password)
	// @TODO should maybe trim history, but this endpoint should not be called often so will just append for now
	user.PasswordChange = time.Now()

	if err := dao.UpdateUser(user); err != nil {
		return nil, errors.InternalServerError(server.Name+".setpasswordhash.dao", fmt.Sprintf("Failed to set user's password hash: %v", err))
	}

	return &setpwd.Response{}, nil
}
