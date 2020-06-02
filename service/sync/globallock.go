package sync

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/go-hailo-lib/multierror"
	"github.com/HailoOSS/service/cassandra"
	inst "github.com/HailoOSS/service/instrumentation"
	"github.com/HailoOSS/gossie/src/gossie"
)

const (
	keyspace     = "sync"
	cfGlobalLock = "globalLock"
)

var (
	minHoldFor     time.Duration = time.Second * 2
	delayFor       time.Duration = time.Millisecond * 20
	defaultWaitFor time.Duration = time.Second
	defaultHoldFor time.Duration = time.Second * 5
)

var (
	ErrContended error = errors.New("Could not obtain lock due to contention")
	ErrGenUuid   error = errors.New("Error locking due to UUID generation failure")
	ErrHoldFor   error = errors.New("Error locking - holdFor duration must be 2 seconds or greater")
)

type globalLock struct {
	id       []byte
	lockId   gossie.UUID
	exit     chan struct{}
	unlocked bool
}

// Unlock releases this global lock
func (gl *globalLock) Unlock() {
	if gl.unlocked {
		return
	}
	// close the exit channel (so we can only Unlock once) which causes our refresher loop (if any) to break
	close(gl.exit)
	gl.unlocked = true

	// delete from C*
	pool, err := cassandra.ConnectionPool(keyspace)
	if err != nil {
		log.Warnf("[Sync:GlobalLock] Release error due to C*: %v", err)
		return
	}
	writer := pool.Writer()
	writer.ConsistencyLevel(gossie.CONSISTENCY_QUORUM).DeleteColumns(
		cfGlobalLock,
		gl.id,
		[][]byte{[]byte(gl.lockId[:])},
	)
	if err := writer.Run(); err != nil {
		log.Warnf("[Sync:GlobalLock] Unlock failed due to C*: %v", err)
	}
}

// GlobalLock attempts to achieve a lock using default timing values of 1 second and 5 seconds
func GlobalLock(id []byte) (Lock, error) {
	return GlobalTimedLock(id, defaultWaitFor, defaultHoldFor)
}

// GlobalTimedLock attempts to achieve a global lock on `id`, waiting for `waitFor` time in
// case of contention (before giving up) and reserving the lock for a maximum `holdFor`
// in the event of failure
// NOTE: locks can and will be held for longer than `holdFor`, but in the case of failure
// (eg: binary crashes) then this is the maximum amount of time other programs will hang
// around contending for the now defunkt lock
func GlobalTimedLock(id []byte, waitFor, holdFor time.Duration) (Lock, error) {
	if int64(holdFor) < int64(minHoldFor) {
		return nil, ErrHoldFor
	}

	u, err := gossie.NewTimeUUID()
	if err != nil {
		log.Warnf("[Sync:GlobalLock] Failed to generate time UUID: %v", err)
		return nil, ErrGenUuid
	}
	l := &globalLock{
		id:     id,
		lockId: u,
		exit:   make(chan struct{}),
	}

	// make my node in C*
	pool, err := cassandra.ConnectionPool(keyspace)
	if err != nil {
		return nil, fmt.Errorf("Error locking due to C*: %v", err)
	}
	writer := pool.Writer()
	writer.ConsistencyLevel(gossie.CONSISTENCY_QUORUM).Insert(cfGlobalLock, &gossie.Row{
		Key: l.id,
		Columns: []*gossie.Column{
			{
				Name:  []byte(l.lockId[:]),
				Value: []byte{}, // @todo could inject some data about who has the lock here
				Ttl:   durationToSeconds(holdFor, 1.0),
			},
		},
	})
	startTime := time.Now()
	err = writer.Run()
	if err != nil {
		inst.Timing(1.0, "sync.globaltimedlock.acquire", time.Since(startTime))
		inst.Counter(1.0, "sync.globaltimedlock.acquire.failure")
		return nil, err
	}

	// read all back and ensure i'm the lowest
	reader := pool.Reader().ConsistencyLevel(gossie.CONSISTENCY_QUORUM).Cf(cfGlobalLock)
	attempts := 0
	errs := multierror.New()
	start := time.Now()
	for {
		// break out if we've waited too long
		if attempts > 0 {
			if time.Now().After(start.Add(waitFor)) {
				inst.Timing(1.0, "sync.globaltimedlock.acquire", time.Since(startTime))
				inst.Counter(1.0, "sync.globaltimedlock.acquire.failure")
				l.Unlock()
				return nil, ErrContended
			}
			// delay a bit to avoid hammering C*
			time.Sleep(addJitter(delayFor))
		}

		attempts++

		row, err := reader.Get(l.id)
		if err != nil {
			errs.Add(fmt.Errorf("C* read back error: %v", err))
			continue
		}
		if row == nil || len(row.Columns) == 0 {
			errs.Add(fmt.Errorf("C* read back error: no columns returned from query"))
			continue
		}

		col := row.Columns[0]
		if bytes.Equal(col.Name, []byte(l.lockId[:])) {
			// we have the lock
			break
		}
	}

	inst.Timing(1.0, "sync.globaltimedlock.acquire", time.Since(startTime))
	inst.Counter(1.0, "sync.globaltimedlock.acquire.success")

	// put in place the refresher loop @todo
	go func() {
		for {
			log.Debug("[Sync:GlobalLock] Doing refresher loop…")
			refresh := time.Duration(float64(holdFor) * 0.75)
			select {
			case <-l.exit:
				log.Debugf("[Sync:GlobalLock] Breaking out of refresher loop")
				return
			case <-time.After(refresh):
				log.Debugf("[Sync:GlobalLock] Refreshing %s [%s]", string(l.id), l.lockId.String())
				writer.ConsistencyLevel(gossie.CONSISTENCY_QUORUM).Insert(cfGlobalLock, &gossie.Row{
					Key: l.id,
					Columns: []*gossie.Column{{
						Name:  []byte(l.lockId[:]),
						Value: []byte{},                        // @todo could inject some data about who has the lock here
						Ttl:   durationToSeconds(holdFor, 1.5), // 1.5 is because we renew the lock earlier than the timeout, so we need to cover that extra bit
					}},
				})
				if err := writer.Run(); err != nil {
					// @todo we could inform clients of this, somehow, eg: via a channel
					log.Warnf("[Sync:GlobalLock] failed to refresh lock .. cannot guarantee exclusivity")
				}
			}
		}
	}()

	return l, nil
}

func durationToSeconds(d time.Duration, factor float64) int32 {
	if factor != 0 {
		d = time.Duration(float64(d) * factor)
	}
	f := math.Ceil(float64(d.Seconds()))
	return int32(f)
}

func addJitter(d time.Duration) time.Duration {
	j := rand.Float64() * float64(d)
	return time.Duration(float64(d) + j)
}
