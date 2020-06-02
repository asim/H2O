package handler

import (
	"fmt"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/login-service/event"
	changeuserstatusproto "github.com/HailoOSS/login-service/proto/changeuserstatus"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"

	log "github.com/cihub/seelog"
)

// Change status will update status field of user and set it to given value
func ChangeUserStatus(req *server.Request) (proto.Message, errors.Error) {
	request := &changeuserstatusproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.login.changeuserstatus.unmarshal", err.Error())
	}

	updatedUids := make([]string, 0)
	uids := make([]string, 0, 10)

	if len(request.GetUid()) > 0 {
		uids = append(uids, request.GetUid())
	} else {
		uids = request.GetUids()
	}

	for _, uid := range uids {
		user, err := dao.ReadUser(domain.Application(request.GetApplication()), uid)
		if err != nil {
			log.Error(errors.InternalServerError("com.HailoOSS.service.login.changeuserstatus.dao.read", err.Error()))
			continue
		}
		if user == nil {
			log.Error(errors.NotFound("com.HailoOSS.service.login.changeuserstatus", fmt.Sprintf("No user with ID %s", request.GetUid())))
			continue
		}

		if request.GetStatus() != "enabled" && request.GetStatus() != "disabled" {
			log.Error(errors.NotFound("com.HailoOSS.service.login.changeuserstatus", fmt.Sprintf("Invalid status %s for user", request.GetStatus())))
			continue
		}

		user.Status = request.GetStatus()

		if err := dao.UpdateUser(user); err != nil {
			log.Error(errors.InternalServerError("com.HailoOSS.service.login.changeuserstats.dao.changeuserstatus", err.Error()))
			continue
		}

		if user.ShouldBePublished() {
			e := event.NewUserUpdateEvent(&event.UserEvent{
				Username: user.Uid,
				Status:   request.GetStatus(),
			})
			e.Publish()
		}

		updatedUids = append(updatedUids, user.Uid)
	}

	return &changeuserstatusproto.Response{Uids: updatedUids}, nil
}
