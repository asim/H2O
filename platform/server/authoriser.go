package server

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/platform/errors"
	inst "github.com/hailo-platform/H2O/service/instrumentation"
)

// Authoriser is an interface that anything that wants to authorise endpoints must satisfy
type Authoriser interface {
	// Authorise should check the request and then return an error if problems
	Authorise(req *Request) errors.Error
}

var DefaultAuthoriser Authoriser = RoleAuthoriser([]string{"ADMIN"})

// RoleAuthoriser requires a service or user calling an endpoint to have ANY of these roles
func RoleAuthoriser(roles []string) Authoriser {
	return &simpleAuthoriser{
		requireUser: false,
		requireRole: true,
		roles:       roles,
	}
}

// SignInAuthoriser requires a real person to be signed in, but doesn't care about roles
func SignInAuthoriser() Authoriser {
	return &simpleAuthoriser{
		requireUser: true,
		requireRole: false,
		roles:       []string{},
	}
}

// SignInRoleAuthoriser requires a real person signed in calling an endpoint to have ANY of these roles
func SignInRoleAuthoriser(roles []string) Authoriser {
	return &simpleAuthoriser{
		requireUser: true,
		requireRole: true,
		roles:       roles,
	}
}

// OpenToTheWorldAuthoriser means ANYONE (in the whole world) can call a service
func OpenToTheWorldAuthoriser() Authoriser {
	return &simpleAuthoriser{
		requireUser: false,
		requireRole: false,
		roles:       []string{},
	}
}

// simpleAuthoriser is a role-based authorisor which will check against defined roles
type simpleAuthoriser struct {
	// requireUser tells us if we need a real person to be authenticated (if true, then service-to-serivce auth won't work)
	requireUser bool
	// requireRole tells us if we need to check roles (if false, then we don't care what the roles are)
	requireRole bool
	// roles to match -- we require users to match AT LEAST ONE
	roles []string
}

// Given a request, returns a "bad role" error. This is used both by Authorise() and is useful within services which do
// row-level permission checking.
func BadRoleError(req *Request) errors.Error {
	return errors.Forbidden("com.hailo-platform/H2O.kernel.auth.badrole", fmt.Sprintf("Must have the correct role to call this "+
		"endpoint [endpoint=%s, service=%s, from=%s]", req.Endpoint(), req.Service(), req.From()), "5")
}

// Authorise tests auth
func (a *simpleAuthoriser) Authorise(req *Request) errors.Error {

	// If we require neither a role or a user, then there is no need to authorise
	if !a.requireUser && !a.requireRole {
		log.Tracef("Skipping auth from %s to %s, as neither user or role required", req.From(), req.Destination())
		return nil
	}

	// Otherwise, authorise this request
	scope := req.Auth()
	log.Tracef("Scope user: %v", scope.AuthUser())
	if a.requireUser && !scope.IsAuth() {
		return errors.Forbidden("com.hailo-platform/H2O.kernel.auth.notsignedin", fmt.Sprintf("Must be signed in to call this endpoint[endpoint=%s, service=%s, from=%s]",
			req.Endpoint(), req.Service(), req.From()), "201")
	}
	if a.requireRole {
		matchesRole := false
		for _, r := range a.roles {
			if scope.HasAccess(r) {
				matchesRole = true
				break
			}
		}
		if !matchesRole {
			if scope.HasTriedAuth() {
				return errors.Forbidden("com.hailo-platform/H2O.kernel.auth.badrole", fmt.Sprintf("Must be signed in to call this endpoint[endpoint=%s, service=%s, from=%s]",
					req.Endpoint(), req.Service(), req.From()), "201")
			}
			// Instrument when service to service auth fails
			inst.Counter(1.0, "auth.servicetoservice.failed", 1)
			return BadRoleError(req)
		}
	}
	return nil
}
