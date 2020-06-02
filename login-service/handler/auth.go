package handler

import (
	log "github.com/cihub/seelog"
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/auther"
	"github.com/HailoOSS/login-service/constants"
	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"

	auth "github.com/HailoOSS/login-service/proto/auth"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

type authResponse struct {
	Token   string `json:"token"`
	Session string `json:"session"`
}

func Auth(req *server.Request) (proto.Message, errors.Error) {
	request := &auth.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.login.auth.unmarshal", err.Error())
	}

	authMech := request.GetMech()

	var rsp *auth.Response
	var err errors.Error

	switch authMech {
	case "oauth":
		log.Debugf("Auth request via OAUTH...")
		rsp, err = authViaOauth(request)
	default:
		log.Debug("Auth request via H2...")
		rsp, err = authViaH2(req, request)
	}

	if err != nil {
		return nil, err
	}

	return rsp, nil
}

func authViaOauth(request *auth.Request) (*auth.Response, errors.Error) {
	app := domain.Application(request.GetApplication())
	deviceType := request.GetDeviceType()
	username := request.GetUsername()
	meta := protoToMap(request.GetMeta())
	token := request.GetOauthToken()
	provider := request.GetProvider()

	sess, err := auther.OAuth(app, deviceType, username, token, provider, meta)
	if err != nil {
		return nil, errors.InternalServerError(constants.OauthUnknownErrCode, err.Error())
	}

	if sess == nil {
		return nil, errors.Forbidden(constants.OauthUserNotFoundErrCode, "User could not be found")
	}

	rsp := &auth.Response{
		SessId: proto.String(sess.Id),
		Token:  proto.String(sess.Token.String()),
	}

	return rsp, nil
}

func authViaH2(req *server.Request, request *auth.Request) (*auth.Response, errors.Error) {
	app := domain.Application(request.GetApplication())
	deviceType := request.GetDeviceType()
	username := request.GetUsername()
	password := []byte(request.GetPassword())
	meta := protoToMap(request.GetMeta())
	newPassword := []byte(request.GetNewPassword())
	noExpire := request.GetNoExpire()

	var currentSession *domain.Session = nil
	if noExpire { // Get existing session
		existingSession, err := dao.ReadSession(req.SessionID())
		if err != nil {
			return nil, errors.InternalServerError("com.HailoOSS.service.login.auth.readsession", err.Error())
		}
		currentSession = existingSession
	}

	sess, err := auther.Auth(app, deviceType, username, password, newPassword, meta, currentSession)
	if err == auther.ErrorChangePassword {
		// need a different code for change password
		return nil, errors.InternalServerError("com.HailoOSS.service.login.auth.change-password", err.Error())
	}
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.auth.auther", err.Error())
	}

	if sess == nil {
		return nil, errors.Forbidden(constants.BadCredentialsErrCode, "Bad credentials")
	}

	rsp := &auth.Response{
		SessId: proto.String(sess.Id),
		Token:  proto.String(sess.Token.String()),
	}
	return rsp, nil
}
