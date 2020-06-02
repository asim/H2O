package server

// use stddev to spot it becoming unhealthy

import (
	"sync"
	"time"
)

type heartbeat struct {
	mu sync.RWMutex

	last    time.Time
	maxDiff time.Duration
}

func newHeartbeat(maxDiff time.Duration) *heartbeat {
	return &heartbeat{
		last:    time.Now(),
		maxDiff: maxDiff,
	}
}

func (self *heartbeat) beat() {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.last = time.Now()
}

func (self *heartbeat) healthy() bool {
	self.mu.RLock()
	defer self.mu.RUnlock()

	if t := self.last.Add(self.maxDiff); t.After(time.Now()) {
		return true
	}

	return false
}
