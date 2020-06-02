package auth

import (
	"fmt"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/hashicorp/golang-lru"
	glob "github.com/obeattie/ohmyglob"
)

const h2AppMechPrefix = "h2."

var (
	compiledGlobSetCache *lru.Cache
)

func init() {
	var err error
	compiledGlobSetCache, err = lru.New(128)
	if err != nil {
		panic(err)
	}
}

type User struct {
	SessId, Mech, Device, Id     string
	CreatedTs, ExpiryTs, RenewTs time.Time
	Roles                        []string
	Token, Sig, Data             []byte
}

// Options which should be used whenever construting a role glob
var RoleGlobOptions = &glob.Options{
	Separator:    '.',
	MatchAtStart: true,
	MatchAtEnd:   true,
}

// CanAutoRenew tests if the token can be auto-renewed at this time (by the
// login service)
func (u *User) CanAutoRenew() bool {
	return !u.RenewTs.IsZero()
}

// Application returns the user's application (derived from the authentication mechanism).
// This is only available for H2-generated tokens; H1 tokens return an empty string.
func (u *User) Application() string {
	if strings.HasPrefix(u.Mech, h2AppMechPrefix) {
		return strings.TrimPrefix(u.Mech, h2AppMechPrefix)
	}

	return ""
}

// HasRole tests if the user has this role
// we test roles against the hierarchy, eg: you could have FOO.BAR where the
// most specific role is FOO.BAR but FOO automatically covers FOO.BAR
func (u *User) HasRole(r string) bool {
	globset, err := compileRolesToGlobset(u.Roles)
	if err != nil {
		log.Errorf("[Auth] Cannot compile user roles %v to globset: %s", u.Roles, err.Error())
		return false
	}

	result, err := matchRoleAgainstCompiledGlobset(r, u.Roles, globset)
	if err != nil {
		log.Errorf("[Auth] %s", err.Error())
	}
	return result
}

// Matches the given role against a role pattern set
func matchRoleAgainstSet(r string, patterns []string) bool {
	globs, err := compileRolesToGlobset(patterns)
	if err != nil {
		log.Errorf("[Auth matcher] Cannot compile user roles %s to glob set: %s", patterns, err.Error())
		return false
	}

	var result bool
	result, err = matchRoleAgainstCompiledGlobset(r, patterns, globs)
	if err != nil {
		log.Errorf("[Auth matcher] %s", err.Error())
	}
	return result
}

// Compiles a list of roles to a globset
func compileRolesToGlobset(patterns []string) (glob.GlobSet, error) {
	// Append .** to the patterns (old patterns were all prefix matches)
	p := make([]string, len(patterns))
	for i, pattern := range patterns {
		p[i] = fmt.Sprintf("%s.**", pattern)
	}

	// Check if globSet exists in cache
	cacheKey := strings.Join(p, "|")
	if globSet, ok := compiledGlobSetCache.Get(cacheKey); ok {
		return globSet.(glob.GlobSet), nil
	}

	globSet, err := glob.CompileGlobSet(p, RoleGlobOptions)
	if err != nil {
		return nil, err
	}

	// Add new globSet to cache
	compiledGlobSetCache.Add(cacheKey, globSet)

	return globSet, nil
}

// Matches a role against a compiled globset, and if there is no match, implements legacy behaviour
func matchRoleAgainstCompiledGlobset(r string, patterns []string, globs glob.GlobSet) (bool, error) {
	log.Tracef("[Auth matcher] Testing if role \"%s\" in globset %s", r, globs.String())
	match := globs.MatchString(r)

	if !match && strings.HasSuffix(r, ".*") {
		// wow many cruft, so horror
		//
		// Okay. About this. Old role definitions (which can and should be phased out) allowed pseudo-glob matches in
		// the role definition itself. So a role authoriser might allow "H4BADMIN.*" to allow admins of *any* H4B
		// organisation to access an endpoint. Here, to support that behaviour, we compile the role being matched to a
		// glob, and match it against the globs in the user's globset. Yes, you heard me: we match globs against globs.
		// Meta.
		// These old-style globs were only permitted as the last component of a path, and were optional. As .** allows
		// any number of components, that isn't appropriate. So we test for an exact match without the .* suffix, and a
		// .* match to match a single component.
		r = strings.TrimSuffix(r, ".*")
		r = glob.EscapeGlobComponent(r, RoleGlobOptions)
		roleGlob, err := glob.Compile(fmt.Sprintf("%s.*", r), RoleGlobOptions)
		if err != nil {
			return false, fmt.Errorf("Cannot compile legacy wildcard %s to glob: %v", r, err)
		}

		// Recompile the globs; we now don't want the .** suffix
		globs, err = glob.CompileGlobSet(patterns, RoleGlobOptions)
		if err != nil {
			return false, fmt.Errorf("Cannot compile user roles %s to glob set for legacy matching: %s", globs.String(),
				err.Error())
		}

		// Iterate in reverse, so we can bail early if we hit a match
		g := globs.Globs()
		for i := len(g) - 1; i >= 0; i-- {
			glob := g[i]
			if glob.String() == r || roleGlob.MatchString(glob.String()) {
				match = true
				break
			}
		}
	}

	return match, nil
}
