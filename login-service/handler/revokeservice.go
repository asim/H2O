package handler

import (
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/dao"
	revokeservice "github.com/HailoOSS/login-service/proto/revokeservice"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// RevokeService is the opposite of GrantService and will remove authorisation for
// a specific service to talk to a specific endpoint on another service
func RevokeService(req *server.Request) (proto.Message, errors.Error) {
	request := &revokeservice.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.login.revokeservice.unmarshal", err.Error())
	}

	epas := protoToEndpointAuth(request.GetEndpoint())
	if err := dao.DeleteEndpointAuths(epas); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.revokeservice.dao", err.Error())
	}

	return &revokeservice.Response{}, nil
}
