package signer

import (
	"crypto"
	"crypto/rsa"
	"sort"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/login-service/domain"
	"time"
)

// the key locations are hardcoded rather than config service such that
// they can't be changed by someone who has access to config service
const (
	privateKeyFn = "/opt/hailo/login-service/private-key"
	publicKeyFn  = "/opt/hailo/login-service/public-key"
)

// defaultSigner is the default implementation - we read prv/pub key from file
type defaultSigner struct {
	pub  *rsa.PublicKey
	prv  *rsa.PrivateKey
	hash crypto.Hash
	// This channel is closed when the keys are loaded (used by WaitForLoad)
	loadedChan       chan struct{}
	loadedChanClosed bool
}

func newDefaultSigner() *defaultSigner {
	log.Infof("Initialising default signer using keys: %v, %v", privateKeyFn, publicKeyFn)
	s := &defaultSigner{
		hash:       crypto.SHA1,
		loadedChan: make(chan struct{}),
	}

	go s.lazyLoadKeys()
	return s
}

// lazyLoadKeys is ment for lazy private/public key initialization and ideally should be run
// in a separate goroutine
func (s *defaultSigner) lazyLoadKeys() {
	log.Debugf("[Lazy key initialiser] Loading private and public keys: %v, %v", privateKeyFn, publicKeyFn)
	var prvKeyLoaded, pubKeyLoaded bool
	retryDelay := time.Second * 10

	// block until we load
	for {
		if !pubKeyLoaded {
			// try to load pub key
			pubBytes, err := loadKeyToBytes(publicKeyFn)
			if err != nil {
				log.Warnf("Lazy key initialiser] Failed to load default signer PUB key: %v", err)
			} else {
				var err error
				s.pub, err = bytesToPublicKey(pubBytes)
				if err != nil {
					log.Warnf("Lazy key initialiser] Failed to load default signer PUB key: %v", err)
				}
				log.Infof("[Lazy key initialiser] Successfully loaded the public key")
				pubKeyLoaded = true
			}
		}

		if !prvKeyLoaded {
			// try to load prv key
			prvBytes, err := loadKeyToBytes(privateKeyFn)
			if err != nil {
				log.Warnf("Lazy key initialiser] Failed to load default signer PRV key: %v", err)
			} else {
				s.prv, err = bytesToPrivateKey(prvBytes)
				if err != nil {
					log.Warnf("[Lazy key initialiser] Failed to load default signer PRV key: %v", err)
				}
				log.Infof("[Lazy key initialiser] Successfully loaded the private key")
				prvKeyLoaded = true
			}
		}

		if pubKeyLoaded && prvKeyLoaded {
			// If loadedChan is not already closed, close it (so WaitForLoad() stops blocking on it)
			if !s.loadedChanClosed {
				close(s.loadedChan)
				s.loadedChanClosed = true
			}
			break
		}

		time.Sleep(retryDelay)
	}

	log.Debug("[Lazy key initialiser] Exiting")
}

// Sign will generate and add a signature to a token, using the configured private key
func (s *defaultSigner) Sign(t *domain.Token) (*domain.Token, error) {
	s.waitForLoad()
	sort.Strings(t.Roles)
	sig, err := sign(s.prv, s.hash, t.DataToSign())
	if err != nil {
		return nil, err
	}
	tsig := t.Copy()
	tsig.Sign(sig)
	return tsig, nil
}

// Validate will test whether a token's signature validates using the public key
func (s *defaultSigner) Verify(t *domain.Token) bool {
	s.waitForLoad()
	ok, err := verify(s.pub, s.hash, t.DecodedSig(), t.DataToSign())
	if err != nil {
		log.Debugf("Error verifying: %v", err)
	}
	return ok
}

func (s *defaultSigner) waitForLoad() {
	<-s.loadedChan
}
