package domain

// policies stores our hard-coded list of policies per-application
// we're hardcoding right now because we don't trust config service to be secure,
// we don't expect them to change often and if they do we want it to be more than
// a config service change (in terms of a barrier to people changing them)
// basically we are prioritising safety/security over ease of configuration
var policies map[Application]Policy = map[Application]Policy{
	Application("DRIVER"): {
		NewPasswordChecks: []PasswordAssertion{
			MinimumPasswordLength(5),
		},
	},
	Application("PASSENGER"): {
		NewPasswordChecks: []PasswordAssertion{
			MinimumPasswordLength(5),
		},
	},
	Application("ADMIN"): {
		NewPasswordChecks: []PasswordAssertion{
			MinimumPasswordLength(8),
			HasUpperCaseChar(),
			HasLowerCaseChar(),
			HasNumericChar(),
			HasNotBeenUsedIn(4),
		},
		PasswordValidFor: 60,
	},
}

// defaultPolicy is what we do if no policy defined per-app
var defaultPolicy Policy = Policy{
	NewPasswordChecks: []PasswordAssertion{
		MinimumPasswordLength(5),
	},
}
