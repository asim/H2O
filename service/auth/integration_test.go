// +build integration

// (relies on having running memcache, config service, login service ...)

package auth

import (
	"sync"
	"testing"

	"github.com/HailoOSS/platform/raven"
	"github.com/HailoOSS/service/config"
)

var setupOnce sync.Once

func setupTests() {
	if online := <-raven.Connect(); !online {
		panic("Failed to connect raven")
	}
}

func TestInvalidTokenRecovery(t *testing.T) {
	setupOnce.Do(setupTests)

	config.LoadFromService("testservice")

	driverToken := `foobarbaz`
	err := RecoverSession(driverToken)
	if err != nil {
		t.Fatalf("Recover session failed with an error: %v", err)
	}
	if IsAuth() {
		t.Error("Recover session was not expected to recover a user")
	}
}

func TestDriverTokenRecovery(t *testing.T) {
	setupOnce.Do(setupTests)

	config.LoadFromService("testservice")

	driverToken := `fYNtPGwtJrSWwhJaGzXEVHspevtgAdbUYToh/6aEJf7ZzpsZl06v63FjA4and1nfQeoU5lrSgwSogCDmK2bcpQ==`
	err := RecoverSession(driverToken)
	if err != nil {
		t.Fatalf("Recover session failed with an error: %v", err)
	}
	if !IsAuth() {
		t.Error("Driver token should have been automatically recovered")
	}
}
