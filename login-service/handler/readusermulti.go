package handler

import (
	"fmt"
	"github.com/hailo-platform/H2O/login-service/dao"
	"github.com/hailo-platform/H2O/login-service/domain"
	readproto "github.com/hailo-platform/H2O/login-service/proto/readuser"
	multireadproto "github.com/hailo-platform/H2O/login-service/proto/readusermulti"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
)

func ReadUserMulti(req *server.Request) (proto.Message, errors.Error) {
	request := &multireadproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.hailocab.service.login.readusermulti.unmarshal", fmt.Sprintf("%v", err.Error()))
	}

	users, err := dao.MultiReadUser(domain.Application(request.GetApplication()), request.GetUid())
	if err != nil {
		return nil, errors.InternalServerError("com.hailocab.service.login.readusermulti.dao.multiread", fmt.Sprintf("%v", err.Error()))
	}
	if len(users) < 1 {
		return nil, errors.NotFound("com.hailocab.service.login.readusermulti", fmt.Sprintf("No users with ID %s", request.GetUid()))
	}

	// Loop and construct return
	rspUsers := make([]*readproto.Response, len(users))
	for i, user := range users {
		rspUsers[i] = userToProto(user)
	}

	rsp := multireadproto.Response{
		Users: rspUsers,
	}

	return &rsp, nil
}
