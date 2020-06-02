package signer

import (
	"github.com/HailoOSS/login-service/domain"
)

type Signer interface {
	// Sign returns a new, signed version of the passed token. Will block if keys are not yet loaded.
	Sign(t *domain.Token) (*domain.Token, error)
	// Verify verifies a token's signature is as expected. Will block if keys are not yet loaded.
	Verify(t *domain.Token) bool
}

var (
	defaultInstance Signer
)

func init() {
	defaultInstance = newDefaultSigner()
}

// Sign wraps defaultInstance.Sign
func Sign(t *domain.Token) (*domain.Token, error) {
	signed, err := defaultInstance.Sign(t)
	return signed, err
}

// Verify wraps defaultInstance.Verify
func Verify(t *domain.Token) bool {
	return defaultInstance.Verify(t)
}
