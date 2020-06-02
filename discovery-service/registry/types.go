package registry

import (
	"fmt"
	"strings"
)

// Sla defines how we expect an instance to perform in terms of response times, resource usage etc.
type Sla struct {
	// Mean is the mean avg response time (time to generate response) promised for this endpoint
	Mean uint32
	// Upper95 is 95th percentile response promised for this endpoint
	Upper95 uint32
}

// Endpoint is a single H2 endpoint which can either be called via REQ/REP or will SUB to messages (for PUB/SUB)
type Endpoint struct {
	// Name of this endpoint
	Name string
	// Subscribe is the topic to SUB to if this endpoint is a subscription endpoint
	Subscribe string
	// Sla that this endpoint promises to keep
	Sla Sla
}

// Instance is a single running version of a service on a single host
type Instance struct {
	Id           string
	Hostname     string
	MachineClass string
	Name         string
	Description  string
	AzName       string
	Source       string
	OwnerEmail   string
	OwnerMobile  string
	OwnerTeam    string
	Version      uint64
	Endpoints    []*Endpoint
}

// GetSubTopics returns a list of the Subscribe topics for each Endpoint this
// instance contains
func (inst Instance) GetSubTopics() []string {
	topics := make([]string, len(inst.Endpoints))
	for i, ep := range inst.Endpoints {
		topics[i] = ep.Subscribe
	}

	return topics
}

// Instances represents a list of instances
type Instances []*Instance

// Filter defines a single way of filtering out instances, where returning true indicates something SHOULD be removed (filtered out)
type Filter func(inst *Instance) bool

// ---

// Filter returns a list of filtered instances
func (list Instances) Filter(f Filter) Instances {
	ret := make(Instances, 0)
	for _, i := range list {
		if !f(i) {
			ret = append(ret, i)
		}
	}
	return ret
}

// MatchingServicePrefix filter by service name
func MatchingServicePrefix(p string) Filter {
	return func(inst *Instance) bool {
		return !strings.HasPrefix(inst.Name, p)
	}
}

// MatchingService filter by service name (exact match)
func MatchingService(name string) Filter {
	return func(inst *Instance) bool {
		return inst.Name != name
	}
}

// MatchingAz filter by availability zone
func MatchingAz(az string) Filter {
	return func(inst *Instance) bool {
		return inst.AzName != az
	}
}

// ---

// zkPath yields the ZK path
func (i *Instance) zkPath() string {
	return zkPathForInstance(i.Id)
}

// zkPathForInstance gets path for an id
func zkPathForInstance(instanceId string) string {
	return fmt.Sprintf(instanceNode, instanceId)
}
