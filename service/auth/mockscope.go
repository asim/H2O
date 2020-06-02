package auth

import (
	"github.com/HailoOSS/platform/multiclient"
)

type MockScope struct {
	MockUid    string
	MockRoles  []string
	authorised bool
}

func (s *MockScope) MockUser(id string, roles []string) {
	s.MockUid = id
	s.MockRoles = roles
}

func (s *MockScope) RpcScope(scoper multiclient.Scoper) Scope                { return s }
func (s *MockScope) Clean() Scope                                            { return s }
func (s *MockScope) RecoverSession(sessId string) error                      { return nil }
func (s *MockScope) RecoverService(toEndpoint, fromService string) error     { return nil }
func (s *MockScope) Auth(mech, device string, creds map[string]string) error { return nil }
func (s *MockScope) SignOut(user *User) error                                { return nil }
func (s *MockScope) HasTriedAuth() bool                                      { return true }
func (s *MockScope) IsAuth() bool                                            { return len(s.MockUid) > 0 }
func (s *MockScope) HasAccess(role string) bool                              { return matchRoleAgainstSet(role, s.MockRoles) }
func (s *MockScope) AuthUser() *User {
	return &User{
		SessId: "test-sess-id",
		Mech:   "mock",
		Device: "test",
		Id:     s.MockUid,
		Roles:  s.MockRoles,
	}
}
func (s *MockScope) Authorised() bool              { return s.authorised }
func (s *MockScope) SetAuthorised(authorised bool) { s.authorised = authorised }
