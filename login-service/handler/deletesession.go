package handler

import (
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/login-service/auther"
	"github.com/hailo-platform/H2O/login-service/dao"
	sessiondel "github.com/hailo-platform/H2O/login-service/proto/deletesession"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

// DeleteSession will delete a session by ID
func DeleteSession(req *server.Request) (proto.Message, errors.Error) {
	request := &sessiondel.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.hailocab.service.login.sessiondelete.unmarshal", err.Error())
	}

	if req.SessionID() != request.GetSessId() {
		// trying to delete someone elses session, require admin
		if err := authoriseAdmin(req); err != nil {
			return nil, err
		}
	}

	sess, err := dao.ReadSession(request.GetSessId())
	if err != nil {
		return nil, errors.InternalServerError("com.hailocab.service.login.sessiondelete.dao", err.Error())
	}
	if sess != nil {
		if err := auther.Expire(sess); err != nil {
			return nil, errors.InternalServerError("com.hailocab.service.login.sessiondelete.expire", err.Error())
		}
	}

	return &sessiondel.Response{}, nil
}
