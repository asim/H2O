// Heartbeat package allows us to keep track of heartbeats received against some UID and judge health based on whether received
package heartbeat

import (
	"fmt"
	"sync"
	"time"
)

// Heartbeat keeps track of inbound ticks for something, and can thus judge health (if not received)
type Heartbeat struct {
	sync.RWMutex

	Id      string
	MaxDiff time.Duration
	last    time.Time
}

// New mints a new healthy heartbeat
func New(id string, maxDiff time.Duration) *Heartbeat {
	return &Heartbeat{
		Id:      id,
		last:    time.Now(),
		MaxDiff: maxDiff,
	}
}

// ID returns a unique ID for the heartbeat
func (self *Heartbeat) ID() string {
	return self.Id
}

// String satisfies Stringer
func (self *Heartbeat) String() string {
	healthy := "UNHEALTHY"
	if self.Healthy() {
		healthy = "HEALTHY"
	}
	return fmt.Sprintf("[Discovery] Heartbeat %v: %v [%v]", self.Id, healthy, self.Last())
}

// Last yields the last time this heart beated
func (self *Heartbeat) Last() time.Time {
	self.RLock()
	defer self.RUnlock()

	return self.last
}

// Beat records a heartbeat activity
func (self *Heartbeat) Beat() {
	self.Lock()
	defer self.Unlock()

	self.last = time.Now()
}

// Healthy judges whether this heartbeat is healthy or not
func (self *Heartbeat) Healthy() bool {
	cutOff := self.Last().Add(self.MaxDiff)

	if cutOff.After(time.Now()) {
		return true
	}

	return false
}

// Payload yields the payload we use for RMQ when we send a heartbeat
func (self *Heartbeat) Payload() []byte {
	return []byte("PING")
}

// ContentType yields the RMQ content type for heartbeats
func (self *Heartbeat) ContentType() string {
	return "text/plain"
}
