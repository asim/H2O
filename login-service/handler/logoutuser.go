package handler

import (
	"fmt"
	"github.com/HailoOSS/login-service/auther"
	"github.com/HailoOSS/login-service/dao"
	logoutproto "github.com/HailoOSS/login-service/proto/logoutuser"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
)

// LogoutUser will invalidate a user's session, thus effectively logging them out
func LogoutUser(req *server.Request) (proto.Message, errors.Error) {
	request := &logoutproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest(server.Name+".logoutuser.unmarshal", fmt.Sprintf("%v", err.Error()))
	}
	sess, err := dao.ReadActiveSessionFor(request.GetMech(), request.GetDeviceType(), request.GetUid())
	if err != nil {
		return nil, errors.InternalServerError(server.Name+".logoutuser.dao.read", fmt.Sprintf("%v", err.Error()))
	}
	if sess != nil {
		if err := auther.Expire(sess); err != nil {
			return nil, errors.InternalServerError(server.Name+".logoutuser.session.expire", fmt.Sprintf("%v", err.Error()))
		}
	}

	return &logoutproto.Response{}, nil
}
