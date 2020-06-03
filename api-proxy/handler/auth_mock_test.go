package handler

import (
	"fmt"
	"testing"

	"github.com/hailo-platform/H2O/service/auth"
)

type mockAuthScope struct {
	auth.MockScope
	T                 testing.TB
	ExpectedSessionId string
}

func (s *mockAuthScope) RecoverSession(sessId string) error {
	if sessId != s.ExpectedSessionId {
		s.T.Logf("Session ID: %s; expected: %s", sessId, s.ExpectedSessionId)
		return fmt.Errorf("Session ID not expected: %s != %s", sessId, s.ExpectedSessionId)
	}
	return s.MockScope.RecoverSession(sessId)
}
