package server

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// registry keeps track of endpoints we have registered
type registry struct {
	sync.RWMutex
	endpoints  map[string]*Endpoint
	middleware []Middleware
}

// newRegistry mints a new registry
func newRegistry() *registry {
	return &registry{
		endpoints:  make(map[string]*Endpoint, 5),
		middleware: make([]Middleware, 0, 5),
	}
}

// add will add an endpoint and enforce some basic laws, like lowercase names, and some Authoriser
func (r *registry) add(ep *Endpoint) (err error) {
	if len(ep.Name) == 0 {
		err = fmt.Errorf("Missing name in endpoint: %+v", ep)
		return
	}
	lowerName := strings.ToLower(ep.Name)
	if lowerName != ep.Name {
		err = fmt.Errorf("Endpoint name should be lowercase: %+v", ep)
		return
	}

	// add a default Authoriser, if none
	if ep.Authoriser == nil || reflect.ValueOf(ep.Authoriser).IsNil() {
		ep.Authoriser = DefaultAuthoriser
	}

	// Apply any registered middleware
	h := ep.Handler
	for _, m := range r.middleware {
		h = m(ep, h)
	}
	ep.Handler = h

	r.Lock()
	defer r.Unlock()
	r.endpoints[ep.Name] = ep

	return
}

// add will add a middleware, these will be applied to any endpoints registered
// after this call
func (r *registry) addMiddleware(mw Middleware) (err error) {
	r.Lock()
	defer r.Unlock()
	r.middleware = append(r.middleware, mw)

	return
}

// find will find an endpoint by name from within the registry
func (r *registry) find(epName string) (ep *Endpoint, ok bool) {
	r.RLock()
	defer r.RUnlock()

	ep, ok = r.endpoints[epName]
	return
}

// iterate locks, copies and returns a snapshot of registered endpoints
func (r *registry) iterate() []*Endpoint {
	r.RLock()
	defer r.RUnlock()

	ret := make([]*Endpoint, len(r.endpoints))
	i := 0
	for _, ep := range r.endpoints {
		ret[i] = ep
		i++
	}
	return ret
}

// size counts up how many registered endpoints
func (r *registry) size() int {
	r.RLock()
	defer r.RUnlock()

	return len(r.endpoints)
}
