package auther

import (
	"fmt"
	"time"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/platform/multiclient"
	"github.com/HailoOSS/platform/server"

	loginproto "github.com/HailoOSS/ldap-service/proto/login"
)

type ldapAuther struct {
	*h2Auther
}

func newLDAPAuther() *ldapAuther {
	return &ldapAuther{}
}

func (a *ldapAuther) ValidateAuth(app domain.Application, username string, password []byte) error {
	username, _, ok := dao.IsLDAPUser(app, username)
	if !ok {
		return fmt.Errorf("Username is not a valid LDAP user")
	}

	// Attempt to validate against LDAP server, if this fails use the fallback
	// auther
	_, err := a.callLDAPLogin(username, string(password))
	if err != nil {
		return err
	}

	return nil
}

func (a *ldapAuther) Auth(app domain.Application, deviceType, username string, password, newPassword []byte, meta map[string]string, session *domain.Session) (*domain.Session, error) {
	username, _, ok := dao.IsLDAPUser(app, username)
	if !ok {
		return nil, fmt.Errorf("Username is not a valid LDAP user")
	}

	startAuth := time.Now()

	user, err := a.callLDAPLogin(username, string(password))
	if err != nil {
		return nil, err
	}

	if session == nil {
		session, err = a.getSessionToken(user, app, deviceType, meta, startAuth)
		if err != nil {
			return nil, err
		}
	}

	err = a.sanityCheckUser(user, password, newPassword, session)
	if err != nil {
		return nil, err
	}

	err = a.persistLogin(user, app, deviceType, meta)
	if err != nil {
		return nil, err
	}

	return session, err
}

func (a *ldapAuther) ChangePassword(user *domain.User, newPassword string, activeSession *domain.Session) error {
	return fmt.Errorf("You must change your password using the LDAP directory manager")
}

func (a *ldapAuther) sanityCheckUser(user *domain.User, password, newPassword []byte, sess *domain.Session) error {
	// Check if the user has requested a password change and handle this separately,
	// this is because password changes are not allowed by LDAP so should not be
	// handled by the H2 authoriser
	if user.MustChangePassword() || len(newPassword) > 0 {
		if err := a.ChangePassword(user, string(newPassword), sess); err != nil {
			return err
		}
	}

	return a.h2Auther.sanityCheckUser(user, password, newPassword, sess)
}

func (a *ldapAuther) callLDAPLogin(username, password string) (*domain.User, error) {
	cl := multiclient.New().DefaultScopeFrom(server.Scoper())

	rsp := &loginproto.Response{}
	cl.AddScopedReq(&multiclient.ScopedReq{
		Uid:      "ldap_login",
		Service:  "com.HailoOSS.service.ldap",
		Endpoint: "login",
		Req: &loginproto.Request{
			Username: proto.String(username),
			Password: proto.String(password),
		},
		Rsp: rsp,
	})

	if cl.Execute().AnyErrors() {
		return nil, cl.PlatformError("ldap_login")
	}

	return dao.ConvertLDAPUser(rsp.User), nil
}
