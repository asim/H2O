package handler

import (
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/login-service/event"
	deleteproto "github.com/HailoOSS/login-service/proto/deleteuser"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// DeleteUser will permanently delete a user from the credential store
func DeleteUser(req *server.Request) (proto.Message, errors.Error) {
	request := &deleteproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.login.deleteuser.unmarshal", err.Error())
	}

	user, err := dao.ReadUser(domain.Application(request.GetApplication()), request.GetUid())
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.deleteuser.dao.read", err.Error())
	}
	if user != nil {
		if err := dao.DeleteUser(user); err != nil {
			return nil, errors.InternalServerError("com.HailoOSS.service.login.deleteuser.dao.delete", err.Error())
		}

		if user.ShouldBePublished() {
			e := event.NewUserDeleteEvent(&event.UserEvent{
				Username: user.Uid,
			})
			e.Publish()
		}
	}

	return &deleteproto.Response{}, nil
}
