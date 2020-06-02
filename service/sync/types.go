package sync

// Leader is an interface used by return values on things that can elect a leader
type Leader interface {
	// Rescinded returns a channel that will be closed if/when leadership is rescinded
	Rescinded() chan struct{}
	// Rescind allows clients to manually rescind leadership
	Rescind()
}

// Lock is an interface used by return values on things that can achieve a lock
type Lock interface {
	// Unlock allows clients to release the lock
	Unlock()
}
