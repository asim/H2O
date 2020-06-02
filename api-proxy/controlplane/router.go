package controlplane

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	log "github.com/cihub/seelog"
)

const (
	urlName = "api" // the thin API is referenced by this name (eg: rather than "hms")
)

// A Router routes a request to a backend (H1, H2, or throttle) according to the first matching rule (rules are sorted
// by specificity)
type Router interface {
	Route() *Rule
	GetHobMode() string
	SetHob(string)
	Region() (region *Region, version int64)
	CorrectHostname(rw http.ResponseWriter) (err error, isCorrect bool, urls Urls, version int64)
}

type RuleRouter struct {
	extractor Extractor
	control   *ControlPlane
}

func (r *RuleRouter) SetHob(hob string) {
	r.extractor.SetHob(hob)
}

// Route routes a request to a backend (H1, H2 or throttle) according to the first
// matching rule (rules sorted by specificity)
func (r *RuleRouter) Route() *Rule {
	if routeStr := r.extractor.Header("X-Hailo-Route"); len(routeStr) > 0 {
		if rule := r.forceRoute(routeStr); rule != nil {
			return rule
		}
	}

	for _, rule := range r.control.Rules() {
		if rule.Matches(r.extractor) {
			return rule
		}
	}

	return nil
}

// GetHobMode returns the mode
func (r *RuleRouter) GetHobMode() string {
	if routeStr := r.extractor.Header("X-Hailo-Route"); len(routeStr) > 0 {
		// for forced route we should return what they're forcing to'
		switch routeStr {
		case "H1":
			return "h1"
		case "H2":
			return "h2"
		}
	}
	return r.control.HobModes().Find(r.extractor.Hob())
}

// Region identifies the region, and its config, that we should be sending API
// requests to for a given HTTP request
func (r *RuleRouter) Region() (region *Region, version int64) {
	loadedCfg := r.control.loadedConfig()
	regionId := loadedCfg.hobRegions.Find(r.extractor.Hob())
	region = loadedCfg.regions[regionId]
	version = loadedCfg.rConfigVersion

	// None found? Return the fallback region (the first one as found lexicographically by region ID)
	if region == nil && len(loadedCfg.regions) > 0 {
		keys := make([]string, 0, len(loadedCfg.regions))
		for regionId, _ := range loadedCfg.regions {
			keys = append(keys, regionId)
		}

		sort.Strings(keys)
		region = loadedCfg.regions[keys[0]]
		log.Debugf("Unable to detect region from Hob, picking a default region %s", region)
	}

	if region == nil || region.IsOnline() {
		return
	}

	// Try failovers
	for _, foId := range region.Failover {
		if loadedCfg.regions[foId].IsOnline() {
			region = loadedCfg.regions[foId]
			return
		}
	}

	// Return anyway, even though the region is offline (better than nothing)
	return
}

// CorrectHostname tests if the request hostname matches the expected hostname for the active region, and if not,
// returns the right one
func (r *RuleRouter) CorrectHostname(rw http.ResponseWriter) (err error, isCorrect bool, urls Urls, version int64) {
	hob := r.extractor.Hob()
	if len(hob) == 0 {
		hob = rw.Header().Get("X-H-HOB")
	}
	// If there is no hob, then just accept the URL as correct (otherwise we would have to pick some region - usually
	// just some default - and thus we pin people to the wrong region). The most obvious place this happened was driver
	// login, where we didn't know the city until AFTER they'd logged in, so the login call always pinned everyone to
	// eu-west-1 (sub-optimal)
	if len(hob) == 0 {
		err = fmt.Errorf("[RuleRouter] No HOB available, unable to check region pinning")
		log.Debug(err)
		return
	}

	var region *Region
	region, version = r.Region()
	if region == nil {
		// Nothing we can do :(
		err = fmt.Errorf("[RuleRouter] No Region available, unable to check region pinning")
		log.Debug(err)
		return
	}

	host := r.extractor.Host()
	urls = region.Urls(r.extractor.Source())

	if host == urls[urlName] {
		isCorrect = true
	}

	return
}

func (r *RuleRouter) forceRoute(routeStr string) *Rule {
	switch strings.ToUpper(routeStr) {
	case "H2":
		return &Rule{Action: ActionSendToH2}
	case "H1":
		return &Rule{Action: ActionProxyToH1}
	case "DEPRECATE":
		return &Rule{Action: ActionDeprecate}
	case "THROTTLE":
		return &Rule{Action: ActionThrottle}
	default:
		return nil
	}
}
