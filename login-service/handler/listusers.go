package handler

import (
	"time"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"

	listproto "github.com/HailoOSS/login-service/proto/listusers"
)

// ListUsers will fetch a bunch of users within a single application namespace
func ListUsers(req *server.Request) (proto.Message, errors.Error) {
	request := &listproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest(server.Name+".readuser.unmarshal", err.Error())
	}

	start := protoToTime(request.RangeStart, time.Now().AddDate(0, -1, 0))
	end := protoToTime(request.RangeEnd, time.Now())
	count := int(request.GetCount())
	if count < 1 {
		count = 1
	}
	if count > 200 {
		count = 200
	}
	lastId := request.GetLastId()

	users, paginateFrom, err := dao.ReadUserList(domain.Application(request.GetApplication()), start, end, count, lastId)
	if err != nil {
		return nil, errors.InternalServerError(server.Name+".listusers.dao.read", err.Error())
	}

	rsp := &listproto.Response{
		Application: request.Application,
		Users:       make([]*listproto.Response_User, len(users)),
		LastId:      proto.String(paginateFrom),
	}

	for i, u := range users {
		rsp.Users[i] = &listproto.Response_User{
			Uid:              proto.String(u.Uid),
			Ids:              idsToStrings(u.Ids),
			CreatedTimestamp: timeToProto(u.Created),
			Roles:            u.Roles,
			PasswordChangeTimestamp: timeToProto(u.PasswordChange),
		}
	}

	return rsp, nil
}
