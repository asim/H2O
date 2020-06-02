package signer

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
)

// NOTE: verify and sign are both written such that they are compatible with the
// original PHP version of the login service (github.com/HailoOSS/login-service)

// verify will ensure that the signature is valid for this data
func verify(pub *rsa.PublicKey, hash crypto.Hash, sig, data []byte) (bool, error) {
	h := hash.New()
	h.Write(data)
	digest := h.Sum(nil)
	if err := rsa.VerifyPKCS1v15(pub, hash, digest, sig); err != nil {
		return false, fmt.Errorf("Failed to verify signature: %v", err)
	}

	return true, nil
}

// sign will sign the data to generate and return signature -- note that the data
// here has NOT been hashed
func sign(prv *rsa.PrivateKey, hash crypto.Hash, data []byte) ([]byte, error) {
	h := hash.New()
	h.Write(data)
	digest := h.Sum(nil)
	sig, err := rsa.SignPKCS1v15(rand.Reader, prv, hash, digest)
	if err != nil {
		return []byte{}, fmt.Errorf("Failed to sign: %v", err)
	}

	return sig, nil
}

// loadKeyToBytes grabs the pure data from a key file and returns as []byte
func loadKeyToBytes(fn string) ([]byte, error) {
	// load key from file
	f, err := os.Open(fn)
	if err != nil {
		return []byte{}, fmt.Errorf("Could not read key '%v': %v", fn, err)
	}
	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return []byte{}, fmt.Errorf("Could not read key '%v': %v", fn, err)
	}

	return bytes, nil
}

// bytesToPublicKey parses []byte into a PublicKey
func bytesToPublicKey(bytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(bytes)
	if block == nil {
		return nil, fmt.Errorf("Failed to decode public key - PEM decode failed")
	}
	someKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse PKIX public key: %v", err)
	}
	pubKey, ok := someKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Failed to cast to RSA public key")
	}

	return pubKey, nil
}

// bytesToPrivateKey parses []byte into a PrivateKey
func bytesToPrivateKey(bytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(bytes)
	if block == nil {
		return nil, fmt.Errorf("Failed to decode private key - PEM decode failed")
	}
	prvKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse PKCS1 private key: %v", err)
	}

	return prvKey, nil
}
