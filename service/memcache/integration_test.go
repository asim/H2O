// +build integration

// (relies on having running memcache and access to config service correctly configured)

package memcache

import (
	"bytes"
	"testing"
	"time"

	"github.com/HailoOSS/service/config"
	"github.com/HailoOSS/gomemcache/memcache"
)

func TestPub(t *testing.T) {
	config.LoadFromService("testservice")

	val := time.Now().Nanosecond()
	setVal := []byte(string(val))

	if err := Set(&memcache.Item{Key: "foo", Value: setVal}); err != nil {
		t.Fatalf("Failed to Set: %v", err)
	}

	it, err := Get("foo")
	if err != nil {
		t.Fatalf("Failed to Get: %v", err)
	}

	if bytes.Equal(it.Value, setVal) == false {
		t.Errorf("Retrieved doesn't match set")
	}
}
