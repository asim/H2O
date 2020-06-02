package event

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/util"
	"github.com/HailoOSS/service/nsq"
	"github.com/HailoOSS/protobuf/proto"
	pproto "github.com/HailoOSS/provisioning-service/proto"
	gouuid "github.com/nu7hatch/gouuid"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	provisioned      = "PROVISIONED"
	deprovisioned    = "DEPROVISIONED"
	provisionError   = "ERROR PROVISIONING"
	deprovisionError = "ERROR DEPROVISIONING"
	restarted        = "RESTARTED"
	eventTTL         = 60
	eventExpiry      = 3600
	nsqTopicName     = "platform.events"
)

var (
	mclass         string
	hostname       string
	azName         string
	defaultManager = newEventManager()
)

type event struct {
	action string
	info   string
	at     time.Time
}

type NSQEvent struct {
	Id        string
	Type      string
	Timestamp string
	Details   map[string]string
}

type eventManager struct {
	mtx     sync.Mutex
	events  map[string]*event
	lastRun time.Time
}

func init() {
	mclass = os.Getenv("H2O_MACHINE_CLASS")
	if len(mclass) == 0 {
		mclass = "default"
	}

	var err error
	if hostname, err = os.Hostname(); err != nil {
		hostname = "localhost.unknown"
	}

	if azName, err = util.GetAwsAZName(); err != nil {
		azName = "unknown"
	}
}

// generatePseudoRand is used in the rare event of proper uuid generation failing
func generatePseudoRand() string {
	alphanum := "0123456789abcdefghigklmnopqrst"
	var bytes = make([]byte, 10)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

func newEventManager() *eventManager {
	return &eventManager{
		events:  make(map[string]*event),
		lastRun: time.Now(),
	}
}

func eventProto(service string, version uint64, action, info string) *pproto.Event {
	return &pproto.Event{
		ServiceName:    proto.String(service),
		ServiceVersion: proto.Uint64(version),
		MachineClass:   proto.String(mclass),
		Hostname:       proto.String(hostname),
		AzName:         proto.String(azName),
		Action:         proto.String(action),
		Info:           proto.String(info),
		Timestamp:      proto.Int64(time.Now().Unix()),
	}
}

func eventToNSQ(service string, version uint64, action, info, mClass, user string) *NSQEvent {
	var uuid string
	u4, err := gouuid.NewV4()
	if err != nil {
		uuid = generatePseudoRand()
	} else {
		uuid = u4.String()
	}

	return &NSQEvent{
		Id:        uuid,
		Timestamp: strconv.Itoa(int(time.Now().Unix())),
		Type:      "com.HailoOSS.kernel.provisioning.event",
		Details: map[string]string{
			"ServiceName":    service,
			"ServiceVersion": strconv.Itoa(int(version)),
			"MachineClass":   mClass,
			"Hostname":       hostname,
			"AzName":         azName,
			"Action":         action,
			"Info":           info,
			"UserId":         user,
		},
	}
}

// cleanup removes services from the list that we haven't seen events for since the expiry.
// Since provisioning is long running we dont want to keep too many items in memory.
// Over optimization.
func (e *eventManager) cleanup() {
	now := time.Now()

	if now.Sub(e.lastRun).Seconds() < eventExpiry {
		return
	}

	for s, ev := range e.events {
		if now.Sub(ev.at).Seconds() > eventExpiry {
			delete(e.events, s)
		}
	}

	e.lastRun = now
}

// pub publishes a provisioning event. It checks to see whether this event was pubbed
// within the last 60 seconds in which case it does nothing.
func (e *eventManager) pub(service string, version uint64, action, info string) {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	e.cleanup()
	now := time.Now()
	name := fmt.Sprintf("%s%d", service, version)

	ev, ok := e.events[name]
	if ok {
		if ev.action == action && now.Sub(ev.at).Seconds() < eventTTL {
			return
		}
	}

	p := eventProto(service, version, action, info)

	if err := client.Pub("com.HailoOSS.kernel.provisioning.event", p); err != nil {
		log.Errorf("Failed to publish provisioning event: %v", err)
		return
	}

	if ev == nil {
		ev = &event{}
		e.events[name] = ev
	}

	ev.at = now
	ev.action = action
}

// pubNSQ publishes a provisioning event to NSQ
func (e *eventManager) pubNSQ(service string, version uint64, action, info, mClass, user string) {
	event := eventToNSQ(service, version, action, info, mClass, user)
	bytes, err := json.Marshal(event)
	if err != nil {
		log.Errorf("Error marshaling nsq event message for %v:%v", service, err)
		return
	}
	err = nsq.Publish(nsqTopicName, bytes)
	if err != nil {
		log.Errorf("Error publishing message to NSQ: %v", err)
		return
	}
}

// ProvisionError publishes a provisioning error event.
func ProvisionError(service string, version uint64, err string) {
	defaultManager.pub(service, version, provisionError, err)
}

// DeprovisionError publishes a deprovisioning error event.
func DeprovisionError(service string, version uint64, err string) {
	defaultManager.pub(service, version, deprovisionError, err)
}

// Provisioned publishes a provisioning event which other services can listen for.
func Provisioned(service string, version uint64) {
	defaultManager.pub(service, version, provisioned, "")
}

// Deprovisioned publishes a deprovisioning event which other services can listen for.
func Deprovisioned(service string, version uint64) {
	defaultManager.pub(service, version, deprovisioned, "")
}

// ProvisionedToNSQ publishes a provisioning event to NSQ
func ProvisionedToNSQ(service string, version uint64, mClass, user string) {
	defaultManager.pubNSQ(service, version, provisioned, "", mClass, user)
}

// DeprovisionedToNSQ publishes a deprovisioning event to NSQ
func DeprovisionedToNSQ(service string, version uint64, mClass, user string) {
	defaultManager.pubNSQ(service, version, deprovisioned, "", mClass, user)
}

// RestartedToNSQ publishes a service restart event to NSQ
func RestartedToNSQ(service string, version uint64, user string) {
	defaultManager.pubNSQ(service, version, restarted, "", mclass, user)
}
