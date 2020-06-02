package controlplane

// Extractor allows us to extract stuff about requests when matching
type Extractor interface {
	Hob() string               // Hob is the city code
	Value(name string) string  // Value is some POST or GET value
	Path() string              // Path is the pathname of the request
	SetHob(string)             // Set the hob on the request parameters
	Source() string            // Source is whether this came from "customer" or "driver" API - expecting return one of these two
	Host() string              // Host of request
	Header(name string) string // Header is some HTTP header
}

// Rules represents a set of unsorted rules, indexed by UID
type Rules map[string]*Rule

// SortedRules represents a set of rules, sorted by specificity
type SortedRules []*Rule

// Rule represents some matching criteria plus an action
type Rule struct {
	Match   *Match   `json:"match,omitempty"`
	Action  Action   `json:"action,omitempty"`
	Payload *Payload `json:"payload,omitempty"`
	Weight  int      `json:"weight,omitempty"`
}

// Match represents some criteria to match an HTTP request against
type Match struct {
	Hob        string  `json:"regulatoryArea,omitempty"` // Hob is our city code, worked out either from hostname or explicit city=FOO parameter (POST or GET)
	Path       string  `json:"path,omitempty"`           // Path is the pathname, like /v1/foo/bar
	Source     string  `json:"source,omitempty"`         // Source is either "customer" or "driver" or "", where we work this out from the hostname (contains customer or driver or not)
	Proportion float32 `json:"proportion,omitempty"`     // Proportion is a float from 0 to 1 that gives us sampling
	Sampler    Sampler `json:"sampler,omitempty"`        // Sampler tells us how to sample
}

// Sampler defines how we should sample requests for matching
type Sampler int

const (
	// RandomSampler samples completely randomly
	RandomSampler Sampler = 0
	// CustomerSampler samples by "customer" POST or GET parameter
	CustomerSampler Sampler = 1
	// DriverSampler samples by "driver" POST or GET parameter
	DriverSampler Sampler = 2
	// DeviceSampler samples by "device" POST or GET parameter
	DeviceSampler Sampler = 3
	// SessionSampler samples by "api_token" or "session_id" POST or GET parameter
	SessionSampler Sampler = 4
)

// Action represents some outcome that is attached to a route, telling us what we should do next
type Action int

const (
	ActionProxyToH1 Action = 1
	ActionSendToH2  Action = 2
	ActionThrottle  Action = 3
	ActionDeprecate Action = 4
)

func (a Action) String() string {
	// We don't want the Action… prefix, so we don't use stringer for this
	switch a {
	case ActionProxyToH1:
		return "H1"
	case ActionSendToH2:
		return "H2"
	case ActionThrottle:
		return "Throttle"
	case ActionDeprecate:
		return "Deprecate"
	default:
		return "?"
	}
}

// Payload represents the information we want to use when serving throttled calls
type Payload struct {
	Body       string            `json:"body,omitempty"`
	HttpStatus int               `json:"httpStatus,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// Regions represents a map of all region definitions for app pinning purposes, indexed by ID
type Regions map[string]*Region

// Urls defines URLs for named purposes, eg: "hms", "api"
type Urls map[string]string

// Region defines config for a single AWS region
type Region struct {
	Id       string          `json:"id,omitempty"`       // ID of this region, eg: us-east-1
	Status   string          `json:"status,omitempty"`   // Status is ONLINE or OFFLINE - not a bool to cope with possible future additions
	Failover []string        `json:"failover,omitempty"` // Failover regions (if this one down)
	Apps     map[string]Urls `json:"apps,omitempty"`     // Apps and their URL config for pinning
}

// HobRegions maps HOBs to primary regions
type HobRegions map[string]string

// HobModes maps HOBs to modes
type HobModes map[string]string

//go:generate stringer -type=Sampler -output=types_string.go
