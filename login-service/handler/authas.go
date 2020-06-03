package handler

import (
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/login-service/auther"
	"github.com/hailo-platform/H2O/login-service/domain"
	authas "github.com/hailo-platform/H2O/login-service/proto/authas"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

func AuthAs(req *server.Request) (proto.Message, errors.Error) {
	request := &authas.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.hailo-platform/H2O.service.login.authas.unmarshal", err.Error())
	}

	app := domain.Application(request.GetApplication())
	deviceType := request.GetDeviceType()
	username := request.GetUsername()
	meta := protoToMap(request.GetMeta())

	sess, err := auther.AuthAs(app, deviceType, username, meta)
	if err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.service.login.authas.auther", err.Error())
	} else if sess == nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.service.login.authas.auther", "No session found")
	}

	rsp := &authas.Response{
		SessId: proto.String(sess.Id),
		Token:  proto.String(sess.Token.String()),
	}

	return rsp, nil
}
