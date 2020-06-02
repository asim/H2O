package handler

import (
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/auther"
	"github.com/HailoOSS/login-service/domain"
	authas "github.com/HailoOSS/login-service/proto/authas"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

func AuthAs(req *server.Request) (proto.Message, errors.Error) {
	request := &authas.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.login.authas.unmarshal", err.Error())
	}

	app := domain.Application(request.GetApplication())
	deviceType := request.GetDeviceType()
	username := request.GetUsername()
	meta := protoToMap(request.GetMeta())

	sess, err := auther.AuthAs(app, deviceType, username, meta)
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.authas.auther", err.Error())
	} else if sess == nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.authas.auther", "No session found")
	}

	rsp := &authas.Response{
		SessId: proto.String(sess.Id),
		Token:  proto.String(sess.Token.String()),
	}

	return rsp, nil
}
