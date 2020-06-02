package auth

import (
	"bytes"
	"time"

	log "github.com/cihub/seelog"

	inst "github.com/HailoOSS/service/instrumentation"
	mc "github.com/HailoOSS/service/memcache"
	"github.com/HailoOSS/gomemcache/memcache"
)

const (
	invalidPlaceholder = "invalid"
	invalidateTimeout  = 3600
)

type Cacher interface {
	Store(u *User) error
	Invalidate(sessId string) error
	Fetch(sessId string) (u *User, cacheHit bool, err error)
	Purge(sessId string) error
}

type memcacheCacher struct{}

// Store will add a user to our token cache; non-nil error indicates we failed
// to add them to the token cache
func (c *memcacheCacher) Store(u *User) error {
	t := time.Now()
	err := c.doStore(u)
	instTiming("auth.cache.store", err, t)
	return err
}

func (c *memcacheCacher) doStore(u *User) error {
	ttl := int32(0)
	if !u.ExpiryTs.IsZero() {
		ttl = int32(u.ExpiryTs.Sub(time.Now()).Seconds())
	}
	return mc.Set(&memcache.Item{
		Key:        u.SessId,
		Value:      u.Token,
		Expiration: ttl,
	})
}

// Invalidate will keep track of the fact this sessId is not valid, to save
// us having to continually look it up with the login service; non-nil error
// indicates we failed to invalidate this in the cache
func (c *memcacheCacher) Invalidate(sessId string) error {
	t := time.Now()
	err := c.doInvalidate(sessId)
	instTiming("auth.cache.invalidate", err, t)
	return err
}

func (c *memcacheCacher) doInvalidate(sessId string) error {
	return mc.Set(&memcache.Item{
		Key:        sessId,
		Value:      []byte(invalidPlaceholder),
		Expiration: invalidateTimeout,
	})
}

// Fetch will attempt to retreive a user from token cache
// If cacheHit == true and u == nil and err == nil then we KNOW they don't
// exist (and so we don't have to bother looking them up via login service)
func (c *memcacheCacher) Fetch(sessId string) (u *User, cacheHit bool, err error) {
	t := time.Now()
	u, hit, err := c.doFetch(sessId)
	instTiming("auth.cache.fetch", err, t)
	if hit {
		inst.Counter(1.0, "auth.cache.fetch.hit", 1)
	} else {
		inst.Counter(1.0, "auth.cache.fetch.miss", 1)
	}
	return u, hit, err
}

func (c *memcacheCacher) doFetch(sessId string) (u *User, cacheHit bool, err error) {
	it, err := mc.Get(sessId)
	if err != nil && err != memcache.ErrCacheMiss {
		// actual error
		log.Warnf("[Auth] Token cache fetch error for '%s': %v", sessId, err)
		return nil, false, err
	}

	if err == memcache.ErrCacheMiss {
		// not found - not an error though
		log.Trace("[Auth] Token cache - miss")
		return nil, false, nil
	}

	if bytes.Equal(it.Value, []byte(invalidPlaceholder)) {
		// cached invalid
		log.Tracef("[Auth] Token cache - invalid placeholder in cache for %s", sessId)
		return nil, true, nil
	}

	u, err = FromSessionToken(sessId, string(it.Value))
	if err != nil {
		// found, but we can't decode - treat as not found
		log.Warnf("[Auth] Token cache decode error: %v", err)
		return nil, false, nil
	}

	return u, true, nil
}

// Purge will remove knowledge about a sessId from the token cache. If the
// sessId doesn't exist then this will be classed as success. Non-nil error
// indicates we failed to remove this cache key.
func (c *memcacheCacher) Purge(sessId string) error {
	t := time.Now()
	err := c.doPurge(sessId)
	instTiming("auth.cache.purge", err, t)

	return err
}

func (c *memcacheCacher) doPurge(sessId string) error {
	if err := mc.Delete(sessId); err != nil {
		return err
	}

	return nil
}
