package handler

import (
	"fmt"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	readproto "github.com/HailoOSS/login-service/proto/readuser"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// ReadUser will fetch a single user from the credential store by UID or secondary ID
func ReadUser(req *server.Request) (proto.Message, errors.Error) {
	request := &readproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.login.readuser.unmarshal", err.Error())
	}

	user, err := dao.ReadUser(domain.Application(request.GetApplication()), request.GetUid())
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.readuser.dao.read", err.Error())
	}
	if user == nil {
		return nil, errors.NotFound("com.HailoOSS.service.login.readuser", fmt.Sprintf("No user with ID %s", request.GetUid()))
	}

	rsp := userToProto(user)

	return rsp, nil
}
