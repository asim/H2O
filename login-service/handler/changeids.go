package handler

import (
	"fmt"

	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/login-service/dao"
	"github.com/hailo-platform/H2O/login-service/domain"
	"github.com/hailo-platform/H2O/login-service/event"
	changeids "github.com/hailo-platform/H2O/login-service/proto/changeids"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

// ChangeIds will edit the Ids
func ChangeIds(req *server.Request) (proto.Message, errors.Error) {
	request := &changeids.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest(server.Name+".changeids.unmarshal", err.Error())
	}

	user, err := dao.ReadUser(domain.Application(request.GetApplication()), request.GetUid())
	if err != nil {
		return nil, errors.InternalServerError(server.Name+".changeids", fmt.Sprintf("Failed to read user: %v", err))
	}

	if user == nil {
		return nil, errors.BadRequest(server.Name+".changeids.invaliduser", "User not found")
	}

	user.Ids = stringsToIds(request.GetIds())

	if errs := userValidator.Validate(user); errs.AnyErrors() {
		return nil, errors.BadRequest(server.Name+".changeids.validate", err.Error())
	}

	if err := dao.UpdateUser(user); err != nil {
		return nil, errors.InternalServerError(server.Name+".changeids.update", err.Error())
	}

	if user.ShouldBePublished() {
		e := event.NewUserUpdateEvent(&event.UserEvent{
			Username:       user.Uid,
			AlternateNames: request.GetIds(),
		})

		if e != nil {
			e.Publish()
		}
	}

	return &changeids.Response{}, nil
}
