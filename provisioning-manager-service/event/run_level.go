package event

/*
   Run level events. Kept separate since they'll be moved out eventually.
*/

import (
	"encoding/json"
	log "github.com/cihub/seelog"
	"github.com/HailoOSS/service/nsq"
	gouuid "github.com/nu7hatch/gouuid"
	"strconv"
	"strings"
	"time"
)

func pubNSQEvent(event *NSQEvent) {
	bytes, err := json.Marshal(event)
	if err != nil {
		log.Errorf("Error marshaling nsq event message for %v:%v", event, err)
		return
	}
	err = nsq.Publish(nsqTopicName, bytes)
	if err != nil {
		log.Errorf("Error publishing message to NSQ: %v", err)
		return
	}
}

func setServiceRunLevelsToNSQ(serviceName string, runLevels []string, user string) {
	var uuid string
	u4, err := gouuid.NewV4()
	if err != nil {
		uuid = generatePseudoRand()
	} else {
		uuid = u4.String()
	}

	pubNSQEvent(&NSQEvent{
		Id:        uuid,
		Timestamp: strconv.Itoa(int(time.Now().Unix())),
		Type:      "com.HailoOSS.platform.runlevel.service",
		Details: map[string]string{
			"Service":   serviceName,
			"RunLevels": strings.Join(runLevels, ", "),
			"UserId":    user,
		},
	})
}

func setRegionRunLevelToNSQ(region, runLevel, user string) {
	var uuid string
	u4, err := gouuid.NewV4()
	if err != nil {
		uuid = generatePseudoRand()
	} else {
		uuid = u4.String()
	}

	pubNSQEvent(&NSQEvent{
		Id:        uuid,
		Timestamp: strconv.Itoa(int(time.Now().Unix())),
		Type:      "com.HailoOSS.platform.runlevel.region",
		Details: map[string]string{
			"Region":   region,
			"RunLevel": runLevel,
			"UserId":   user,
		},
	})
}

func SetRegionRunLevel(region, runLevel, user string) {
	setRegionRunLevelToNSQ(region, runLevel, user)
}

func SetServiceRunLevels(serviceName string, runLevels []string, user string) {
	setServiceRunLevelsToNSQ(serviceName, runLevels, user)
}
