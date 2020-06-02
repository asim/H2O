package login

import (
	"github.com/HailoOSS/protobuf/proto"
	log "github.com/cihub/seelog"

	"github.com/HailoOSS/hshell/util"
	"github.com/HailoOSS/platform/client"

	auth "github.com/HailoOSS/login-service/proto/auth"
	deletesession "github.com/HailoOSS/login-service/proto/deletesession"
)

var User string
var Session string
var FromService string = "com.HailoOSS.hshell"

func Login(user string, password string) (string, error) {
	return callAuth(user, password, "admin", "")
}

func LoginH2(user, password string) (string, error) {
	return callAuth(user, password, "h2", "ADMIN")
}

func NewPassword(user string, password string, newPassword string) (string, error) {
	return callNewAuth(user, password, newPassword, "admin", "")
}

func NewPasswordH2(user, password, newPassword string) (string, error) {
	return callNewAuth(user, password, newPassword, "h2", "ADMIN")
}

func Logout() error {
	request, _ := client.NewRequest(
		"com.HailoOSS.service.login",
		"deletesession",
		&deletesession.Request{
			SessId: proto.String(Session),
		},
	)
	request.SetFrom(FromService)
	request.SetSessionID(Session)
	response := &deletesession.Response{}
	if err := util.SendRequest(request, response); err != nil {
		log.Error(err)
		return err
	}

	User = ""
	Session = ""
	return nil
}

func callAuth(user, password, mech, application string) (string, error) {
	request, _ := client.NewRequest(
		"com.HailoOSS.service.login",
		"auth",
		&auth.Request{
			Username:    proto.String(user),
			Password:    proto.String(password),
			Mech:        proto.String(mech),
			Application: proto.String(application),
			DeviceType:  proto.String("hshell"),
		},
	)

	request.SetFrom(FromService)
	response := &auth.Response{}
	if err := util.SendRequest(request, response); err != nil {
		log.Error(err)
		return "", err
	}
	User = user
	Session = response.GetSessId()
	return response.GetSessId(), nil
}

func callNewAuth(user string, password string, newPassword string, mech string, application string) (string, error) {
	request, _ := client.NewRequest(
		"com.HailoOSS.service.login",
		"auth",
		&auth.Request{
			Username:    proto.String(user),
			Password:    proto.String(password),
			Mech:        proto.String(mech),
			DeviceType:  proto.String("hshell"),
			NewPassword: proto.String(newPassword),
			Application: proto.String(application),
		},
	)

	request.SetFrom(FromService)
	response := &auth.Response{}
	if err := util.SendRequest(request, response); err != nil {
		log.Error(err)
		return "", err
	}
	User = user
	Session = response.GetSessId()
	return response.GetSessId(), nil
}
