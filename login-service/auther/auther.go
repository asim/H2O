package auther

import (
	"math/big"

	"github.com/HailoOSS/login-service/domain"
)

const sessionIdSizeInBits = 1280

type Auther interface {
	Auth(app domain.Application, deviceType, username string, password, newPassword []byte, meta map[string]string, session *domain.Session) (*domain.Session, error)
	AuthAs(app domain.Application, deviceType, username string, meta map[string]string) (*domain.Session, error)
	OAuth(app domain.Application, deviceType, username, oauthtoken, oauthprovider string, meta map[string]string) (*domain.Session, error)
	AutoRenew(s *domain.Session) (*domain.Session, error)
	Expire(s *domain.Session) error
	// ChangePassword changes a user's password for the passed newPassword. It will also immediately invalidate the
	// user's _other_ active sessions; that is, it will not invalidate the passed active session. If activeSession is
	// nil, _all_ the user's sessions will be invalidated.
	ChangePassword(user *domain.User, newPassword string, activeSession *domain.Session) error
	ValidateAuth(app domain.Application, username string, password []byte) error
}

var (
	maxSessRand     *big.Int
	defaultInstance Auther
)

func init() {
	defaultInstance = newApplicationAuther()
	maxSessRand = big.NewInt(0)
	maxSessRand = maxSessRand.SetBit(maxSessRand, sessionIdSizeInBits, 1)
}

// ValidateAuth wraps defaultInstance.ValidateAuth
func ValidateAuth(app domain.Application, username string, password []byte) error {
	return defaultInstance.ValidateAuth(app, username, password)
}

// Auth wraps defaultInstance.Auth
func Auth(app domain.Application, deviceType, username string, password, newPassword []byte, meta map[string]string, session *domain.Session) (*domain.Session, error) {
	return defaultInstance.Auth(app, deviceType, username, password, newPassword, meta, session)
}

// OAuth wraps defaultInstance.OAuth
func OAuth(app domain.Application, deviceType, username, token, provider string, meta map[string]string) (*domain.Session, error) {
	return defaultInstance.OAuth(app, deviceType, username, token, provider, meta)
}

// AuthAs wraps defaultInstance.AuthAs
func AuthAs(app domain.Application, deviceType, username string, meta map[string]string) (*domain.Session, error) {
	return defaultInstance.AuthAs(app, deviceType, username, meta)
}

// AutoRenew wraps defaultInstance.AutoRenew
func AutoRenew(s *domain.Session) (*domain.Session, error) {
	return defaultInstance.AutoRenew(s)
}

// Expire wraps defaultInstance.Expire
func Expire(s *domain.Session) error {
	return defaultInstance.Expire(s)
}

// ChangePassword wraps defaultInstance.ChangePassword
func ChangePassword(user *domain.User, newPassword string, activeSession *domain.Session) error {
	return defaultInstance.ChangePassword(user, newPassword, activeSession)
}
