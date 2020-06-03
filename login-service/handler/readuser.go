package handler

import (
	"fmt"

	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/login-service/dao"
	"github.com/hailo-platform/H2O/login-service/domain"
	readproto "github.com/hailo-platform/H2O/login-service/proto/readuser"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

// ReadUser will fetch a single user from the credential store by UID or secondary ID
func ReadUser(req *server.Request) (proto.Message, errors.Error) {
	request := &readproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.hailocab.service.login.readuser.unmarshal", err.Error())
	}

	user, err := dao.ReadUser(domain.Application(request.GetApplication()), request.GetUid())
	if err != nil {
		return nil, errors.InternalServerError("com.hailocab.service.login.readuser.dao.read", err.Error())
	}
	if user == nil {
		return nil, errors.NotFound("com.hailocab.service.login.readuser", fmt.Sprintf("No user with ID %s", request.GetUid()))
	}

	rsp := userToProto(user)

	return rsp, nil
}
