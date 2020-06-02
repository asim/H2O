package handler

import (
	"fmt"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/login-service/event"
	changeaccountexpirationdate "github.com/HailoOSS/login-service/proto/changeaccountexpirationdate"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// Change status will update status field of user and set it to given value
func ChangeAccountExpirationDate(req *server.Request) (proto.Message, errors.Error) {
	request := &changeaccountexpirationdate.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.login.changeaccountexpirationdate.unmarshal", err.Error())
	}

	user, err := dao.ReadUser(domain.Application(request.GetApplication()), request.GetUid())

	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.changeaccountexpirationdate.dao.read", err.Error())
	}

	if user == nil {
		return nil, errors.NotFound("com.HailoOSS.service.login.changeaccountexpirationdate", fmt.Sprintf("No user with ID %s", request.GetUid()))
	}

	user.AccountExpirationDate = request.GetAccountExpirationDate()

	if err := dao.UpdateUser(user); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.changeaccountexpirationdate.dao.updateuser", err.Error())
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
