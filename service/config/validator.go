package config

// Validator function takes config as slice of bytes, evaluates based on some rules
// and returns whether the config is valid as boolean.
type Validator func([]byte) bool

// AddValidator adds a validator function to the default config instance.
func AddValidator(v Validator) {
	DefaultInstance.AddValidator(v)
}
