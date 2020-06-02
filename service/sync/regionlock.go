package sync

import (
	"errors"
	"fmt"
	"strings"
	sy "sync"
	"time"

	log "github.com/cihub/seelog"

	inst "github.com/HailoOSS/service/instrumentation"
	zk "github.com/HailoOSS/service/zookeeper"
	gozk "github.com/HailoOSS/go-zookeeper/zk"
)

var (
	minRegionHoldFor     time.Duration = time.Second
	defaultRegionWaitFor time.Duration = time.Second
	defaultRegionHoldFor time.Duration = time.Second * 2
	defaultReapTime      time.Duration = time.Second * 10 // TODO sanity check

	// @todo cruft - move this into the lock registry, or something with a mutex
	regionLockNamespace = ""
)

var (
	ErrRegionHoldFor error = errors.New(fmt.Sprintf("Error locking - holdFor duration must be %v or greater", minRegionHoldFor))
	defaultReaper    *reaper
	once             sy.Once
)

const (
	reaperThreshold = 2 // max number of reapings where node doesn't exist before we remove from the list
)

type regionLock struct {
	zkLock gozk.Locker
}

// reaper periodically sweeps ZK and deletes nodes. Based on Netflix Curator Reaper
type reaper struct {
	paths    map[string]int // the paths to reap to count of how many times in a row they've been seen with no children
	pathsMtx sy.RWMutex
}

func startReaper() {
	log.Infof("[Sync:RegionLock] Initialising RegionLock reaper")
	defaultReaper = &reaper{
		paths: make(map[string]int),
	}
	go defaultReaper.reapLoop()
}

// Utility methods

// constructLockPath with a namespace and an ID to lock on
func constructLockPath(namespace, id string) (string, error) {
	// don't use '/' character in lock id's!!
	// replace with hashes
	id = strings.Replace(id, "/", "#", -1)

	path := fmt.Sprintf("/%s/%s", namespace, id)

	if namespace == "" || id == "" {
		return path, fmt.Errorf("Namespace or ID cannot be blank")
	}

	if strings.Contains(path, "//") {
		return path, fmt.Errorf("Generated lock path contains double slash")
	}

	// @todo also check for other invalid characters:
	// https://zookeeper.apache.org/doc/r3.1.2/zookeeperProgrammers.html#ch_zkDataModel

	return path, nil
}

// SetRegionLockNamespace should be set to the service name on startup, and never again!
func SetRegionLockNamespace(ns string) {
	regionLockNamespace = ns
}

// RegionLock attempts to achieve a lock using default timing values
func RegionLock(id []byte) (Lock, error) {
	return RegionTimedLock(id, defaultRegionWaitFor, defaultRegionHoldFor)
}

// RegionTimedLock attempts to achieve a regional lock on `id`, waiting for `waitFor` time in
// case of contention (before giving up) and reserving the lock for a maximum `holdFor`
// in the event of failing to Unlock()
func RegionTimedLock(id []byte, waitFor, holdFor time.Duration) (Lock, error) {
	// Ensure we are reaping
	once.Do(startReaper)

	if int64(holdFor) < int64(minRegionHoldFor) {
		return nil, ErrRegionHoldFor
	}

	lockId := string(id) // we use []byte for function signature compatibility with the global lock

	// Our locks are namespaced per service
	path, err := constructLockPath(regionLockNamespace, lockId)
	if err != nil {
		return nil, err
	}
	log.Tracef("[Sync:RegionTimedLock] Attempting to acquire '%s'; held for %v", path, holdFor)

	// Create new lock which we will be lock()ed
	lock := &regionLock{
		zkLock: zk.NewLock(path, gozk.WorldACL(gozk.PermAll)),
	}
	lock.zkLock.SetTTL(holdFor)
	lock.zkLock.SetTimeout(waitFor)
	// Acquire a lock
	startTime := time.Now()
	err = lock.zkLock.Lock()
	inst.Timing(1.0, "sync.regionlock.acquire", time.Since(startTime))
	defaultReaper.addPath(path) // only add path to reaper AFTER we've acquired the lock (or not)

	if err == nil {
		log.Tracef("[Sync:RegionTimedLock] Successfully acquired '%s'", path)
		inst.Counter(1.0, "sync.regionlock.acquire.success")
	} else {
		log.Errorf("[Sync:RegionTimedLock] Failed to acquire '%s': %s", path, err.Error())
		inst.Counter(1.0, "sync.regionlock.acquire.failure")
	}

	return lock, err
}

// Unlock releases this regional lock
func (rl *regionLock) Unlock() {
	if rl == nil || rl.zkLock == nil {
		return
	}

	// This should check for a ZK connection, and also report errors to the caller. But as Matt Heath designed this to
	// match the interface of global locks, which cannot return an error, we don't return any error here.
	if err := rl.zkLock.Unlock(); err != nil {
		log.Errorf("[Sync:RegionLock] Failed to release ZooKeeper lock with: %s", err.Error())
	}
}

func (r *reaper) reap() {
	r.pathsMtx.RLock()
	// snapshot paths
	keys := make([]string, len(r.paths))
	i := 0
	for k := range r.paths {
		keys[i] = k
		i++
	}
	r.pathsMtx.RUnlock()
	for _, path := range keys {
		exists, stat, err := zk.Exists(path)
		if !exists {
			if err != nil && err != gozk.ErrNoNode {
				log.Errorf("[Sync:RegionLock] Error checking path %s %v", path, err)
				r.resetPath(path)
				// something is still using this
				continue
			}
			// no node
		} else {
			if stat.NumChildren > 0 {
				r.resetPath(path)
				continue
			}
			// node but no children
		}
		// increment reap number
		n := r.incrementPath(path)
		if n >= reaperThreshold {
			// reaped enough times and it's come out as empty. Delete it
			if err := zk.Delete(path, -1); err == nil || err == gozk.ErrNoNode {
				// success
				r.removePath(path)
			} else {
				log.Warnf("[Sync:RegionLock] Error reaping path %s. %v", path, err) // debug not error on purpose
				// some error, most likely node being used by something else, reset and let it live
				r.resetPath(path)
			}
		}
	}
}

// incrementPath increments the count for this path and returns the new value for how many times the path has been seen empty
func (r *reaper) incrementPath(path string) int {
	r.pathsMtx.Lock()
	defer r.pathsMtx.Unlock()
	n, ok := r.paths[path]
	if ok {
		r.paths[path] = n + 1
	} else {
		// shouldn't happen
		log.Warnf("[Sync:RegionLock] Tried to increment reap count on path which doesn't exist %s", path)
	}
	return r.paths[path]

}

func (r *reaper) reapLoop() {
	tick := time.NewTicker(defaultReapTime)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			log.Debugf("[Sync:RegionLock] Reaping region locks. Currently %d paths", len(r.paths))
			now := time.Now()
			r.reap()
			log.Debugf("[Sync:RegionLock] Finished reaping region locks in %dms. Now %d paths", time.Since(now)/time.Millisecond, len(r.paths))
		}
	}
}

// addPath adds this path with value of 0 for how many times path has been seen empty.
func (r *reaper) addPath(path string) {
	r.resetPath(path)
}

// resetPath resets the reap count for this to 0
func (r *reaper) resetPath(path string) {
	r.pathsMtx.Lock()
	r.paths[path] = 0
	r.pathsMtx.Unlock()
}

// removePath removes this path from the reap list
func (r *reaper) removePath(path string) {
	r.pathsMtx.Lock()
	// last check in case we've added the path back while we were reaping it
	if cnt, ok := r.paths[path]; ok && cnt >= reaperThreshold {
		delete(r.paths, path)
	}
	r.pathsMtx.Unlock()
}
