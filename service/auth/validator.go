package auth

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/service/config"
)

var (
	defaultValidator   validator = newConfigServiceValidator()
	startRetryDelay              = time.Millisecond * 100
	maxRetryDelay                = time.Second * 10
	waitForConfigDelay           = time.Second * 2
)

// verify wraps defaultValidator.verify
func verify(sig, data []byte) (bool, error) {
	ok, err := defaultValidator.verify(sig, data)
	return ok, err
}

type validator interface {
	// verify executes a VerifyPKCS1v15 test against some data with the supplied sig
	verify(sig, data []byte) (bool, error)
}

// validator is responsible for verifying signatures against some public key
type validatorImpl struct {
	sync.RWMutex

	pub      *rsa.PublicKey
	lastRead []byte
	hash     crypto.Hash
}

// newConfigServiceValidator initiates a validator that loads public key location from config service
func newConfigServiceValidator() validator {
	v := &validatorImpl{}
	v.hash = crypto.SHA1

	ch := config.SubscribeChanges()
	immediate := make(chan bool)

	go func() {
		for {
			select {
			case <-ch:
			case <-immediate:
			}

			v.loadFromConfig()
		}
	}()

	// push immediate message down to force background reload
	immediate <- true

	return v
}

func (v *validatorImpl) verify(sig, data []byte) (bool, error) {
	v.RLock()
	defer v.RUnlock()

	if v.pub == nil {
		return false, errors.New("Public key is not loaded")
	}
	h := v.hash.New()
	h.Write(data)
	digest := h.Sum(nil)
	if err := rsa.VerifyPKCS1v15(v.pub, v.hash, digest, sig); err != nil {
		return false, err
	}

	return true, nil
}

// loadFromConfig including contiuous retries until we have managed to load it
func (v *validatorImpl) loadFromConfig() {
	if !config.WaitUntilLoaded(waitForConfigDelay) {
		// put out a warning anyway, to make it clear we are going to struggle to load key
		log.Warnf("[Auth] Failed to load config after %v, kicking off public key loading anyway...", waitForConfigDelay)
	}

	// block until we load
	attempts := 0
	for {
		if err := v.load(); err != nil {
			attempts++
			delay := time.Duration(int64(startRetryDelay) * int64(attempts))
			if delay > maxRetryDelay {
				delay = maxRetryDelay
			}
			log.Tracef("[Auth] Failed to load public key from config: %v (sleeping for %v)", err, delay)
			time.Sleep(delay)
			continue
		}
		break
	}
}

// load will load public key location from config service and switch pub key
func (v *validatorImpl) load() error {
	v.Lock()
	defer v.Unlock()

	fn := config.AtPath("hailo", "service", "authentication", "publicKey").AsString("")
	log.Tracef("[Auth] Loading auth library public key from: %s", fn)
	if fn == "" {
		return fmt.Errorf("public key filename undefined in config")
	}

	// load key from file
	f, err := os.Open(fn)
	if err != nil {
		return fmt.Errorf("Failed to open public key %s (%v)", fn, err)
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("Failed to read public key from %s (%v)", fn, err)
	}

	if bytes.Equal(b, v.lastRead) {
		// no change
		return nil
	}

	// turn bytes into an actual key instance
	k, err := bytesToKey(b)
	if err != nil {
		return fmt.Errorf("Failed to read public key from %s (%v)", fn, err)
	}

	log.Infof("[Auth] Loaded public key: %v", k)

	v.pub = k
	v.lastRead = b
	return nil
}

// bytesToKey turns raw bytes into a public key -- parsing it
func bytesToKey(bytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(bytes)
	if block == nil {
		return nil, fmt.Errorf("Failed to decode public key")
	}
	someKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse PKIX public key (%v)", err)
	}
	pubKey, ok := someKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Failed to cast to RSA public key")
	}

	return pubKey, nil
}
