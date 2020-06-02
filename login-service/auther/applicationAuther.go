package auther

import "github.com/HailoOSS/login-service/domain"
import "github.com/HailoOSS/login-service/dao"

type applicationAuther struct {
	normalAuther Auther
	adminAuther  Auther
}

func newApplicationAuther() Auther {
	normalAuther := newH2Auther()
	adminAuther := newLDAPAuther()

	return &applicationAuther{
		normalAuther: normalAuther,
		adminAuther:  adminAuther,
	}
}

// ValidateAuth wraps defaultInstance.ValidateAuth
func (a *applicationAuther) ValidateAuth(app domain.Application, username string, password []byte) error {
	return a.getAuther(app, username).ValidateAuth(app, username, password)
}

// Auth wraps defaultInstance.Auth
func (a *applicationAuther) Auth(app domain.Application, deviceType, username string, password, newPassword []byte, meta map[string]string, session *domain.Session) (*domain.Session, error) {
	return a.getAuther(app, username).Auth(app, deviceType, username, password, newPassword, meta, session)
}

// AuthAs wraps defaultInstance.AuthAs
func (a *applicationAuther) AuthAs(app domain.Application, deviceType, username string, meta map[string]string) (*domain.Session, error) {
	return a.getAuther(app, username).AuthAs(app, deviceType, username, meta)
}

// Auth wraps defaultInstance.Auth
func (a *applicationAuther) OAuth(app domain.Application, deviceType, username, oauthtoken, provider string, meta map[string]string) (*domain.Session, error) {
	return a.getAuther(app, username).OAuth(app, deviceType, username, oauthtoken, provider, meta)
}

// AutoRenew wraps defaultInstance.AutoRenew
func (a *applicationAuther) AutoRenew(s *domain.Session) (*domain.Session, error) {
	return a.getAuther(s.Token.Application(), s.Token.Id).AutoRenew(s)
}

// Expire wraps defaultInstance.Expire
func (a *applicationAuther) Expire(s *domain.Session) error {
	return a.getAuther(s.Token.Application(), s.Token.Id).Expire(s)
}

// ChangePassword wraps defaultInstance.ChangePassword
func (a *applicationAuther) ChangePassword(user *domain.User, newPassword string, activeSession *domain.Session) error {
	return a.getAuther(user.App, user.Uid).ChangePassword(user, newPassword, activeSession)
}

func (a *applicationAuther) getAuther(app domain.Application, uid string) Auther {
	if _, _, ok := dao.IsLDAPUser(app, uid); ok {
		return a.adminAuther
	}

	return a.normalAuther
}
