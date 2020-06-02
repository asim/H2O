package domain

import (
	"testing"
	"time"
)

func TestSetDriverPassword(t *testing.T) {
	user := &User{
		App:             Application("DRIVER"),
		Uid:             "LON1234",
		Ids:             []Id{},
		Created:         time.Now(),
		Roles:           []string{"DRIVER"},
		PasswordHistory: make([][]byte, 0),
		PasswordChange:  time.Time{},
	}
	// set a bunch of passwords that should all succeed
	testCases := []struct {
		s     string
		valid bool
	}{
		{"foobar", true},
		{"foobar", true}, // repeated is fine
		{"foobar", true}, // repeated is fine again!
		{"fooba", true},
		{"FooBar", true},
		{"FOOBAR123", true},
		{"12345", true},
		{" 1234", true},
		{"     ", true}, // @todo do we want this to be allowed?
		{"1234", false}, // too short
		{"123", false},  // too short
		{"12", false},   // too short
		{"1", false},    // too short
		{"", false},     // too short
		{"a", false},    // too short
		{"ABCD", false}, // too short
	}

	for _, tc := range testCases {
		err := user.SetPassword(tc.s)
		ok := true
		if err != nil {
			ok = false
		}
		if ok != tc.valid {
			t.Errorf("Expected password %v to return %v", tc.s, tc.valid)
		}
	}
}

func TestSetElasticRidePassword(t *testing.T) {
	user := &User{
		App:             Application("ADMIN"),
		Uid:             "dave",
		Ids:             []Id{},
		Created:         time.Now(),
		Roles:           []string{"ADMIN"},
		PasswordHistory: make([][]byte, 0),
		PasswordChange:  time.Time{},
	}
	// set a bunch of passwords that should all succeed
	testCases := []struct {
		s     string
		valid bool
	}{
		{"password1", false},   // no upper
		{"PASSWORD1", false},   // no lower
		{"Password", false},    // no number
		{"Ab345", false},       // too short
		{"Password1", true},    // password policy ftw!	1st pass set
		{"fooBar11!£", true},   // having chars is ok		2nd pass set
		{"Password1", false},   // can't reuse this yet
		{"Password12", true},   // ok						3rd pass set
		{"Password1", false},   // can't reuse this yet
		{"Password123", true},  // ok						4th pass set
		{"Password1", false},   // can't reuse this yet
		{"Password1234", true}, // ok						4th pass set
		{"Password1", true},    // can finally reuse this!
		{"Password1", false},   // but not again
	}

	for _, tc := range testCases {
		err := user.SetPassword(tc.s)
		ok := true
		if err != nil {
			ok = false
		}
		if ok != tc.valid {
			t.Errorf("Expected password %v to return %v", tc.s, tc.valid)
		}
	}
}
