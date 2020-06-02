package controlplane

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"math/rand"
	"sort"
	"strings"
)

// Sort rules by specificity
func (rs Rules) Sort() SortedRules {
	if rs == nil {
		return SortedRules{}
	}
	ret := make(SortedRules, len(rs))
	i := 0
	for _, r := range rs {
		ret[i] = r
		i++
	}
	sort.Sort(ret)
	return ret
}

// Add a rule to a map of rules
func (rs Rules) Add(r *Rule) Rules {
	rs[r.Id()] = r
	return rs
}

// Len is part of sort.Interface
func (s SortedRules) Len() int {
	return len(s)
}

// Swap is part of sort.Interface
func (s SortedRules) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less is part of sort.Interface
func (s SortedRules) Less(i, j int) bool {
	if s[i].Weight != s[j].Weight {
		return s[i].Weight > s[j].Weight
	}
	return s[i].Specificity() > s[j].Specificity()
}

// Validate tests if the sorted rules are considered a valid set
func (s SortedRules) Validate() error {
	if s == nil || len(s) == 0 {
		return fmt.Errorf("Must have at least one routing rule defined")
	}
	return nil
}

// Matches tests if a route matches a request, wrapped with an extractor
func (r *Rule) Matches(ext Extractor) bool {
	return r != nil && r.Match != nil && r.Match.matches(ext)
}

// Specificity returns a number that tells us how specific this rule is - in
// a similar vein to CSS. We will then check the most specific first.
func (r *Rule) Specificity() int {
	s := 0

	if r != nil && r.Match != nil {
		if len(r.Match.Hob) > 0 {
			s += 5
		}
		if len(r.Match.Source) > 0 {
			s += 5
		}
		if len(r.Match.Path) > 0 {
			s += 10
		}
	}

	return s
}

// Id generats a deterministic unique ID for a rule
func (r *Rule) Id() string {
	b, _ := json.Marshal(r)
	h := fnv.New32a()
	h.Write(b)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// matches tests if a match matches an HTTP request
func (m *Match) matches(ext Extractor) bool {
	// check path first (easiest)
	if len(m.Path) > 0 && !strings.HasPrefix(ext.Path(), m.Path) {
		return false
	}

	// check source, since we don't have to dig into the request
	if len(m.Source) > 0 && ext.Source() != m.Source {
		return false
	}

	// check regulatory area
	if len(m.Hob) > 0 && !withinCsv(m.Hob, ext.Hob()) {
		return false
	}

	// apply sampling
	if !m.sample(ext) {
		return false
	}

	return true
}

// sample calculates if we should allow this request based on sampling
// returns true if this request matches based on sample
func (m *Match) sample(ext Extractor) bool {
	switch m.Sampler {
	case CustomerSampler:
		return hashSample(ext.Value("customer"), m.Proportion)
	case DriverSampler:
		return hashSample(ext.Value("driver"), m.Proportion)
	case DeviceSampler:
		return hashSample(ext.Value("device"), m.Proportion)
	case SessionSampler:
		return hashSample(ext.Value("session_id"), m.Proportion)
	}

	// default is random
	roll := rand.Float32()
	if roll > m.Proportion {
		return false
	}
	return true
}

// hashSample hashes v into a number, then calculates a remainder and thus decides if we should sample or not
func hashSample(v string, prop float32) bool {
	// if blank, then we ALWAYS sample, on the basis that we can't fairly calculate random
	// chance if we don't have any value to go on, thus we err on the side of matching
	// we err on matching, since this kind of sampling is most useful for throttling requests
	// and we want the default behaviour to be isThrottled: true
	if v == "" {
		return true
	}
	// if 0 then we know we can't match and we can skip hashing
	if prop <= 0.0 {
		return false
	}
	h := fnv.New64()
	io.WriteString(h, v)
	i := h.Sum64()
	// split into 1000000 buckets
	bucket := i % 1000000
	if bucket > uint64(prop*1000000) {
		return false
	}
	return true
}

// withinCsv tests to see if some value is within a CSV of possible values
func withinCsv(csv, test string) bool {
	values := strings.Split(csv, ",")
	for _, v := range values {
		if v == test {
			return true
		}
	}
	return false
}
