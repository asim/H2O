package server

import (
	"testing"
	"time"
)

func TestBeat(t *testing.T) {
	hb := newHeartbeat(5 * time.Second)

	// pretend last heartbeat was 5 seconds ago
	last := time.Now().Add(-4 * time.Second)
	hb.last = last

	// beat
	hb.beat()

	if !hb.last.After(last) {
		t.Error("Beat should update to newer timestamp")
	}

	if !hb.healthy() {
		t.Error("Heartbeat should be healthy")
	}
}

func TestUnhealthy(t *testing.T) {
	hb := newHeartbeat(2 * time.Second)

	// pretend last heartbeat was 4 seconds ago
	hb.last = time.Now().Add(-4 * time.Second)

	if hb.healthy() {
		t.Error("Heartbeat should be unhealthy")
	}
}
