package handler

import (
	"time"

	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/login-service/dao"
	"github.com/hailo-platform/H2O/login-service/domain"
	readproto "github.com/hailo-platform/H2O/login-service/proto/readlogin"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

// ReadLogin will fetch a list of logins between two dates for a given user
func ReadLogin(req *server.Request) (proto.Message, errors.Error) {
	request := &readproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.hailo-platform/H2O.service.login.readlogin.unmarshal", err.Error())
	}

	start := protoToTime(request.RangeStart, time.Now().AddDate(0, -1, 0))
	end := protoToTime(request.RangeEnd, time.Now())
	count := request.GetCount()
	lastId := request.GetLastId()

	logins, lastId, err := dao.ReadUserLogins(domain.Application(request.GetApplication()), request.GetUid(), start, end, int(count), lastId)
	if err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.service.login.readlogin.dao.read", err.Error())
	}

	return &readproto.Response{
		Login:  loginsToProto(logins),
		LastId: proto.String(lastId),
	}, nil
}
