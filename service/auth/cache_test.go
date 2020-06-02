package auth

import (
	"errors"
)

// testCache is for testing
type testCache struct {
	users       map[string]*User
	invalidated map[string]bool
	failure     bool
}

func newTestCache() *testCache {
	return &testCache{
		users:       make(map[string]*User),
		invalidated: make(map[string]bool),
	}
}

func (c *testCache) Store(u *User) error {
	if c.failure {
		return errors.New("Simulated failure")
	}
	c.users[u.SessId] = u
	return nil
}

func (c *testCache) Invalidate(sessId string) error {
	if c.failure {
		return errors.New("Simulated failure")
	}
	c.invalidated[sessId] = true
	if _, ok := c.users[sessId]; ok {
		delete(c.users, sessId)
	}
	return nil
}

func (c *testCache) Fetch(sessId string) (*User, bool, error) {
	if c.failure {
		return nil, false, errors.New("Simulated failure")
	}
	if c.invalidated[sessId] { // invalidated user cached - hit, but no user
		return nil, true, nil
	}
	if u, ok := c.users[sessId]; ok { // user found - hit with user
		return u, true, nil
	}
	return nil, false, nil // no user found in cache - cache miss
}

func (c *testCache) Purge(sessId string) error {
	if c.failure {
		return errors.New("Simulated failure")
	}
	if _, ok := c.users[sessId]; ok {
		delete(c.users, sessId)
	}
	if _, ok := c.invalidated[sessId]; ok {
		delete(c.invalidated, sessId)
	}
	return nil
}
