package handler

import (
	"testing"
	"time"

	"github.com/HailoOSS/login-service/domain"
)

func TestRoles(t *testing.T) {
	user := &domain.User{
		App:             domain.Application("DRIVER"),
		Uid:             "LON1234",
		Ids:             []domain.Id{},
		Created:         time.Now(),
		Roles:           []string{},
		PasswordHistory: make([][]byte, 0),
		PasswordChange:  time.Time{},
	}
	// set a bunch of passwords that should all succeed
	testCases := []struct {
		roles []string
		valid bool
	}{
		{[]string{"foobar"}, false},
		{[]string{"FOO"}, true},
		{[]string{".FOO"}, false},
		{[]string{"FOO."}, false},
		{[]string{"FOO.BAR"}, true},
		{[]string{"FOO.BAR.BAZ"}, true},
		{[]string{"Foo.BAR.BAZ"}, false},
		{[]string{"H4BADMIN.a400cb6d-8187-4546-4389-5c9f58a2df28"}, true},
		{[]string{"H4BADMIN.280372650114322432"}, true},
		{[]string{"FOO..BAR"}, false},
	}

	for _, tc := range testCases {
		user.Roles = tc.roles
		ok := true
		if errs := userValidator.Validate(user); errs.AnyErrors() {
			ok = false
		}
		if ok != tc.valid {
			t.Errorf("Expected role %v to be %v", tc.roles, tc.valid)
		}
	}
}
