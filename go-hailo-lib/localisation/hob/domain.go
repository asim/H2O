package hob

import (
	"github.com/HailoOSS/platform/server"

	"fmt"
	"sort"
	"time"
)

const (
	forceReloadPeriod      = 1 * time.Hour
	forceReloadJitter      = time.Minute
	triggerReloadJitter    = time.Second * 10
	failedMinLoadRetry     = time.Minute * 1
	failedMaxLoadRetry     = time.Minute * 30
	failedLoadRetryFactor  = 2
	reloadMaxRetryAttempts = 4
)

var (
	Cache       HobsCache = MemoryHobsCache()
	serverScope           = server.Scoper()
)

// Deprecated, use GetHob(hob) instead.
func ReadHob(scope *server.Request, hob string) (*Hob, error) {
	return GetHob(hob)
}

// Hob return the Hob object
func GetHob(hob string) (*Hob, error) {
	cachedHob := Cache.ReadHob(hob)
	if cachedHob != nil {
		return cachedHob, nil
	}

	return h2HobService.ReadHob(hob)
}

// Deprecated, use GetServiceType(hob, serviceType) instead
func ReadServiceType(scope *server.Request, hob string, serviceType string) (*ServiceType, error) {
	return GetServiceType(hob, serviceType)
}

// ServiceType returns the service type for the given parameters
func GetServiceType(hob string, serviceType string) (*ServiceType, error) {
	serviceTypes, err := GetServiceTypes(hob)
	if err != nil {
		return nil, err
	}
	result := serviceTypes[serviceType]
	if result == nil {
		return nil, fmt.Errorf(`Could not find service type "%s" in hob "%s"`, serviceType, hob)
	}
	return result, nil
}

// Deprecated, use ServiceTypes instead
func ReadServiceTypes(scope *server.Request, hob string) (ServiceTypes, error) {
	return GetServiceTypes(hob)
}

// ServiceTypes returns all the service types for the given hob
func GetServiceTypes(hob string) (ServiceTypes, error) {
	cachedServiceTypes := Cache.ReadServiceTypes(hob)
	if cachedServiceTypes != nil {
		return cachedServiceTypes, nil
	}

	return h2HobService.ReadServiceTypes(hob)
}

// Depricated, use GetTieredServiceTypeList instead
func ReadTieredServiceTypeList(scope *server.Request, hob string) ([]*ServiceType, error) {
	return GetTieredServiceTypeList(hob)
}

func GetTieredServiceTypeList(hob string) ([]*ServiceType, error) {
	sts, err := GetServiceTypes(hob)
	if err != nil {
		return nil, err
	}

	// Let's sort the service types by tier
	tiers := make([]*ServiceType, len(sts))
	i := 0
	for _, st := range sts {
		tiers[i] = st
		i++
	}

	sort.Sort(TieredServiceTypes(tiers))

	return tiers, nil
}

type TieredServiceTypes []*ServiceType

func (t TieredServiceTypes) Len() int {
	return len(t)
}

func (t TieredServiceTypes) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t TieredServiceTypes) Less(i, j int) bool {
	return t[i].Tier < t[j].Tier
}
