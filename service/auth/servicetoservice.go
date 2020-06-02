package auth

// this implementation would ideally be replaced with something which just loads these
// rules from the config service, and then verifies they are OK using a public key
// (so the authority comes from knowledge that login service put them _in_ to config)

import (
	"fmt"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/davegardnerisme/deephash"
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/platform/client"

	endpointauth "github.com/HailoOSS/login-service/proto/endpointauth"
)

const (
	reloadInterval = 5 * time.Minute
	retryInterval  = 5 * time.Second
)

type role string

type grantedServices map[string]role

// serviceToService is how we store/load "service to service" auth rules right now
type serviceToService struct {
	sync.RWMutex

	myService string
	endpoints map[string]grantedServices
}

func newServiceToService() *serviceToService {
	s := &serviceToService{
		endpoints: make(map[string]grantedServices),
	}

	// update config occasionally
	ch := make(chan bool, 1)
	go func() {
		tick := time.NewTicker(reloadInterval)
		defer tick.Stop()
		for {
			// block until we need to reload/load
			select {
			case <-ch:
			case <-tick.C:
			}
			// block until we succeed
			for {
				if err := s.load(); err != nil {
					log.Warnf("[Auth] Failed to load service-to-service authorisation rules: %v", err)
				} else {
					break
				}
				time.Sleep(retryInterval)
			}
		}
	}()
	ch <- true

	return s
}

// setService defines the service name for THIS service and influences which rules we load
func (s *serviceToService) setService(name string) {
	s.Lock()
	s.myService = name
	s.Unlock()

	// now load config - which locks as well once it's done
	go s.load()
}

// getService locks and fetches the service name
func (s *serviceToService) getService() string {
	s.RLock()
	defer s.RUnlock()
	svc := s.myService

	return svc
}

// load config via login service
func (s *serviceToService) load() error {
	svc := s.getService()
	if svc == "" {
		log.Debug("[Auth] Skipping loading service-to-service auth rules (no service defined)")
		return nil
	}

	log.Tracef("[Auth] Loading service-to-service auth rules for %s", svc)
	reqProto := &endpointauth.Request{
		Service: proto.String(svc),
	}
	req, err := client.NewRequest("com.HailoOSS.service.login", "endpointauth", reqProto)
	if err != nil {
		return err
	}
	// scope it
	req.SetFrom(svc)
	rsp := &endpointauth.Response{}
	if err := client.Req(req, rsp); err != nil {
		return err
	}

	newEndpoints := make(map[string]grantedServices)

	for _, ep := range rsp.GetEndpoints() {
		name := ep.GetEndpoint()
		if _, ok := newEndpoints[name]; !ok {
			newEndpoints[name] = make(grantedServices)
		}
		// add in the granted services to this endpoint
		for _, gs := range ep.GetGranted() {
			newEndpoints[name][gs.GetName()] = role(gs.GetRole())
		}
	}

	// check if changed - to avoid locking/changing/logging if not
	if hashEndpoints(newEndpoints) == s.hash() {
		return nil
	}

	// switch in config
	s.Lock()
	defer s.Unlock()
	s.endpoints = newEndpoints

	log.Debugf("[Auth] Loaded service-to-service auth rules: %#v", s.endpoints)

	return nil
}

// assumedRole tests if we have service-to-service role authorisation
func (s *serviceToService) assumedRole(toEndpoint, fromService string) string {
	s.RLock()
	defer s.RUnlock()

	if grants, ok := s.endpoints[toEndpoint]; ok {
		if role, serviceAllowed := grants[fromService]; serviceAllowed {
			return string(role)
		}
	}

	return ""
}

// hash returns a hash of all currently loaded rules
func (s *serviceToService) hash() string {
	s.RLock()
	defer s.RUnlock()

	return hashEndpoints(s.endpoints)
}

func hashEndpoints(m map[string]grantedServices) string {
	return fmt.Sprintf("%x", deephash.Hash(m))
}
