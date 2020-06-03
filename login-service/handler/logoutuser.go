package handler

import (
	"fmt"
	"github.com/hailo-platform/H2O/login-service/auther"
	"github.com/hailo-platform/H2O/login-service/dao"
	logoutproto "github.com/hailo-platform/H2O/login-service/proto/logoutuser"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
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
