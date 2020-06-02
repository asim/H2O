package handler

import (
	"fmt"
	"time"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/login-service/event"
	createproto "github.com/HailoOSS/login-service/proto/createuser"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// CreateUser will create a new user account within the credential store
func CreateUser(req *server.Request) (proto.Message, errors.Error) {
	request := &createproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.login.createuser.unmarshal", err.Error())
	}

	user := &domain.User{
		App:                   domain.Application(request.GetApplication()),
		Uid:                   request.GetUid(),
		Ids:                   stringsToIds(request.GetIds()),
		Created:               protoToTime(request.CreatedTimestamp, time.Time{}),
		Roles:                 request.GetRoles(),
		PasswordHistory:       make([][]byte, 0),
		PasswordChange:        protoToTime(request.PasswordChangeTimestamp, time.Time{}),
		AccountExpirationDate: request.GetAccountExpirationDate(),
	}

	// authorise based on what type of thing we are trying to create
	// NB: for anything ADMIN we are going to require an actual person (not service-to-service) with ADMIN role
	if err := authoriseCreate(req, user); err != nil {
		return nil, err
	}

	if request.GetRequirePassword() || request.GetPassword() != "" {
		// set password now we know we can proceed
		if err := user.SetPassword(request.GetPassword()); err != nil {
			return nil, errors.BadRequest("com.HailoOSS.service.login.createuser.badpassword", fmt.Sprintf("Password %s", err.Error()))
		}
	}

	if errs := userValidator.Validate(user); errs.AnyErrors() {
		return nil, errors.BadRequest("com.HailoOSS.service.login.createuser.validate", errs.Error())
	}

	if err := dao.CreateUser(user, request.GetPassword()); err != nil {
		return nil, err
	}

	if user.ShouldBePublished() {
		e := event.NewUserCreateEvent(&event.UserEvent{
			FirstName:             request.GetFirstname(),
			LastName:              request.GetLastname(),
			AlternateNames:        request.GetIds(),
			CreatedAt:             user.Created.Format(time.RFC3339),
			FullName:              fmt.Sprintf("%s %s", request.GetFirstname(), request.GetLastname()),
			Roles:                 request.GetRoles(),
			Username:              request.GetUid(),
			AccountExpirationDate: request.GetAccountExpirationDate(),
		})

		e.Publish()
	}

	return &createproto.Response{}, nil
}
