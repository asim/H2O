package auther

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/login-service/event"
	"github.com/HailoOSS/login-service/sessinvalidator"
	"github.com/HailoOSS/login-service/signer"
	"github.com/HailoOSS/platform/multiclient"
	"github.com/HailoOSS/service/jstats"
	oauth "github.com/HailoOSS/oauth-client-service/proto/info"
	"github.com/HailoOSS/protobuf/proto"
)

const (
	tokenTtl         = 8 * time.Hour
	tokenRenewWindow = 30 * time.Minute
)

var (
	ErrorChangePassword    = errors.New("Authentication failed - you must change your password")
	ErrorAccountIsDisabled = errors.New("Authentication failed - your account is disabled")
	ErrorAccountIsExpired  = errors.New("Authentication failed - your account is expired")
)

// h2autheR is the default implementation
type h2Auther struct {
}

func newH2Auther() *h2Auther {
	return &h2Auther{}
}

func (a *h2Auther) ChangePassword(user *domain.User, newPassword string, activeSession *domain.Session) error {
	log.Tracef("[Default auther] Changing user password for user with UID %s", user.Uid)

	if len(newPassword) == 0 {
		return ErrorChangePassword
	}
	if err := user.SetPassword(newPassword); err != nil {
		return fmt.Errorf("Authentication failed - invalid new password: %s", err.Error())
	}
	if err := dao.UpdateUser(user); err != nil {
		return fmt.Errorf("Authentication failed - failed to save user's new password: %s", err.Error())
	}

	// Invalidate any of the user's other active sessions (but not this one)
	sessionIds, err := dao.ReadActiveSessionIdsFor(user.Uid)
	if err != nil {
		return fmt.Errorf("Failed to get active sessions: %s", err.Error())
	}

	log.Debugf("[Change password] user sessions found %d", len(sessionIds))
	for deviceType, sessionId := range sessionIds {
		if activeSession != nil && sessionId == activeSession.Id {
			continue
		}

		log.Debugf("[Change password:Invalidating] Going to delete session: %v", sessionId)
		if err = dao.DeleteSession(sessionId, deviceType); err != nil {
			return fmt.Errorf("Failed to expire other sessions: %s", err.Error())
		}

		sessinvalidator.BroadcastSessionExpiry(sessionId)
	}

	jstats.PublishEvent("passwordChange", map[string]string{
		"userId":      user.Uid,
		"application": string(user.App),
	})

	return nil
}

func (a *h2Auther) sanityCheckAuthRequest(app domain.Application, deviceType, username string) error {
	if len(deviceType) == 0 {
		return errors.New("Authentication failed - missing device type")
	}
	return nil
}

// ValidateAuth will attempt to authenticate a user with credentials `username` and `password` against
// a single application.
// No session is created or user data updated. Just validates credentials are ok.
func (a *h2Auther) ValidateAuth(app domain.Application, username string, password []byte) error {
	user, err := dao.ReadUser(app, username)

	if err != nil {
		return fmt.Errorf("Authentication failed - DAO error: %v", err)
	}
	if user == nil {
		log.Debug("[Auther] Auth -- User not found")
		return fmt.Errorf("Authentication failed")
	}

	if err := user.PasswordMatches(password); err != nil {
		log.Debug("[Auther] Auth -- Password does not match stored")
		return fmt.Errorf("Authentication failed")
	}

	return nil
}

func (a *h2Auther) getUser(app domain.Application, deviceType, username string) (*domain.User, error) {
	if err := a.sanityCheckAuthRequest(app, deviceType, username); err != nil {
		return nil, err
	}

	startReadUser := time.Now()
	user, err := dao.ReadUser(app, username)
	endReadUser := time.Now()

	log.Debugf("[Auther] ReadUser %dms", endReadUser.Sub(startReadUser)/time.Millisecond)

	if err != nil {
		return nil, fmt.Errorf("Authentication failed - DAO error: %v", err)
	} else if user == nil {
		log.Debug("[Auther] Auth -- User not found")
		return nil, nil
	}

	return user, nil
}

func (a *h2Auther) sanityCheckUser(user *domain.User, password, newPassword []byte, sess *domain.Session) error {
	// If a new password is specified, or a the user must change their password, attempt to do so
	if user.MustChangePassword() || len(newPassword) > 0 {
		if err := a.ChangePassword(user, string(newPassword), sess); err != nil {
			return err
		}
	}

	if user.IsDisabled() {
		return ErrorAccountIsDisabled
	}

	if user.IsAccountExpired() {
		return ErrorAccountIsExpired
	}

	// passwords matches - lets check format
	if user.OldHashFormat() {
		user.SetPassword(string(password))
		if err := dao.UpdateUser(user); err != nil {
			log.Errorf("[Auther] Failed to update user with new password format: %v", err)
			// only log error, as we should still allow login
		} else {
			log.Debugf("[Auther] Migrated H1 password to H2 for user '%v'", user.Uid)
		}
	}

	return nil
}

func (a *h2Auther) sanityCheckSession(user *domain.User, sess *domain.Session, app domain.Application, deviceType string, meta map[string]string) error {
	existingSess, err := dao.ReadActiveSessionFor(app.ToAuthMechanism(), deviceType, user.Uid)
	if err != nil {
		return fmt.Errorf("Authentication failed - failed checking for existing session: %v", err)
	} else if existingSess != nil {
		if err := Expire(existingSess); err != nil {
			return fmt.Errorf("Authentication failed - failed to release existing session: %v", err)
		}
	}

	// Persist new session
	if err := dao.WriteSession(sess); err != nil {
		return fmt.Errorf("Authentication failed - cannot store session: %v", err)
	}
	return nil
}

func (a *h2Auther) newUserToken(user *domain.User, app domain.Application, deviceType string, now time.Time) *domain.Token {
	expires := now.Add(tokenTtl)

	// configure for auto-renew, but only if roles allow -- specifically can't
	// have any admin role or derivative of
	autoRenew := time.Time{}
	if !user.AnyAdminRoles() {
		autoRenew = expires.Add(-tokenRenewWindow)
	}

	return &domain.Token{
		Created:       now,
		AuthMechanism: app.ToAuthMechanism(),
		DeviceType:    deviceType,
		Id:            user.Uid,
		Expires:       expires,
		AutoRenew:     autoRenew,
		Roles:         user.Roles,
	}
}

// Auth will attempt to authenticate a user with credentials `username` and `password` against
// a single application.
// Where we cannot auth, but there is no error, we return nil session.
// We purposefully don't give any indication of why auth failed, unless it's a change password error
func (a *h2Auther) Auth(app domain.Application, deviceType, username string, password, newPassword []byte, meta map[string]string, session *domain.Session) (*domain.Session, error) {
	startAuth := time.Now()

	user, err := a.getUser(app, deviceType, username)
	if user == nil || err != nil {
		return nil, err
	}

	startPwdMatch := time.Now()
	if err := user.PasswordMatches(password); err != nil {
		// don't return this as an actual error
		log.Debug("[Auther] Auth -- Password does not match stored")
		return nil, nil
	}
	endPwdMatch := time.Now()

	log.Debugf("[Auther] PwdMatch %dms", endPwdMatch.Sub(startPwdMatch)/time.Millisecond)

	var retSession *domain.Session
	if session != nil { // Session was passed in? No need to create one
		retSession = session
	} else {
		retSession, err = a.getSessionToken(user, app, deviceType, meta, startAuth)
		if err != nil {
			return nil, err
		}
	}

	err = a.sanityCheckUser(user, password, newPassword, retSession)
	if err != nil {
		return nil, err
	}

	err = a.persistLogin(user, app, deviceType, meta)
	if err != nil {
		return nil, err
	}

	return retSession, err
}

// AuthAs will retrieve a user token without asking for a password
func (a *h2Auther) AuthAs(app domain.Application, deviceType, username string, meta map[string]string) (*domain.Session, error) {
	log.Debugf("app: %+v, deviceType: %+v, username: %+v", app, deviceType, username)
	startAuth := time.Now()

	user, err := a.getUser(app, deviceType, username)
	if user == nil || err != nil {
		return nil, err
	}

	return a.getSessionToken(user, app, deviceType, meta, startAuth)
}

func (a *h2Auther) getSessionToken(user *domain.User, app domain.Application, deviceType string, meta map[string]string, startAuth time.Time) (*domain.Session, error) {
	now := time.Now()
	// ok -- mint a session and store
	token := a.newUserToken(user, app, deviceType, now)

	startSign := time.Now()
	signed, err := signer.Sign(token)
	endSign := time.Now()
	if err != nil {
		return nil, fmt.Errorf("Authentication failed - cannot sign token: %v", err)
	}

	sess := &domain.Session{
		Id:      newSessionId(),
		Created: now,
		Token:   *signed,
	}
	startLock := time.Now()
	// lock on app + device + userId combo and then see if any existing sessions (and expire them)
	lck, err := lockDeviceUser(app.ToAuthMechanism(), deviceType, user.Uid)
	gotLock := time.Now()
	defer func() {
		startUnlock := time.Now()
		lck.Unlock()
		endUnlock := time.Now()

		log.Debugf("[Auther] Auth duration %dms, Signing %dms, Critical section %dms, Lock time %dms, Unlock time %dms",
			endUnlock.Sub(startAuth)/time.Millisecond,
			endSign.Sub(startSign)/time.Millisecond,
			startUnlock.Sub(gotLock)/time.Millisecond,
			gotLock.Sub(startLock)/time.Millisecond,
			endUnlock.Sub(startUnlock)/time.Millisecond,
		)
	}()

	if err != nil {
		return nil, fmt.Errorf("Authentication failed - failed to lock on device/user: %v", err)
	}

	if err := a.sanityCheckSession(user, sess, app, deviceType, meta); err != nil {
		return nil, err
	}

	return sess, nil
}

func (a *h2Auther) persistLogin(user *domain.User, app domain.Application, deviceType string, meta map[string]string) error {
	// augment meta data with roles
	meta["roles"] = strings.Join(user.Roles, ",")

	// store login record
	login := &domain.Login{
		App:           app,
		Uid:           user.Uid,
		LoggedIn:      time.Now(),
		AuthMechanism: app.ToAuthMechanism(),
		DeviceType:    deviceType,
		Meta:          meta,
	}
	if err := dao.WriteLogin(login); err != nil {
		return fmt.Errorf("Authentication failed - cannot store login record: %v", err)
	}

	if user.ShouldBePublished() {
		e := event.NewUserUpdateEvent(&event.UserEvent{
			Username:    user.Uid,
			LastLoginAt: time.Now().Format(time.RFC3339),
		})
		e.Publish()
	}

	return nil
}

func (a *h2Auther) OAuth(app domain.Application, deviceType, username, oauthtoken, provider string, meta map[string]string) (*domain.Session, error) {
	log.Debugf("app: %+v, deviceType: %+v, username: %+v, provider: %+v, token: %+v", app, deviceType, username, provider, oauthtoken)
	startAuth := time.Now()

	user, err := a.getUser(app, deviceType, username)
	if user == nil {
		return nil, err
	}

	startTokenCheck := time.Now()

	oauthReq := &oauth.Request{
		Token:    proto.String(oauthtoken),
		Provider: proto.String(provider),
	}
	oauthRsp := &oauth.Response{}
	call := multiclient.New().AddScopedReq(&multiclient.ScopedReq{
		Uid:      "oauth",
		Service:  "com.HailoOSS.service.oauth-client",
		Endpoint: "info",
		Req:      oauthReq,
		Rsp:      oauthRsp,
	}).Execute()
	if call.AnyErrors() {
		err := call.Succeeded("oauth")
		// don't return this as an actual error
		log.Errorf("[Auther] Auth -- Token problem %s", err)
		return nil, fmt.Errorf("Failed to retrieve info from token")
	}

	endTokenCheck := time.Now()

	log.Debugf("[Auther] TokenCheck %dms", endTokenCheck.Sub(startTokenCheck)/time.Millisecond)

	if username != oauthRsp.GetEmail() {
		log.Errorf("[Auther] Auth -- Token email doesn't match username %s %s", username, err)
		return nil, nil
	}

	sess, err := a.getSessionToken(user, app, deviceType, meta, startAuth)
	if err != nil {
		return nil, err
	}

	err = a.persistLogin(user, app, deviceType, meta)
	if err != nil {
		return nil, err
	}

	return sess, nil
}

// Expire will remove all knowledge of a session such that it cannot be used anymore
func (a *h2Auther) Expire(s *domain.Session) error {
	if err := dao.DeleteSession(s.Id, s.Token.AuthMechanism); err != nil {
		return fmt.Errorf("Session expire failed: %v", err)
	}
	// Broadcast this via NSQ so all regions can expunge from caches, if they want to
	sessinvalidator.BroadcastSessionExpiry(s.Id)
	return nil
}

// AutoRenew will, if possible, extend the lifetime of a session's token
// Only tokens with an "auto renew" timestamp baked in can be renewed, and then
// only within a certain window towards the end of their life
// Will return nil Session if cannot be renewed
func (a *h2Auther) AutoRenew(s *domain.Session) (*domain.Session, error) {
	if !s.Token.CanAutoRenew() {
		return nil, nil
	}

	// Make sure existing token validates
	if !signer.Verify(&s.Token) {
		return nil, fmt.Errorf("Failed to verify existing token -- refusing to auto-renew.")
	}

	// ok - we can auto-renew, let's go do it
	// what we actually do is mint a "new" token (well we update created timestamp),
	// extend the expiry/auto-renew timestamps and then sign it and store it
	renewed := s.Copy()

	renewed.Token.Created = time.Now()
	renewed.Token.Expires = renewed.Token.Created.Add(tokenTtl)
	renewed.Token.AutoRenew = renewed.Token.Expires.Add(-tokenRenewWindow)

	tSigned, err := signer.Sign(&renewed.Token)
	if err != nil {
		return nil, fmt.Errorf("Cannot renew - failed to sign new token: %v", err)
	}
	renewed.Token = *tSigned

	if err := dao.WriteSession(renewed); err != nil {
		return nil, fmt.Errorf("Cannot renew - failed to save updated token: %v", err)
	}

	log.Debugf("[Auther] Automatically renewed session %v to expire at %v", renewed.Id, renewed.Token.Expires)
	return renewed, nil
}

func newSessionId() string {
	bigi, err := rand.Int(rand.Reader, maxSessRand)
	if err != nil {
		panic("Unexpectedly cannot mint session ID")
	}
	return base64.StdEncoding.EncodeToString(bigi.Bytes())
}
