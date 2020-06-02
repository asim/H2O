package auth

import (
	"errors"
)

var (
	BadCredentialsError error = errors.New("Bad credentials")
	defaultScope        Scope
	defaultS2S          *serviceToService // TODO delete when removing s2s rules
)

func init() {
	defaultScope = New()
	defaultS2S = newServiceToService() // TODO delete when removing s2s rules
}

// Invalidate wraps `Invalidate` against our default memcache-based `Cacher`
func Invalidate(sessId string) error {
	c := &memcacheCacher{}
	return c.Invalidate(sessId)
}

// SetCurrentService defines the current service, as used for service-to-service auth
// This defines who _we_ are, and thus which rules we'll load that define which other
// services will be allowed via HasAccess to call us with assumed role auth
// TODO delete when removing s2s rules
func SetCurrentService(name string) {
	defaultS2S.setService(name)
}

// Clean wraps defaultScope.Clean
func Clean() {
	defaultScope.Clean()
}

// RecoverSession wraps defaultScope.RecoverSession
func RecoverSession(sessId string) error {
	return defaultScope.RecoverSession(sessId)
}

// RecoverService wraps defaultScope.RecoverService
func RecoverService(toEndpoint, fromService string) error {
	return defaultScope.RecoverService(toEndpoint, fromService)
}

// Auth wraps defaultScope.Auth
func Auth(mech, device string, creds map[string]string) error {
	return defaultScope.Auth(mech, device, creds)
}

// IsAuth wraps defaultScope.IsAuth
func IsAuth() bool {
	return defaultScope.IsAuth()
}

// AuthUser wraps defaultScope.AuthUser
func AuthUser() *User {
	return defaultScope.AuthUser()
}

// HasAccess wraps defaultScope.HasAccess
func HasAccess(role string) bool {
	return defaultScope.HasAccess(role)
}

// SignOut wraps defaultScope.SignOut
func SignOut(user *User) {
	defaultScope.SignOut(user)
}
