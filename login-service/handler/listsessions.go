package handler

import (
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/login-service/dao"
	"github.com/hailo-platform/H2O/login-service/domain"
	listproto "github.com/hailo-platform/H2O/login-service/proto/listsessions"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

// ListSessions fetches a list of the logged-in user's active sessions
func ListSessions(r *server.Request) (proto.Message, errors.Error) {
	sessionIds, err := dao.ReadActiveSessionIdsFor(r.Auth().AuthUser().Id)
	if err != nil {
		return nil, errors.InternalServerError("com.hailocab.service.login.listsessions.dao.readids", err.Error())
	}

	rsp := &listproto.Response{
		Sessions: make([]*listproto.Session, len(sessionIds)),
		Uid:      proto.String(r.Auth().AuthUser().Id),
	}

	i := 0
	for _, sessionId := range sessionIds {
		var s *domain.Session
		if s, err = dao.ReadSession(sessionId); err != nil {
			return nil, errors.InternalServerError("com.hailocab.service.login.listsessions.dao.read", err.Error())
		}

		rsp.Sessions[i] = sessionToProto(s)
		i++
	}

	return rsp, nil
}
