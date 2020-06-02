package auth

import (
	"testing"
	"time"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/multiclient"
	ptesting "github.com/HailoOSS/platform/testing"

	authproto "github.com/HailoOSS/login-service/proto/auth"
	sessdelproto "github.com/HailoOSS/login-service/proto/deletesession"
	sessreadproto "github.com/HailoOSS/login-service/proto/readsession"
)

const (
	testToken  = `am=admin:d=cli:id=dave:ct=1372956175:et=1372984975:rt=:r=ADMIN:sig=OqmD7GCddj7uU0IKy3zflMBpTjnHFk6TG2wtaQTwZTPyC3g/qqE+Zrx0gIVDBb5x2VuXTPHFwjT9Vl85E4NEIy1GUon4GBLt264Kg4hMPxAxMdhcogbjaxmlOCcroCiKfJ06FdKvFvvQUup2tjLAek3XjOqIaPX/x7e7RZzITxYxSI7Mpbuhs0f5rzF1bYuH4/akeQdU1kODqVXpOWP+zJjDyMVATMxc69P7ijRvSKszgomb6m9vmsmpERQdvyNW09NBjGZlLbilPnZ3YKoaFZosYjIXTGNbeywGLf10N4t0qvP2Ms/Z1oNIeFdLqMfgicti00uv+bttTL1vUtBghA==`
	testSessId = `fktWghcPxwqoIMtykAPpPkwesLrAFACsRUExvsnqJRt1e1yCP0TuBJxfTocZgtbZ`
)

func TestSessionRecoverySuite(t *testing.T) {
	ptesting.RunSuite(t, new(sessionRecoverySuite))
}

type sessionRecoverySuite struct {
	ptesting.Suite
}

func (suite *sessionRecoverySuite) SetupSuite() {
	setupKeys()
	suite.Suite.SetupSuite()
}

func (suite *sessionRecoverySuite) TestNew() {
	scope := New().(*realScope)
	suite.Assertions.False(scope.HasTriedAuth())
	suite.Assertions.NotNil(scope.rpcScoper)
}

// TestRecoverSessionHappyFound tests happy case (no service call failures) when we do find a user
func (suite *sessionRecoverySuite) TestRecoverSessionHappyFound() {
	scope := New().(*realScope)
	scope.userCache = newTestCache()

	mock := multiclient.NewMock()
	stub := &multiclient.Stub{
		Service:  loginService,
		Endpoint: readSessionEndpoint,
		Response: &sessreadproto.Response{
			SessId: proto.String(testSessId),
			Token:  proto.String(testToken),
		},
	}
	mock.Stub(stub)
	multiclient.SetCaller(mock.Caller())

	err := scope.RecoverSession(testSessId)
	suite.Assertions.NoError(err, "Unexpected recovery error")
	suite.Assertions.True(scope.IsAuth())
	suite.Assertions.True(scope.HasTriedAuth())
	suite.Assertions.Equal(1, stub.CountCalls(), "Expecting 1 call to readsession")
	req := &sessreadproto.Request{}
	err = stub.Request(0).Unmarshal(req)
	suite.Assertions.NoError(err)
	suite.Assertions.Equal(testSessId, req.GetSessId())
	suite.Assertions.False(req.GetNoRenew())
	u := scope.authUser
	suite.Assertions.NotNil(u)
	suite.Assertions.Equal("dave", u.Id)

	// Clean out scope
	scope.Clean()
	suite.Assertions.False(scope.IsAuth(), "Expecting scope to be IsAuth==false after Clean()")
	suite.Assertions.False(scope.HasTriedAuth(), "Expecting scope to have HasTriedAuth()==false after Clean()")
	suite.Assertions.Nil(scope.AuthUser(), "Expecting AuthUser()==nil after Clean()")

	// recover AGAIN -- this time it shoud be cached
	suite.Assertions.NoError(scope.RecoverSession(testSessId), "Unexpected recovery error")
	suite.Assertions.True(scope.IsAuth())
	suite.Assertions.Equal(1, stub.CountCalls(), "Expecting 1 call to readsession (should be cached now)")
}

// TestRecoverSessionHappyNotFound tests happy case (no service call failures) when
// we do not find a user with the session ID
func (suite *sessionRecoverySuite) TestRecoverSessionHappyNotFound() {
	t := suite.T()

	scope := New().(*realScope)
	scope.userCache = newTestCache()

	mock := multiclient.NewMock()
	stub := &multiclient.Stub{
		Service:  loginService,
		Endpoint: readSessionEndpoint,
		Error:    errors.NotFound("com.HailoOSS.service.login.readsession", "Session not found"),
	}
	mock.Stub(stub)
	multiclient.SetCaller(mock.Caller())

	err := scope.RecoverSession(testSessId)
	if err != nil {
		t.Errorf("Unexpected recover error (not found should NOT be classed as a recovery error): %v", err)
	}
	if scope.IsAuth() {
		t.Error("Expecting scope to be IsAuth==false after recovery where NOT FOUND")
	}
	if !scope.HasTriedAuth() {
		t.Error("Expecting scope to have HasTriedAuth()==true after recovery attempt")
	}
	if u := scope.AuthUser(); u != nil {
		t.Error("Expecting AuthUser()==nil after recover attempt")
	}

	// verify we made correct request(s)
	if stub.CountCalls() != 1 {
		t.Fatalf("Expecting 1 call to readsession; got %v", stub.CountCalls())
	}

	// recover AGAIN -- we should NOT cache NOT FOUNDs, because of C* replication/eventual consistency
	err = scope.RecoverSession(testSessId)
	if err != nil {
		t.Errorf("Unexpected recover error (not found should NOT be classed as a recovery error): %v", err)
	}
	if scope.IsAuth() {
		t.Error("Expecting scope to be IsAuth==false after recovery where NOT FOUND")
	}

	// verify we called again
	if stub.CountCalls() != 2 {
		t.Fatalf("Expecting 2 call to readsession (because it should NOT be cached); got %v", stub.CountCalls())
	}
}

// TestRecoverSessionSad tests sad case (login service has some fatal error)
func (suite *sessionRecoverySuite) TestRecoverSessionSad() {
	t := suite.T()

	scope := New().(*realScope)
	scope.userCache = newTestCache()

	mock := multiclient.NewMock()
	stub := &multiclient.Stub{
		Service:  loginService,
		Endpoint: readSessionEndpoint,
		Error:    errors.InternalServerError("com.HailoOSS.service.login.foo", "Things are foo barred"),
	}
	mock.Stub(stub)
	multiclient.SetCaller(mock.Caller())

	err := scope.RecoverSession(testSessId)
	if err == nil {
		t.Error("Expecting recovery error, because login call should fail.")
	}
	if scope.IsAuth() {
		t.Error("Expecting scope to be IsAuth==false after failed recovery")
	}
	if scope.HasTriedAuth() {
		t.Error("Expecting scope to have HasTriedAuth()==false after _failed_ recovery")
	}

	// verify we made correct request(s)
	if stub.CountCalls() != 1 {
		t.Fatalf("Expecting 1 call to readsession; got %v", stub.CountCalls())
	}
}

func (suite *sessionRecoverySuite) TestRecoverService() {
	t := suite.T()
	scope := New().(*realScope)

	toEndpoint := "someendpoint"
	fromService := "com.HailoOSS.service.foo"

	if scope.toEndpoint != "" {
		t.Errorf("Expecting default 'toEndpoint' to be blank, got '%s'", scope.toEndpoint)
	}
	if scope.fromService != "" {
		t.Errorf("Expecting default 'fromService' to be blank, got '%s'", scope.fromService)
	}

	err := scope.RecoverService(toEndpoint, fromService)
	if err != nil {
		t.Errorf("Error recovering service scope: %v", err)
	}

	if scope.toEndpoint != toEndpoint {
		t.Errorf("Expecting default 'toEndpoint' to be '%s', got '%s'", toEndpoint, scope.toEndpoint)
	}
	if scope.fromService != fromService {
		t.Errorf("Expecting default 'fromService' to be '%s', got '%s'", fromService, scope.fromService)
	}
}

// TestAuthHappyCaseValid tests when things work, and when the credentials are valid
func (suite *sessionRecoverySuite) TestAuthHappyCaseValid() {
	t := suite.T()

	scope := New().(*realScope)
	scope.userCache = newTestCache()

	mock := multiclient.NewMock()
	stub := &multiclient.Stub{
		Service:  loginService,
		Endpoint: authEndpoint,
		Response: &authproto.Response{
			SessId: proto.String(testSessId),
			Token:  proto.String(testToken),
		},
	}
	sessLookupStub := &multiclient.Stub{
		Service:  loginService,
		Endpoint: readSessionEndpoint,
		Response: &sessreadproto.Response{
			SessId: proto.String(testSessId),
			Token:  proto.String(testToken),
		},
	}
	mock.Stub(stub).Stub(sessLookupStub)
	multiclient.SetCaller(mock.Caller())

	testMech, testDeviceType := "h2", "cli"
	testUsername, testPassword := "dave", "Securez1"
	testCreds := map[string]string{
		"username": testUsername,
		"password": testPassword,
	}

	err := scope.Auth(testMech, testDeviceType, testCreds)
	if err != nil {
		t.Errorf("Unexpected auth error: %v", err)
	}
	if !scope.IsAuth() {
		t.Error("Expecting scope to be IsAuth==true after auth")
	}
	if !scope.HasTriedAuth() {
		t.Error("Expecting scope to have HasTriedAuth()==true after auth")
	}

	// verify we made correct request(s)
	if stub.CountCalls() != 1 {
		t.Fatalf("Expecting 1 call to auth; got %v", stub.CountCalls())
	}
	req := &authproto.Request{}
	err = stub.Request(0).Unmarshal(req)
	if err != nil {
		t.Fatalf("Unexpected error unmarshaling our request: %v", err)
	}
	if req.GetMech() != testMech {
		t.Errorf("Request did not contain our expected mech '%s', got '%s'", testMech, req.GetMech())
	}
	if req.GetDeviceType() != testDeviceType {
		t.Errorf("Request did not contain our expected device type '%s', got '%s'", testDeviceType, req.GetDeviceType())
	}
	if req.GetUsername() != testUsername {
		t.Errorf("Request did not contain our expected username '%s', got '%s'", testUsername, req.GetUsername())
	}
	if req.GetPassword() != testPassword {
		t.Errorf("Request did not contain our expected password '%s', got '%s'", testPassword, req.GetPassword())
	}

	// verify user returned
	u := scope.AuthUser()
	if u == nil {
		t.Fatal("IsAuthed scope returned nil user")
	}
	if u.Id != "dave" {
		t.Errorf("Expecting user ID 'dave'; got '%s'", u.Id)
	}

	// clean out scope
	scope.Clean()
	if scope.IsAuth() {
		t.Error("Expecting scope to be IsAuth==false after Clean()")
	}
	if scope.HasTriedAuth() {
		t.Error("Expecting scope to have HasTriedAuth()==false after Clean()")
	}
	if u := scope.AuthUser(); u != nil {
		t.Error("Expecting AuthUser()==nil after Clean()")
	}

	// auth AGAIN -- we should be calling login service _again_
	err = scope.Auth(testMech, testDeviceType, testCreds)
	if err != nil {
		t.Errorf("Unexpected recover error: %v", err)
	}
	if !scope.IsAuth() {
		t.Error("Expecting scope to be IsAuth=true after recovery")
	}

	// verify we haven't made _any more_ requests
	if stub.CountCalls() != 2 {
		t.Fatalf("Expecting 2 call to auth (because it should be called each time); got %v", stub.CountCalls())
	}

	// session should be pushed into cache
	err = scope.RecoverSession(testSessId)
	if err != nil {
		t.Errorf("Unexpected recover error: %v", err)
	}
	if sessLookupStub.CountCalls() != 0 {
		t.Fatalf("Expecting 0 call to readsession (because it should be cached); got %v", stub.CountCalls())
	}
}

// TestAuthHappyCaseValid tests when things work, and when the credentials are bad
func (suite *sessionRecoverySuite) TestAuthHappyCaseInvalid() {
	t := suite.T()

	scope := New().(*realScope)
	scope.userCache = newTestCache()

	mock := multiclient.NewMock()
	stub := &multiclient.Stub{
		Service:  loginService,
		Endpoint: authEndpoint,
		Error:    errors.Forbidden("com.HailoOSS.service.login.auth.badCredentials", "Bad credentials"),
	}
	mock.Stub(stub)
	multiclient.SetCaller(mock.Caller())

	testMech, testDeviceType := "h2", "cli"
	testUsername, testPassword := "dave", "Securez1"
	testCreds := map[string]string{
		"username": testUsername,
		"password": testPassword,
	}

	err := scope.Auth(testMech, testDeviceType, testCreds)
	if err == nil {
		t.Fatal("Expecting auth error")
	}
	if err != BadCredentialsError {
		t.Errorf("Expecting auth error to be BadCredentialsError; got %v", err)
	}
	if scope.IsAuth() {
		t.Error("Expecting scope to be IsAuth==false after bad credentials auth")
	}
	if !scope.HasTriedAuth() {
		t.Error("Expecting scope to have HasTriedAuth()==true after bad credentials auth")
	}
	if u := scope.AuthUser(); u != nil {
		t.Error("Expecting AuthUser()==nil after bad credentials auth")
	}

	// verify we made correct request(s)
	if stub.CountCalls() != 1 {
		t.Fatalf("Expecting 1 call to auth; got %v", stub.CountCalls())
	}
}

// TestAuthUnhappyCase tests when things don't work
func (suite *sessionRecoverySuite) TestAuthUnhappyCaseInvalid() {
	t := suite.T()

	scope := New().(*realScope)
	scope.userCache = newTestCache()

	mock := multiclient.NewMock()
	stub := &multiclient.Stub{
		Service:  loginService,
		Endpoint: authEndpoint,
		Error:    errors.InternalServerError("com.HailoOSS.service.login.auth.foobarred", "It's FOOBARRED"),
	}
	mock.Stub(stub)
	multiclient.SetCaller(mock.Caller())

	testMech, testDeviceType := "h2", "cli"
	testUsername, testPassword := "dave", "Securez1"
	testCreds := map[string]string{
		"username": testUsername,
		"password": testPassword,
	}

	err := scope.Auth(testMech, testDeviceType, testCreds)
	if err == nil {
		t.Fatal("Expecting auth error")
	}
	if err == BadCredentialsError {
		t.Error("Error should not bubble up as a BAD CREDENTIALS error")
	}
	if scope.IsAuth() {
		t.Error("Expecting scope to be IsAuth==false after failed auth")
	}
	if scope.HasTriedAuth() {
		t.Error("Expecting scope to have HasTriedAuth()==false after failed auth")
	}
	if u := scope.AuthUser(); u != nil {
		t.Error("Expecting AuthUser()==nil after failed auth")
	}

	// verify we made correct request(s)
	if stub.CountCalls() != 1 {
		t.Fatalf("Expecting 1 call to auth; got %v", stub.CountCalls())
	}
}

// TestSignOutHappy tests signout when it works
func (suite *sessionRecoverySuite) TestSignOutHappy() {
	t := suite.T()

	scope := New().(*realScope)
	scope.userCache = newTestCache()

	mock := multiclient.NewMock()
	stub := &multiclient.Stub{
		Service:  loginService,
		Endpoint: deleteSessionEndpoint,
		Response: &sessdelproto.Response{},
	}
	readSessStub := &multiclient.Stub{
		Service:  loginService,
		Endpoint: readSessionEndpoint,
		Response: &sessreadproto.Response{
			SessId: proto.String(testSessId),
			Token:  proto.String(testToken),
		},
	}
	mock.Stub(stub).Stub(readSessStub)
	multiclient.SetCaller(mock.Caller())

	// need to recover first
	err := scope.RecoverSession(testSessId)
	if err != nil || !scope.IsAuth() {
		t.Errorf("Unexpected recover error, or not Authed as expected: %v", err)
	}

	// now signout
	err = scope.SignOut(scope.AuthUser())
	if err != nil {
		t.Errorf("Unexpected signout error: %v", err)
	}

	// verify we made correct request(s)
	if stub.CountCalls() != 1 {
		t.Fatalf("Expecting 1 call to deletesession; got %v", stub.CountCalls())
	}
	req := &sessdelproto.Request{}
	err = stub.Request(0).Unmarshal(req)
	if err != nil {
		t.Fatalf("Unexpected error unmarshaling our request: %v", err)
	}
	if req.GetSessId() != testSessId {
		t.Errorf("Request did not contain our expected sessId '%s', got '%s'", testSessId, req.GetSessId())
	}
	if scope.IsAuth() {
		t.Error("Expecting scope to be IsAuth==false after SignOut()")
	}
	if scope.HasTriedAuth() {
		t.Error("Expecting scope to have HasTriedAuth()==false after SignOut()")
	}
	if u := scope.AuthUser(); u != nil {
		t.Error("Expecting AuthUser()==nil after SignOut()")
	}

	// make sure we have purged cache
	err = scope.RecoverSession(testSessId)
	if err != nil {
		t.Errorf("Expecting error when recovering session again")
	}
	if readSessStub.CountCalls() != 2 {
		t.Error("Expecting 2 calls to readsession - sincee we should have purged the cache after SignOut()")
	}
}

func (suite *sessionRecoverySuite) TestExpiredCacheRecovery() {
	scope := New().(*realScope)
	scope.userCache = newTestCache()

	mock := multiclient.NewMock()
	stub := &multiclient.Stub{
		Service:  loginService,
		Endpoint: readSessionEndpoint,
		Response: &sessreadproto.Response{
			SessId: proto.String(testSessId),
			Token:  proto.String(testToken),
		},
	}
	mock.Stub(stub)
	multiclient.SetCaller(mock.Caller())

	u, err := FromSessionToken(testSessId, testToken)
	suite.Assertions.Equal(testSessId, u.SessId)
	suite.Assertions.NoError(err)
	suite.Assertions.NotNil(u)
	u.RenewTs = time.Now().Add(-2 * time.Minute)
	u.ExpiryTs = u.RenewTs.Add(1 * time.Minute)
	suite.Assertions.NoError(scope.userCache.Store(u))
	suite.Assertions.Equal(0, stub.CountCalls())

	// As this session has expired, the login service should be called to recover (and renew) it
	suite.Assertions.False(scope.IsAuth())
	suite.Assertions.NoError(scope.RecoverSession(testSessId))
	suite.Assertions.True(scope.IsAuth())
	u = scope.AuthUser()
	suite.Assertions.NotNil(u)
	suite.Assertions.Equal(1, stub.CountCalls())
}
