package auth

import (
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/multiclient"
	inst "github.com/HailoOSS/service/instrumentation"

	loginproto "github.com/HailoOSS/login-service/proto"
	authproto "github.com/HailoOSS/login-service/proto/auth"
	sessdelproto "github.com/HailoOSS/login-service/proto/deletesession"
	sessreadproto "github.com/HailoOSS/login-service/proto/readsession"
)

const (
	loginService          = "com.HailoOSS.service.login"
	readSessionEndpoint   = "readsession"
	deleteSessionEndpoint = "deletesession"
	authEndpoint          = "auth"
	badCredentialsErrCode = "com.HailoOSS.service.login.auth.badCredentials"
)

// Scope represents some session witin which we may know about a user who has
// somehow identified themselves to us, or some service that has identified
// itself to us (and we trust)
type Scope interface {
	RpcScope(scoper multiclient.Scoper) Scope
	Clean() Scope
	RecoverSession(sessId string) error
	RecoverService(toEndpoint, fromService string) error
	Auth(mech, device string, creds map[string]string) error
	IsAuth() bool
	AuthUser() *User
	HasAccess(role string) bool
	SignOut(user *User) error
	HasTriedAuth() bool
	Authorised() bool
	SetAuthorised(authorised bool)
}

type realScope struct {
	sync.RWMutex

	authUser                *User  // user auth scope
	toEndpoint, fromService string // service-to-service auth scope

	rpcScoper multiclient.Scoper // the scope we should use when making requests to login service (mainly useful for tracing)
	userCache Cacher             // userCacher is able to cache sess->token lookups

	triedAuth bool // we set to true if/when we attempt to recover session scope

	authorised bool // whether the request has been authorised
}

// New mints a new scope
func New() Scope {
	return &realScope{
		rpcScoper: multiclient.ExplicitScoper(), // a blank scope - client should override
		userCache: &memcacheCacher{},
	}
}

// RpcScope sets up how we scope RPC requests to the login service
// This is primarily useful so that requests are traced correctly
func (s *realScope) RpcScope(scoper multiclient.Scoper) Scope {
	s.Lock()
	defer s.Unlock()
	s.rpcScoper = scoper

	return s
}

// getRpcScope fetches this property, protected by RLock
func (s *realScope) getRpcScope() multiclient.Scoper {
	s.RLock()
	defer s.RUnlock()

	return s.rpcScoper
}

// Clean wipes out any knowledge of who is authenticated within this scope
func (s *realScope) Clean() Scope {
	s.Lock()
	defer s.Unlock()

	s.authUser = nil
	s.triedAuth = false

	return s
}

// RecoverSession will try to turn a sessId into a valid user/token, if possible
// error will be non-nil if something goes wrong during this process - if we
// can't find any valid user with this sessId that is *not* an error
// If there is an error, the current state of the scope *will not have been changed*
// If there is no error, then the state will be updated, either to the recovered
// user *or* to nil, if no user was recovered
func (s *realScope) RecoverSession(sessId string) error {
	t := time.Now()

	u, err := s.doRecoverSession(sessId)
	instTiming("auth.recoverSession", err, t)

	if s.IsAuth() {
		inst.Counter(0.01, "auth.recoverSession.recovered", 1)
	} else {
		inst.Counter(1.0, "auth.recoverSession.badCredentials", 1)
	}

	s.Lock()
	defer s.Unlock()

	s.authUser = u

	if err == nil {
		s.triedAuth = true
	}

	return err
}

// doRecoverSession is the meat and veg for RecoverSession
func (s *realScope) doRecoverSession(sessId string) (*User, error) {
	// Check cache; ignore errors (will have impact on service performance, but not functionality)
	queryLogin := false
	u, hit, err := s.userCache.Fetch(sessId)
	if err != nil {
		log.Warnf("[Auth] Error fetching session from cache (will call login service): %v", err)
		queryLogin = true
	} else if u != nil && u.ExpiryTs.Before(time.Now()) && u.CanAutoRenew() { // Cached token has expired
		log.Infof("[Auth] Cache-recovered token has expired (%s); will call login service", u.ExpiryTs.String())
		queryLogin = true
	} else {
		queryLogin = u == nil && !hit
	}

	if queryLogin {
		cl := multiclient.New().DefaultScopeFrom(s.getRpcScope())
		rsp := &sessreadproto.Response{}
		cl.AddScopedReq(&multiclient.ScopedReq{
			Uid:      "readsess",
			Service:  loginService,
			Endpoint: readSessionEndpoint,
			Req: &sessreadproto.Request{
				SessId: proto.String(sessId),
			},
			Rsp: rsp,
		})

		if cl.Execute().AnyErrorsIgnoring([]string{errors.ErrorNotFound}, nil) {
			err := cl.Succeeded("readsess")
			log.Errorf("[Auth] Auth scope recovery error [%s: %s] %v", err.Type(), err.Code(), err.Description())
			return nil, err
		}

		// found a session?
		if rsp.GetSessId() == "" && rsp.GetToken() == "" {
			log.Debugf("[Auth] Session '%s' not found (not valid) when trying to recover from login service", sessId)
			// @todo we could cache this (at least for a short time) to prevent repeated hammering of login service
		} else {
			u, err = FromSessionToken(rsp.GetSessId(), rsp.GetToken())
			if err != nil {
				log.Errorf("[Auth] Error getting user from session: %v", err)
			} else {
				log.Tracef("[Auth] Auth scope - recovered user '%s' from session '%s'", u.Id, rsp.GetSessId())
			}
		}

		// ignore errors; just means we have no user
		if u != nil {
			s.userCache.Store(u)
		}
	}

	return u, nil
}

// RecoverService will try to add the calling service to our auth scope
// @todo eventually this should crytographically verify the service (which might
// have to change from string)
// NOTE: it's the fromService we don't want to trust since this has come from some
// remote source
func (s *realScope) RecoverService(toEndpoint, fromService string) error {
	s.Lock()
	defer s.Unlock()

	s.toEndpoint = toEndpoint
	s.fromService = fromService

	return nil
}

// Auth will pass the supplied details onto the login service in an attempt
// to authenticate a brand new session
func (s *realScope) Auth(mech, device string, creds map[string]string) error {
	t := time.Now()

	u, err := s.doAuth(mech, device, creds)

	instTiming("auth.auth", err, t)
	if s.IsAuth() {
		inst.Counter(0.01, "auth.authenticate.recovered", 1)
	} else {
		inst.Counter(1.0, "auth.authenticate.badCredentials", 1)
	}

	s.Lock()
	defer s.Unlock()

	s.authUser = u
	if err == nil || err == BadCredentialsError {
		s.triedAuth = true
	}

	return err
}

func (s *realScope) doAuth(mech, device string, creds map[string]string) (*User, error) {
	reqProto := &authproto.Request{
		Mech:       proto.String(mech),
		DeviceType: proto.String(device),
		Meta:       make([]*loginproto.KeyValue, 0),
	}
	for k, v := range creds {
		switch k {
		case "username":
			reqProto.Username = proto.String(v)
		case "password":
			reqProto.Password = proto.String(v)
		case "newPassword":
			reqProto.NewPassword = proto.String(v)
		case "application":
			reqProto.Application = proto.String(v)
		default:
			// Add additional fields to Meta, such as DeviceId, osVersion, appVersion
			reqProto.Meta = append(reqProto.Meta, &loginproto.KeyValue{
				Key:   proto.String(k),
				Value: proto.String(v),
			})
		}
	}

	cl := multiclient.New().DefaultScopeFrom(s.getRpcScope())
	rsp := &authproto.Response{}
	cl.AddScopedReq(&multiclient.ScopedReq{
		Uid:      "auth",
		Service:  loginService,
		Endpoint: authEndpoint,
		Req:      reqProto,
		Rsp:      rsp,
		Options:  client.Options{"retries": 0},
	})

	if cl.Execute().AnyErrors() {
		// specfically map out bad credentials error
		err := cl.Succeeded("auth")
		if err.Code() == badCredentialsErrCode {
			return nil, BadCredentialsError
		}
		return nil, err
	}

	// recover this user
	u, err := FromSessionToken(rsp.GetSessId(), rsp.GetToken())
	if err != nil {
		return nil, err
	}

	if err := s.userCache.Store(u); err != nil {
		log.Errorf("[Auth] Error caching session: %v", err)
	}

	return u, nil
}

// IsAuth tests to see if this scope is currently authenticated
func (s *realScope) IsAuth() bool {
	s.RLock()
	defer s.RUnlock()

	return s.authUser != nil
}

// AuthUser returns the details about the currently auth'd user (if IsAuth)
// or nil (if !IsAuth)
func (s *realScope) AuthUser() *User {
	s.RLock()
	defer s.RUnlock()

	return s.authUser
}

// HasAccess tests if the current authentication scope has access to the given role
// This can be satisfied through either service-to-service authentication OR from a user role
func (s *realScope) HasAccess(role string) bool {
	s.RLock()
	defer s.RUnlock()

	// auth against user
	if s.authUser != nil && s.authUser.HasRole(role) {
		return true
	}

	// auth against service
	// TODO delete when removing s2s rules
	if assume := defaultS2S.assumedRole(s.toEndpoint, s.fromService); assume != "" {
		if matchRoleAgainstSet(role, []string{assume}) {
			return true
		}
	}

	// check whether the request has been marked as authorised
	return s.authUser == nil && s.Authorised()
}

// SignOut destroys the current session so that it cannot be used again
func (s *realScope) SignOut(user *User) error {
	cl := multiclient.New().DefaultScopeFrom(s.getRpcScope())

	cl.AddScopedReq(&multiclient.ScopedReq{
		Uid:      "deletesess",
		Service:  loginService,
		Endpoint: deleteSessionEndpoint,
		Req: &sessdelproto.Request{
			SessId: proto.String(user.SessId),
		},
		Rsp: &sessdelproto.Response{},
	})

	if cl.Execute().AnyErrors() {
		return cl.Succeeded("deletesess")
	}

	if err := s.userCache.Purge(user.SessId); err != nil {
		log.Errorf("[Auth] Error purging session cache: %v", err)
	}

	s.Lock()
	defer s.Unlock()

	s.authUser = nil
	s.triedAuth = false

	return nil
}

// HasTriedAuth returns whether we have tried to auth a token or session ID
// This is useful to determine "unknown user, who hasn't tried to auth"
func (s *realScope) HasTriedAuth() bool {
	s.RLock()
	defer s.RUnlock()

	return s.triedAuth
}

func (s *realScope) Authorised() bool {
	s.RLock()
	defer s.RUnlock()

	return s.authorised
}

func (s *realScope) SetAuthorised(authorised bool) {
	s.Lock()
	defer s.Unlock()

	s.authorised = authorised
}
