package handler

import (
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	deleteindexproto "github.com/HailoOSS/login-service/proto/deleteindex"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// DeleteIndex will delete an index for a user, leaving the other ones untouched
func DeleteIndex(req *server.Request) (proto.Message, errors.Error) {
	request := &deleteindexproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.login.deleteindex.unmarshal", err.Error())
	}

	user := &domain.User{
		App: domain.Application(request.GetApplication()),
		Uid: request.GetUid(),
	}

	err := dao.DeleteUserIndexes(user, request.GetUid(), []domain.Id{})

	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.deleteindex.dao.read", err.Error())
	}

	return &deleteindexproto.Response{}, nil
}
