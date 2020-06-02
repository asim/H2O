package auth

import (
	"crypto"
	"encoding/base64"
	"testing"
)

type mockValidatorImpl struct {
	// Holds the "real" validator while it's mocked
	shelvedValidator validator
}

func (v *mockValidatorImpl) verify(sig, data []byte) (bool, error) {
	return true, nil
}

// Replace the global validator with an implementation that will always validate
func mockValidator() {
	defaultValidator = &mockValidatorImpl{
		shelvedValidator: defaultValidator,
	}
}

// Set the global validator back to the real deal
func unmockValidator() {
	switch defaultValidator.(type) {
	case *mockValidatorImpl:
		defaultValidator = defaultValidator.(*mockValidatorImpl).shelvedValidator
	default:
	}
}

const publicKey = `-----BEGIN PUBLIC KEY-----
MIIBIDANBgkqhkiG9w0BAQEFAAOCAQ0AMIIBCAKCAQEAxEiAaFDgroIvK6U+d4GO
Rt6s9rHpDiZ/Mpf/IcaPDHKLrnFb+HrZTVM/AAkKG6AnqejBaJMUTUgnRqq2Zjzl
sodS668L3GBxv2IJfFrtX/bAMN43zonHthJlnTredPdfmtNS0B6QFyA32Y9VLdMP
9Nbum4KHUZJK86mpoTqhLBAWFXC3uWXQD98DItWkYZQ8AgM9f9/XFOUzKg+pMG+Q
C1bUPZg0oARfpMpZGw3ksQTmDj47pl6W/NhllnSeULjHWg23LCE6XL4I9cKgkIOH
hoOgxx4BTDTtm+t2C4Kq6H52buZFUbhuxi3/Vqw9OGFYZqQ/Dd9D3OJO3BA3LoT2
mQIBIw==
-----END PUBLIC KEY-----`

func setupKeys() {
	k, _ := bytesToKey([]byte(publicKey))
	defaultValidator = &validatorImpl{
		pub:  k,
		hash: crypto.SHA1,
	}
}

func TestValidator(t *testing.T) {
	setupKeys()
	sig, err := base64.StdEncoding.DecodeString(`OqmD7GCddj7uU0IKy3zflMBpTjnHFk6TG2wtaQTwZTPyC3g/qqE+Zrx0gIVDBb5x2VuXTPHFwjT9Vl85E4NEIy1GUon4GBLt264Kg4hMPxAxMdhcogbjaxmlOCcroCiKfJ06FdKvFvvQUup2tjLAek3XjOqIaPX/x7e7RZzITxYxSI7Mpbuhs0f5rzF1bYuH4/akeQdU1kODqVXpOWP+zJjDyMVATMxc69P7ijRvSKszgomb6m9vmsmpERQdvyNW09NBjGZlLbilPnZ3YKoaFZosYjIXTGNbeywGLf10N4t0qvP2Ms/Z1oNIeFdLqMfgicti00uv+bttTL1vUtBghA==`)
	if err != nil {
		t.Errorf("Failed to base64 decode: %v", err)
	}
	data := []byte(`am=admin:d=cli:id=dave:ct=1372956175:et=1372984975:rt=:r=ADMIN`)
	if _, err := verify(sig, data); err != nil {
		t.Errorf("Failed to validate: %v", err)
	}
}

func TestFromSessionToken(t *testing.T) {
	setupKeys()
	tok := `am=admin:d=cli:id=dave:ct=1372956175:et=1372984975:rt=:r=ADMIN:sig=OqmD7GCddj7uU0IKy3zflMBpTjnHFk6TG2wtaQTwZTPyC3g/qqE+Zrx0gIVDBb5x2VuXTPHFwjT9Vl85E4NEIy1GUon4GBLt264Kg4hMPxAxMdhcogbjaxmlOCcroCiKfJ06FdKvFvvQUup2tjLAek3XjOqIaPX/x7e7RZzITxYxSI7Mpbuhs0f5rzF1bYuH4/akeQdU1kODqVXpOWP+zJjDyMVATMxc69P7ijRvSKszgomb6m9vmsmpERQdvyNW09NBjGZlLbilPnZ3YKoaFZosYjIXTGNbeywGLf10N4t0qvP2Ms/Z1oNIeFdLqMfgicti00uv+bttTL1vUtBghA==`
	sess := `fktWghcPxwqoIMtykAPpPkwesLrAFACsRUExvsnqJRt1e1yCP0TuBJxfTocZgtbZ`
	if u, err := FromSessionToken(sess, tok); err != nil || u == nil {
		t.Errorf("Failed to make user from token (%v)", err)
	}
}
