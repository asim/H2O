package handler

import (
	"time"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	readproto "github.com/HailoOSS/login-service/proto/readlogin"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// ReadLogin will fetch a list of logins between two dates for a given user
func ReadLogin(req *server.Request) (proto.Message, errors.Error) {
	request := &readproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.login.readlogin.unmarshal", err.Error())
	}

	start := protoToTime(request.RangeStart, time.Now().AddDate(0, -1, 0))
	end := protoToTime(request.RangeEnd, time.Now())
	count := request.GetCount()
	lastId := request.GetLastId()

	logins, lastId, err := dao.ReadUserLogins(domain.Application(request.GetApplication()), request.GetUid(), start, end, int(count), lastId)
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.readlogin.dao.read", err.Error())
	}

	return &readproto.Response{
		Login:  loginsToProto(logins),
		LastId: proto.String(lastId),
	}, nil
}
