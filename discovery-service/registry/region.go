package registry

import (
	"encoding/json"
	log "github.com/cihub/seelog"
	zk "github.com/HailoOSS/service/zookeeper"
	"os"
	"sync"
	"time"
)

const syncInterval = time.Minute * 5

type regionReg struct {
	sync.RWMutex
	instances map[string]*Instance
}

func newRegionReg() *regionReg {
	r := &regionReg{
		instances: make(map[string]*Instance),
	}

	go r.syncer()

	return r
}

// syncer will continually sync with ZK, or quit on failure
func (r *regionReg) syncer() {
	log.Debug("[Discovery] Launching syncer...")
	for {
		instanceIds, _, watch, err := zk.ChildrenW(rootNode)
		if err != nil {
			log.Errorf("[Discovery] Failed to read children, so exiting: %v", err)
			break
		}

		if err := r.sync(instanceIds); err != nil {
			log.Errorf("[Discovery] Failed to sync instances, so exiting: %v", err)
			break
		}

		// @todo not entirely sure what happens when zk conn.Close() happens - hopefully sender closes channel
		e := <-watch
		log.Debugf("[Discovery] Watch triggered for event %v", e)
	}

	log.Infof("[Discovery] Quitting syncer and exiting")
	os.Exit(1)
}

// sync
func (r *regionReg) sync(instanceIds []string) error {
	r.Lock()
	defer r.Unlock()

	seen := map[string]bool{}
	for _, id := range instanceIds {
		// do we know about this? assumption: registry never changes for an instance ID
		if _, ok := r.instances[id]; !ok {
			// look up and add this instance
			b, _, err := zk.Get(zkPathForInstance(id))
			if err != nil {
				return err
			}
			// unmarshal the document
			instance := &Instance{}
			if err := json.Unmarshal(b, instance); err != nil {
				return err
			}
			r.instances[id] = instance
		}
		seen[id] = true
	}

	// remove any not seen
	for id, _ := range r.instances {
		if !seen[id] {
			// strip
			delete(r.instances, id)
		}
	}

	return nil
}

// allInstances returns all registered instances within the region
func (r *regionReg) allInstances() Instances {
	r.RLock()
	defer r.RUnlock()
	ret := make(Instances, len(r.instances))
	i := 0
	for _, inst := range r.instances {
		ret[i] = inst
		i++
	}
	return ret
}

// singleInstance returns one instance, by ID, or nil
func (r *regionReg) singleInstance(instId string) *Instance {
	r.RLock()
	defer r.RUnlock()
	if elem, ok := r.instances[instId]; ok {
		return elem
	}
	return nil
}

func pathToId(path string) string {
	return path
}
