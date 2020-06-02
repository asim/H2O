package event

import (
	"crypto/rand"
	"os"
	"strconv"
	"time"

	"github.com/HailoOSS/platform/util"
	gouuid "github.com/nu7hatch/gouuid"
)

const (
	provisioned   = "PROVISIONED"
	deprovisioned = "DEPROVISIONED"
	nsqTopicName  = "platform.events"
)

var (
	mclass   string
	hostname string
	azName   string
)

type NSQEvent struct {
	Id        string
	Type      string
	Timestamp string
	Details   map[string]string
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

// ProvisionedToNSQ publishes a provisioning event to NSQ
func ProvisionedToNSQ(service string, version uint64, mClass, user string) {
	pubNSQEvent(eventToNSQ(service, version, provisioned, "", mClass, user))
}

// DeprovisionedToNSQ publishes a deprovisioning event to NSQ
func DeprovisionedToNSQ(service string, version uint64, mClass, user string) {
	pubNSQEvent(eventToNSQ(service, version, deprovisioned, "", mClass, user))
}
