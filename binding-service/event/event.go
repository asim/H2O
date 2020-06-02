package event

import (
	"crypto/rand"
	"encoding/json"
	"os"
	"strconv"
	"time"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/platform/util"
	"github.com/HailoOSS/service/nsq"
	gouuid "github.com/nu7hatch/gouuid"
)

const (
	CreateRule = "CREATED"
	DeleteRule = "DELETED"
	nsqTopic   = "platform.events"
)

var (
	hostname string
	azName   string
)

func init() {
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

func PubRuleChange(service, version, action, user string, weight int32) {
	var uuid string
	u4, err := gouuid.NewV4()
	if err != nil {
		uuid = generatePseudoRand()
	} else {
		uuid = u4.String()
	}

	event := map[string]interface{}{
		"id":        uuid,
		"timestamp": strconv.Itoa(int(time.Now().Unix())),
		"type":      "com.HailoOSS.kernel.binding.event",
		"details": map[string]string{
			"ServiceName":    service,
			"ServiceVersion": version,
			"AzName":         azName,
			"Hostname":       hostname,
			"Action":         action,
			"Weight":         strconv.Itoa(int(weight)),
			"UserId":         user,
		},
	}

	bytes, err := json.Marshal(event)
	if err != nil {
		log.Errorf("Error marshaling nsq event message for %v:%v", event, err)
		return
	}
	err = nsq.Publish(nsqTopic, bytes)
	if err != nil {
		log.Errorf("Error publishing message to NSQ: %v", err)
		return
	}
}
