package hostmapping

import (
	"fmt"
	"strings"
	"sync"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/service/config"
)

var (
	mtx      sync.RWMutex
	mappings map[string]string = make(map[string]string)
)

func init() {
	ch := config.SubscribeChanges()
	im := make(chan bool, 1)
	go func() {
		for {
			// block for trigger
			select {
			case <-ch:
			case <-im:
			}
			// reload
			for {
				if err := loadMappings(); err != nil {
					log.Warnf("Failed to load hostname mappings: %v", err)
				} else {
					break
				}
			}
		}
	}()

	im <- true
}

// loadMappings loads them from config service
func loadMappings() error {
	v := config.AtPath("api", "proxyMappings").AsStringMap()
	mtx.Lock()
	defer mtx.Unlock()
	mappings = v

	log.Infof("Loaded %v hostname mappings from config service: %#v", len(v), mappings)

	return nil
}

// Map maps the proxy address for a hostname, returning the default of "v1-<original-hostname>" if none explicitly defined
func Map(hostname string) string {
	// strip port
	li := strings.LastIndex(hostname, ":")
	if li > 0 {
		hostname = hostname[0:li]
	}
	mtx.RLock()
	defer mtx.RUnlock()
	s := mappings[hostname]
	if s == "" {
		s = fmt.Sprintf("v1-%s", hostname)
	}
	return s
}
