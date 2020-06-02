package handler

import (
	"fmt"
	"time"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	expwd "github.com/HailoOSS/login-service/proto/expirepassword"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

func ExpirePassword(req *server.Request) (proto.Message, errors.Error) {
	request := &expwd.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest(server.Name+".expirepassword.unmarshal", err.Error())
	}

	user, err := dao.ReadUser(domain.Application(request.GetApplication()), request.GetUid())
	if err != nil {
		return nil, errors.InternalServerError(server.Name+".expirepassword", fmt.Sprintf("Failed to read user: %v", err))
	}

	if user == nil {
		return nil, errors.BadRequest(server.Name+".expirepassword.invaliduser", "User not found")
	}

	// set the passwordchange to epoch start
	user.PasswordChange = time.Time{}

	if errs := userValidator.Validate(user); errs.AnyErrors() {
		return nil, errors.BadRequest(server.Name+".expirepassword.validate", err.Error())
	}

	if err := dao.UpdateUser(user); err != nil {
		return nil, errors.InternalServerError(server.Name+".expirepassword.update", err.Error())
	}

	return &expwd.Response{}, nil
}
