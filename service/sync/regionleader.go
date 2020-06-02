package sync

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff"

	log "github.com/cihub/seelog"

	inst "github.com/HailoOSS/service/instrumentation"

	"github.com/HailoOSS/service/zookeeper"
	gozk "github.com/HailoOSS/go-zookeeper/zk"
)

const (
	regionLeaderPath       = "/sync/regionleader/%v"
	cleanupDelay           = 20 * time.Millisecond
	backoffInitialInterval = 50 * time.Millisecond
	backoffMaxInterval     = 10 * time.Minute
)

var (
	mu  sync.Mutex
	rls []*regionLeader
)

func init() {
	rls = make([]*regionLeader, 0)
}

type regionLeader struct {
	active    bool
	lockNode  string
	rescinded chan struct{}
	cleanup   sync.Once
}

func newRegionLeader(lockNode string) *regionLeader {
	rl := &regionLeader{
		active:    true,
		lockNode:  lockNode,
		rescinded: make(chan struct{}),
	}
	// establish a watch to cleanup
	go func() {
		_, _, watch, err := zookeeper.GetW(rl.lockNode)
		if err != nil {
			rl.Rescind()
			return
		}
		<-watch
		log.Debugf("[Sync:RegionLeader] Watch triggered on '%v', will rescind leadership", rl.lockNode)
		inst.Counter(1.0, "sync.regionleader.remotely-rescinded")
		rl.Rescind()
		return
	}()

	// Register region leader
	mu.Lock()
	rls = append(rls, rl)
	mu.Unlock()

	return rl
}

// Rescinded returns a channel that will close when you are no longer the leader
func (rl *regionLeader) Rescinded() chan struct{} {
	return rl.rescinded
}

// Rescind should be called to indicate you no longer wish to be the leader
func (rl *regionLeader) Rescind() {
	rl.cleanup.Do(func() {
		log.Debugf("[Sync:RegionLeader] Cleaning up leadership of '%v'...", rl.lockNode)
		close(rl.rescinded)
		// keep trying to delete the ZK node (to release leadership) until we're sure it doesn't exist
		for {
			err := zookeeper.Delete(rl.lockNode, -1)
			if err == nil || err == gozk.ErrNoNode {
				log.Debugf("[Sync:RegionLeader] Have deleted leadership node '%v'", rl.lockNode)
				inst.Counter(1.0, "sync.regionleader.rescinded")
				break
			}
			log.Warnf("[Sync:RegionLeader] Failed to cleanup/rescind leadership (will retry): %v", err)
			time.Sleep(cleanupDelay)
		}

		// Unregister region leader
		mu.Lock()
		for i := 0; i < len(rls); i++ {
			if rls[i] == rl {
				rls = append(rls[:i], rls[i+1:]...)
				break
			}
		}
		mu.Unlock()
	})
}

// RegionLeader block indefinitely until this invocation has been elected the "leader" within the local operating region.
// It will then return a channel that will eventually be closed when leadership is rescinded.
func RegionLeader(id string) Leader {
	path := fmt.Sprintf(regionLeaderPath, id)
	prefix := path + "/lock-"
	var lockNode string

	for {
		// create our lock node -- retry until this is done, use exponential backoff
		// to add some delay between attempts
		b := backoff.NewExponentialBackOff()
		b.InitialInterval = backoffInitialInterval
		b.MaxInterval = backoffMaxInterval
		b.MaxElapsedTime = 0 // Never stop retrying

		backoff.RetryNotify(func() (err error) {
			log.Infof("[Sync:RegionLeader] Attepting to create ephemeral lock node for leadership election")
			lockNode, err = zookeeper.CreateProtectedEphemeralSequential(prefix, []byte{}, gozk.WorldACL(gozk.PermAll))

			return
		}, b, func(err error, d time.Duration) {
			if err == gozk.ErrNoNode {
				createParents(path)
			} else if err != nil {
				log.Warnf("[Sync:RegionLeader] ZooKeeper error creating ephemeral lock node for leadership election: %s. Waiting %s", err, d)
			}
		})

		err := waitForWinner(path, lockNode)
		if err != nil {
			// try to cleanup - then go again
			zookeeper.Delete(lockNode, -1)
			time.Sleep(time.Second)
			continue
		}

		// we are the leader
		break
	}

	log.Infof("[Sync:RegionLeader] Elected leader of '%v'", id)
	inst.Counter(1.0, "sync.regionleader.elected")

	return newRegionLeader(lockNode)
}

// CleanupRegionLeaders is a cleanup callback function which is run when the
// service is interrupted and rescinds any outstanding region leaders.
func CleanupRegionLeaders() {
	// Copy region leaders
	mu.Lock()
	rlsCopy := make([]*regionLeader, len(rls))
	for i, rl := range rls {
		rlsCopy[i] = rl
	}
	mu.Unlock()

	for _, rl := range rlsCopy {
		rl.Rescind()
	}
}

func waitForWinner(path, ourNode string) error {
	seq, err := parseSeq(ourNode)
	if err != nil {
		return err
	}

	for {
		lowestSeq, prevSeqPath, err := findLowestSequenceNode(path, seq)
		if err != nil {
			return err
		}

		if seq == lowestSeq {
			// we are now the leader!
			break
		}

		// wait on the node next in line for the lock
		_, _, ch, err := zookeeper.GetW(path + "/" + prevSeqPath)
		if err != nil && err != gozk.ErrNoNode {
			return err
		} else if err != nil && err == gozk.ErrNoNode {
			// try again
			continue
		}

		ev := <-ch
		if ev.Err != nil {
			return ev.Err
		}
	}

	return nil
}

func createParents(path string) error {
	parts := strings.Split(path, "/")
	pth := ""
	for _, p := range parts[1:] {
		pth += "/" + p
		_, err := zookeeper.Create(pth, []byte{}, 0, gozk.WorldACL(gozk.PermAll))
		if err != nil && err != gozk.ErrNodeExists {
			return err
		}
	}
	return nil
}

func parseSeq(path string) (int, error) {
	parts := strings.Split(path, "-")
	return strconv.Atoi(parts[len(parts)-1])
}

// findLowestSequenceNode within a particular lock path
func findLowestSequenceNode(path string, seq int) (lowestSeq int, prevSeqPath string, err error) {
	// Read all the children of the node
	// This is why we create sequential nodes under a parent node based on the lock ID
	// If not, say we stored all ephemeral locks under a higher level parent,
	// we would be checking nodes of every lock currently in play, rather than locks
	// on this ID
	children, _, err := zookeeper.Children(path)
	if err != nil {
		return -1, "", err
	}

	var ttl time.Time
	lowestSeq = seq
	prevSeq := 0
	prevSeqPath = ""
	for _, p := range children {
		// Check if this lock has timed out
		data, _, _ := zookeeper.Get(path + "/" + p)
		if len(data) > 0 {
			ttl.GobDecode(data)
			if ttl.Before(time.Now()) {
				log.Tracef("[RegionLock] Deleting expired lock '%s'", path+"/"+p)
				zookeeper.Delete(path+"/"+p, -1)
				continue
			}
		}

		s, err := parseSeq(p)
		if err != nil {
			return -1, "", err
		}
		if s < lowestSeq {
			lowestSeq = s
		}
		if s < seq && s > prevSeq {
			prevSeq = s
			prevSeqPath = p
		}
	}

	return lowestSeq, prevSeqPath, err
}
