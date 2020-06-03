package sync

import (
	"github.com/hailo-platform/H2O/service/config"
)

// NewGlobalLocker returns a global leader which is basically just a region leader pinned to one region based on
// config.
func NewGlobalLeader(id string) Leader {
	for {
		if config.AtPath("leaders", "isLeader").AsBool() {
			break
		}
		<-config.SubscribeChanges()
	}
	return RegionLeader(id)
}
