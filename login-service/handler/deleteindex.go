package handler

import (
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/login-service/dao"
	"github.com/hailo-platform/H2O/login-service/domain"
	deleteindexproto "github.com/hailo-platform/H2O/login-service/proto/deleteindex"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

// DeleteIndex will delete an index for a user, leaving the other ones untouched
func DeleteIndex(req *server.Request) (proto.Message, errors.Error) {
	request := &deleteindexproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.hailo-platform/H2O.service.login.deleteindex.unmarshal", err.Error())
	}

	user := &domain.User{
		App: domain.Application(request.GetApplication()),
		Uid: request.GetUid(),
	}

	err := dao.DeleteUserIndexes(user, request.GetUid(), []domain.Id{})

	if err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.service.login.deleteindex.dao.read", err.Error())
	}

	return &deleteindexproto.Response{}, nil
}
