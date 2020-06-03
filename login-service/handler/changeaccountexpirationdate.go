package handler

import (
	"fmt"

	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/login-service/dao"
	"github.com/hailo-platform/H2O/login-service/domain"
	"github.com/hailo-platform/H2O/login-service/event"
	changeaccountexpirationdate "github.com/hailo-platform/H2O/login-service/proto/changeaccountexpirationdate"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

// Change status will update status field of user and set it to given value
func ChangeAccountExpirationDate(req *server.Request) (proto.Message, errors.Error) {
	request := &changeaccountexpirationdate.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.hailocab.service.login.changeaccountexpirationdate.unmarshal", err.Error())
	}

	user, err := dao.ReadUser(domain.Application(request.GetApplication()), request.GetUid())

	if err != nil {
		return nil, errors.InternalServerError("com.hailocab.service.login.changeaccountexpirationdate.dao.read", err.Error())
	}

	if user == nil {
		return nil, errors.NotFound("com.hailocab.service.login.changeaccountexpirationdate", fmt.Sprintf("No user with ID %s", request.GetUid()))
	}

	user.AccountExpirationDate = request.GetAccountExpirationDate()

	if err := dao.UpdateUser(user); err != nil {
		return nil, errors.InternalServerError("com.hailocab.service.login.changeaccountexpirationdate.dao.updateuser", err.Error())
	}

	if user.ShouldBePublished() {
		e := event.NewUserUpdateEvent(&event.UserEvent{
			Username:              user.Uid,
			AccountExpirationDate: request.GetAccountExpirationDate(),
		})
		e.Publish()
	}

	return &changeaccountexpirationdate.Response{}, nil
}
