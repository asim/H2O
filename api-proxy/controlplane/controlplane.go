package controlplane

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	log "github.com/cihub/seelog"
	"github.com/davegardnerisme/deephash"
	"gopkg.in/tomb.v2"

	"github.com/HailoOSS/service/config"
)

const (
	filename        = "/opt/hailo/var/cache/api-proxy-config"
	reloadFailDelay = time.Second
)

// LoadLastGoodConfig loads the last known "good" config from file
func LoadLastGoodConfig() {
	// try to load ONCE and never reload
	f, err := os.Open(filename)
	if err != nil {
		log.Errorf("[Control Plane] Last known config load failed: %v", err)
	}
	defer f.Close()

	err = config.Load(f)
	if err != nil {
		log.Errorf("[Control Plane] Last known config load failed: %v", err)
	}
}

// As we want to treat updates to loaded config as an atomic whole, we wrap them in this struct
type loadedCpConfig struct {
	rules          SortedRules
	regions        Regions
	hobRegions     HobRegions
	hobModes       HobModes
	rConfigVersion int64  // region config version - a timestamp
	configHash     string // hash of ALL config last loaded so we avoid reloading unless changed
}

// ControlPlane represents our config-based system for controlling Hailo traffic
// both in terms of routing of requests to regions (region "pinning") and also
// routing of traffic to H1 vs H2 backends
type ControlPlane struct {
	// @TODO: Use atomic.Value in Go >=1.4
	tomb.Tomb
	_loadedConfig unsafe.Pointer // *loadedCpConfig -- dereferenced automatically by loadedConfig()
	loadCycleLock sync.Mutex     // used to only allow one config reload loop at a time
}

// New initialises a control plane that loads via the config service
func New() *ControlPlane {
	cp := &ControlPlane{}
	ch := config.SubscribeChanges()
	cp.Tomb.Go(func() error {
		for {
			select {
			case <-cp.Dying():
				log.Debug("[Control Plane] Dying in response to tomb death")
				return nil
			case <-ch:
				log.Debug("[Control Plane] Got notification of config change; reloading rules...")
				if err := cp.loadCycle(); err != nil {
					log.Debugf("[Control Plane] Failed to reload rules on config change: %s", err.Error())
				}
			}
		}
	})

	if err := cp.tryLoad(); err != nil { // Load immediately too
		log.Errorf("[Control Plane] Failed to load config on first attempt (synchronously): %v", err)
	}

	return cp
}

func (cp *ControlPlane) loadedConfig() loadedCpConfig {
	if cfg := atomic.LoadPointer(&cp._loadedConfig); cfg != nil {
		return *((*loadedCpConfig)(cfg))
	} else {
		return loadedCpConfig{}
	}
}

// Router takes a request and prepares us for routing it to a backend and/or region
func (cp *ControlPlane) Router(req *http.Request) Router {
	return &RuleRouter{
		extractor: newExtractor(req),
		control:   cp,
	}
}

// Regions obtains the current region config from the control plane
func (cp *ControlPlane) Regions() Regions {
	if cp == nil {
		return nil
	}

	result := make(Regions)
	for k, v := range cp.loadedConfig().regions {
		if v != nil {
			result[k] = v
		}
	}
	return result
}

// HobRegions obtains the current hob to region mappings from the control plane
func (cp *ControlPlane) HobRegions() HobRegions {
	if cp == nil {
		return nil
	}
	return cp.loadedConfig().hobRegions
}

func (cp *ControlPlane) Rules() SortedRules {
	return cp.loadedConfig().rules
}

func (cp *ControlPlane) HobModes() HobModes {
	return cp.loadedConfig().hobModes
}

// loadCycle blocks on loading until successfully completed. Once completed it writes out the "last good" config to
// disk. There can only be one load cycle at a time.
func (cp *ControlPlane) loadCycle() error {
	cp.loadCycleLock.Lock()
	defer cp.loadCycleLock.Unlock()

cycleLoop:
	for {
		select {
		case <-cp.Dying():
			break cycleLoop
		default:
		}

		if err := cp.tryLoad(); err != nil {
			log.Warnf("[Control Plane] Failed to load config: %v", err)
			time.Sleep(reloadFailDelay)
		} else {
			break
		}
	}

	return nil
}

type parsedConfig struct {
	Cp parsedControlPlane `json:"controlPlane"`
}

type parsedControlPlane struct {
	Rules         Rules      `json:"rules,omitempty"`
	Regions       Regions    `json:"regions,omitempty"`
	HobRegions    HobRegions `json:"hobRegions,omitempty"`
	ConfigVersion float64    `json:"configVersion"`
	HobModes      HobModes   `json:"hobModes,omitempty"`
}

// tryLoad parses config from config service and checks validity, returning an error
// if any config is invalid -- with fairly strict rules/expectations
func (cp *ControlPlane) tryLoad() error {
	log.Tracef("[Control Plane] Trying to load config")
	// for our last-known "good copy" -- grab ALL config at once, then chop it up
	rawConfig := config.Raw()

	if len(rawConfig) == 0 {
		return fmt.Errorf("error loading config -- zero length")
	}

	parsed := parsedConfig{}
	if err := json.Unmarshal(rawConfig, &parsed); err != nil {
		return fmt.Errorf("JSON unmarshal error: %v", err)
	}

	sorted := parsed.Cp.Rules.Sort()
	regions, hobRegions, hobModes := parsed.Cp.Regions, parsed.Cp.HobRegions, parsed.Cp.HobModes
	configVersion := int64(parsed.Cp.ConfigVersion)

	// sanity check
	if err := sorted.Validate(); err != nil {
		return err
	}
	if err := regions.Validate(); err != nil {
		return err
	}

	// see if anything has changed
	h := deephash.Hash([]interface{}{
		sorted,
		regions,
		hobRegions,
		configVersion,
		hobModes,
	})

	newHash := fmt.Sprintf("%x", h)
	currHash := cp.currentHash()

	if newHash == currHash {
		return nil
	}

	// update our control plane config now
	tmp := &ControlPlane{}
	atomic.StorePointer(&tmp._loadedConfig, unsafe.Pointer(&loadedCpConfig{

		rules:          sorted,
		regions:        regions,
		hobRegions:     hobRegions,
		rConfigVersion: configVersion,
		hobModes:       hobModes,
		configHash:     newHash,
	}))
	err := tmp.saveConfigToFile(rawConfig)
	if err != nil {
		log.Errorf("[Control Plane] Failed to write last good config: %v", err)
	}

	cfg := tmp.loadedConfig()
	atomic.StorePointer(&cp._loadedConfig, unsafe.Pointer(&cfg))

	log.Infof("[Control Plane] Loaded - %d rules, %d regions, %d HOB regions, %d HOB modes - regionTS=%d", len(sorted),
		len(regions), len(hobRegions), len(hobModes), configVersion)

	return nil
}

// currentHash just returns the current config hash
func (cp *ControlPlane) currentHash() string {
	return cp.loadedConfig().configHash
}

// saveConfigToFile will save the current config to file -- this is called after every
// successful load/validate cycle, so we know at this point the config is valid
func (cp *ControlPlane) saveConfigToFile(c []byte) error {
	dirPath := filepath.Dir(filename)

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}

	// Minimise the likelihood of borking the existing config by doing a write to a temp file and then a move
	tmpFile, err := ioutil.TempFile(dirPath, filepath.Base(filename))
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(tmpFile.Name(), c, 0644); err != nil {
		return fmt.Errorf("Error writing config file: %v", err)
	}

	// Shuffle the temp file to the new file
	if err := os.Rename(tmpFile.Name(), filename); err != nil {
		return err
	}

	log.Infof("[Control Plane] Last good config saved to '%v'", filename)
	return nil
}
