package handler

import (
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/login-service/dao"
	revokeservice "github.com/hailo-platform/H2O/login-service/proto/revokeservice"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

// RevokeService is the opposite of GrantService and will remove authorisation for
// a specific service to talk to a specific endpoint on another service
func RevokeService(req *server.Request) (proto.Message, errors.Error) {
	request := &revokeservice.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.hailo-platform/H2O.service.login.revokeservice.unmarshal", err.Error())
	}

	epas := protoToEndpointAuth(request.GetEndpoint())
	if err := dao.DeleteEndpointAuths(epas); err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.service.login.revokeservice.dao", err.Error())
	}

	return &revokeservice.Response{}, nil
}
