package controlplane

import (
	"fmt"
)

// Validate checks that the regions list is valid
func (rs Regions) Validate() error {
	if rs == nil || len(rs) == 0 {
		return fmt.Errorf("Must have at least one region defined")
	}
	return nil
}

// Find locates the region ID for a HOB, falling back to "default" if none found and returning "" if that isn't present
func (hr HobRegions) Find(hob string) string {
	if r, ok := hr[hob]; ok {
		return r
	}
	if r, ok := hr["default"]; ok {
		return r
	}
	return ""
}

// Find locates the mode for a HOB, falling back to "default" if not found and returning "" if that isn't present
func (hm HobModes) Find(hob string) string {
	if m, ok := hm[hob]; ok {
		return m
	}

	if m, ok := hm["default"]; ok {
		return m
	}

	return ""
}

// IsOnline tells us if this region is configured to accept traffic
func (r *Region) IsOnline() bool {
	return r != nil && r.Status != "OFFLINE" // only this specific word takes us offline
}

// Urls returns URLs for a given app within a region
func (r *Region) Urls(app string) Urls {
	if u, ok := r.Apps[app]; ok {
		return u
	}
	if u, ok := r.Apps["default"]; ok {
		return u
	}
	return nil
}
