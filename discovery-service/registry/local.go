package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/nu7hatch/gouuid"

	"github.com/HailoOSS/discovery-service/heartbeat"
	"github.com/HailoOSS/platform/raven"
	zk "github.com/HailoOSS/service/zookeeper"
	gozk "github.com/HailoOSS/go-zookeeper/zk"
)

const (
	heartbeatInterval = 10 * time.Second
	maxHeartbeatDiff  = 30 * time.Second
	initAttempts      = 30
	initDelay         = time.Second
)

type localReg struct {
	sync.RWMutex
	aliveInstances map[string]*heartbeat.Heartbeat
	id             string
	hostname       string
}

func newLocalReg() *localReg {
	r := &localReg{
		aliveInstances: make(map[string]*heartbeat.Heartbeat),
	}
	r.hostname, _ = os.Hostname()
	uuid, _ := uuid.NewV4()
	r.id = "discovery-" + uuid.String()

	log.Infof("[Discovery] Initialising local registry on %v...", r.hostname)

	// create root node
	attempts := 0
	for {
		var (
			err    error
			exists bool
		)
		exists, _, err = zk.Exists(rootNode)
		if err == nil {
			if exists {
				break
			}
			log.Infof("[Discovery] Creating root node %v...", rootNode)
			_, err = zk.Create(rootNode, []byte{}, 0, gozk.WorldACL(gozk.PermAll))
			if err == nil {
				break
			}
		}

		// some error
		attempts++
		if attempts > initAttempts {
			log.Criticalf("[Discovery] Failed to check/create root node %v times -- unable to initialise discovery service so exiting", attempts)
			os.Exit(3)
		}
		log.Warnf("[Discovery] Failed to check/create root node: %v -- delaying for %v", err, initDelay)
		time.Sleep(initDelay)
	}

	// listen for incoming heartbeat responses & send out heartbeats
	go r.listenHearbeats()

	// use a ticker to send HBs because we quite want them to go regularly, rather than sleeping for example
	go func() {
		tick := time.NewTicker(heartbeatInterval)
		for {
			<-tick.C
			r.sendHeartbeats()
		}

	}()

	return r
}

// inboundHearbeats processes deliveries from AMQP
func (r *localReg) listenHearbeats() {
	if deliveries, err := raven.Consume(r.id); err != nil {
		log.Criticalf("[Discovery] Fail to consume from %v: %v", r.id, err)
	} else {
		for d := range deliveries {
			r.RLock()
			hb, ok := r.aliveInstances[d.ReplyTo]
			r.RUnlock()
			if ok {
				hb.Beat()
			}
		}
	}

	// heartbeat AMQP connection died, so kill discovery service
	os.Exit(1)
}

// sendHeartbeats takes a snapshot of all alive instances, tests heartbeats, removes unhealthy ones
// or pings a message to healthy ones
func (r *localReg) sendHeartbeats() {
	// take a snapshot
	r.RLock()
	alive := make([]*heartbeat.Heartbeat, len(r.aliveInstances))
	i := 0
	for _, hb := range r.aliveInstances {
		alive[i] = hb
		i++
	}
	r.RUnlock()

	log.Debugf("[Discovery] Sending heartbeats to %v instances", len(alive))

	for _, hb := range alive {
		// remove if not alive
		if !hb.Healthy() {
			go r.remove(hb.Id)
			continue
		}

		if err := raven.SendHeartbeat(hb, r.id); err != nil {
			log.Warnf("[Discovery] Error sending HB: %v", err)
		}
	}
}

// add will add this instance to the local registry
func (r *localReg) add(i *Instance) error {
	b, err := json.Marshal(i)
	if err != nil {
		return fmt.Errorf("[Discovery] Failed to marshal instance JSON: %v", err)
	}

	_, err = zk.Create(i.zkPath(), b, gozk.FlagEphemeral, gozk.WorldACL(gozk.PermAll))
	// If the node already exists then ignore the error
	if err != gozk.ErrNodeExists && err != nil {
		return fmt.Errorf("Failed to add %v to local registry: %v", i, err)
	}

	// squirrel into our list, so we send heartbeats
	r.Lock()
	defer r.Unlock()
	r.aliveInstances[i.Id] = heartbeat.New(i.Id, maxHeartbeatDiff)

	go pubServiceUp(i)

	return nil
}

// remove will remove this instance ID from the local registry
func (r *localReg) remove(instanceId string) error {
	// try to grab instance details before we start, for broadcast msg
	i := region.singleInstance(instanceId)

	r.Lock()
	defer r.Unlock()

	// try to delete
	path := zkPathForInstance(instanceId)
	if err := zk.Delete(path, -1); err != nil {
		// exists?
		if exists, _, exErr := zk.Exists(path); exErr == nil && !exists {
			// assume replay, simply carry on
			return nil
		}
		return err
	}

	if _, ok := r.aliveInstances[instanceId]; ok {
		delete(r.aliveInstances, instanceId)
	}
	if i != nil {
		go pubServiceDown(i)
	}

	return nil
}
