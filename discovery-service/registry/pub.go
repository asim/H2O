package registry

import (
	log "github.com/cihub/seelog"
	servicedown "github.com/HailoOSS/discovery-service/proto/servicedown"
	serviceup "github.com/HailoOSS/discovery-service/proto/serviceup"
	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/protobuf/proto"
)

// pubServiceUp transmits via the platform the fact that we've come up
func pubServiceUp(inst *Instance) {
	pub, err := client.NewPublication(
		"com.HailoOSS.kernel.discovery.serviceup",
		&serviceup.Request{
			InstanceId:     proto.String(inst.Id),
			Hostname:       proto.String(inst.Hostname),
			ServiceName:    proto.String(inst.Name),
			ServiceVersion: proto.Uint64(inst.Version),
			AzName:         proto.String(inst.AzName),
			EndpointName:   proto.String(""),
			SubTopic:       inst.GetSubTopics(),
		},
	)
	if err != nil {
		log.Warnf("[Discovery] Failed to create serviceup message: %v", err)
	} else {
		err := client.AsyncTopic(pub)
		if err != nil {
			log.Warnf("[Discovery] Failed to publish serviceup: %v", err)
		}
	}
}

// pubServiceDown transmits via the platform the fact that we've gone down
func pubServiceDown(inst *Instance) {
	pub, err := client.NewPublication("com.HailoOSS.kernel.discovery.servicedown", &servicedown.Request{
		InstanceId:     proto.String(inst.Id),
		Hostname:       proto.String(inst.Hostname),
		ServiceName:    proto.String(inst.Name),
		ServiceVersion: proto.Uint64(inst.Version),
		AzName:         proto.String(inst.AzName),
		EndpointName:   proto.String(""),
	})
	if err != nil {
		log.Warn("[Discovery] Failed to create servicedown message ", err)
	} else {
		err := client.AsyncTopic(pub)
		if err != nil {
			log.Warn("[Discovery] Failed to publish servicedown ", err)
		}
	}
}
