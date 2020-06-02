package localisation

import (
	"fmt"
	"strings"
	"sync"

	log "github.com/cihub/seelog"
	"github.com/golang/groupcache/lru"

	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// HobDiscriminator defines a function that can determine a HOB for a given server request. If no HOB can be determined,
// an empty string should be returned.
type HobDiscriminator func(req *server.Request) (hob string)

// cachingHobDiscriminator is a decorator for a HobDiscriminator, memoising the result (per-request) using a LRU cache
func cachingHobDiscriminator(discriminatorImpl HobDiscriminator) HobDiscriminator {
	resultCache := lru.New(300)
	var cacheLock sync.RWMutex // lru isn't concurrency safe

	return func(req *server.Request) (hob string) {
		reqId := req.MessageID()

		// Try to get the result from the cache
		cacheLock.RLock()
		_hob, ok := resultCache.Get(reqId)
		cacheLock.RUnlock()

		if ok && _hob != nil {
			// Cache hit yey
			hob = _hob.(string)
		} else {
			// It wasn't in the cache; compute it and store it in the cache
			hob = discriminatorImpl(req)
			cacheLock.Lock()
			resultCache.Add(reqId, hob)
			cacheLock.Unlock()
		}

		return hob
	}
}

type hobAuthoriser struct {
	roles            []string
	authoriserImpl   func([]string) server.Authoriser
	hobDiscriminator HobDiscriminator
}

func (ha *hobAuthoriser) Authorise(req *server.Request) errors.Error {
	// Try and authorise *without* consulting the hobDiscriminator (as it may potentially have to do lots of work that
	// we can save ourselves from). If they have a more global role than the HOB-specific one, then party on.
	if err := ha.authoriserImpl(ha.roles).Authorise(req); err == nil {
		log.Tracef("[HOB authoriser] Matched non-HOB-specific role; not calling the HOB discriminator")
		return nil
	}

	hob := ha.hobDiscriminator(req)
	var reqRoles []string

	if hob == "" {
		reqRoles = ha.roles
	} else {
		reqRoles = make([]string, len(ha.roles))
		for i, role := range ha.roles {
			reqRoles[i] = fmt.Sprintf("%s.%s", role, hob)
		}
	}

	err := ha.authoriserImpl(reqRoles).Authorise(req)
	if err != nil {
		log.Debugf("[HOB authoriser] Failed to authorise in HOB %s", hob)
	}
	return err
}

// Checks the roles passed are valid for use in a hobAuthoriser. Panics if they are not. As this happens at service
// initialisation time, it will be immediately obvious to a service author if not.
func validateHobRoles(roles []string) {
	for _, role := range roles {
		if strings.HasSuffix(role, ".*") {
			panic("HOB-specific roles may not have a .* suffix")
		}
	}
}

// HobRoleAuthoriser requires a service or user calling an endpoint to have ANY of the roles passed, which will be
// scoped to a particular HOB (as determined by hobDiscriminator). Following the convention of a HOB being the last
// portion of a role, the matched role(s) will be of the form "{{ role }}.{{ HOB }}" (or just "{{ role }}" if the
// hobDiscriminator returns "").
func HobRoleAuthoriser(roles []string, hobDiscriminator HobDiscriminator) server.Authoriser {
	return &hobAuthoriser{
		roles:            roles,
		authoriserImpl:   server.RoleAuthoriser,
		hobDiscriminator: cachingHobDiscriminator(hobDiscriminator),
	}
}

// SignInHobRoleAuthoriser requires a real person signed in calling an endpoint to have ANY of the passed roles, which
// will be scoped to a particular HOB (as determined by hobDiscriminator). Following the convention of a HOB being the
// last portion of a role, the matched role(s) will be of the form "{{ role }}.{{ HOB }}" (or just "{{ role }}" if the
// hobDiscriminator returns "").
func SignInHobRoleAuthoriser(roles []string, hobDiscriminator HobDiscriminator) server.Authoriser {
	return &hobAuthoriser{
		roles:            roles,
		authoriserImpl:   server.SignInRoleAuthoriser,
		hobDiscriminator: cachingHobDiscriminator(hobDiscriminator),
	}
}
