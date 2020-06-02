package domain

import (
	"fmt"
	"regexp"
)

var (
	regexpUpper   = regexp.MustCompile("[A-Z]{1}")
	regexpLower   = regexp.MustCompile("[a-z]{1}")
	regexpNumeric = regexp.MustCompile("[0-9]{1}")
)

// MinimumPasswordLength mints a PasswordAssertion to test for a minimum number of chars in password
func MinimumPasswordLength(length int) PasswordAssertion {
	return func(newPass string, user *User) error {
		if len([]rune(newPass)) < length {
			return fmt.Errorf("must be %v characters in length or more", length)
		}
		return nil
	}
}

// HasUpperCaseChar mints a PasswordAssertion to test a password contains an uppercase character
func HasUpperCaseChar() PasswordAssertion {
	return func(newPass string, user *User) error {
		if regexpUpper.MatchString(newPass) == false {
			return fmt.Errorf("must include an uppercase character")
		}

		return nil
	}
}

// HasLowerCaseChar mints a PasswordAssertion to test a password contains an lowercase character
func HasLowerCaseChar() PasswordAssertion {
	return func(newPass string, user *User) error {
		if regexpLower.MatchString(newPass) == false {
			return fmt.Errorf("must include a lowercase character")
		}
		return nil
	}
}

// HasNumericChar mints a PasswordAssertion to test a password contains a number
func HasNumericChar() PasswordAssertion {
	return func(newPass string, user *User) error {
		if regexpNumeric.MatchString(newPass) == false {
			return fmt.Errorf("must include a numeric character")
		}
		return nil
	}
}

// HasNotBeenUsedIn mints a PasswordAssertion to test a password hasn't been used in N recent password changes
func HasNotBeenUsedIn(changes int) PasswordAssertion {
	return func(newPass string, user *User) error {
		if user.InPasswordHistory([]byte(newPass), changes) {
			return fmt.Errorf("password has been used within the last %v changes", changes)
		}
		return nil
	}
}
