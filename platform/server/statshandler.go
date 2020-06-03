package server

import (
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/stats"
)

// registerStats starts the runtime stats collection and monitoring
func registerStats() {
	stats.ServiceName = Name
	stats.ServiceVersion = Version
	stats.ServiceType = "h2.platform"
	stats.InstanceID = InstanceID
	for _, ep := range reg.iterate() {
		stats.RegisterEndpoint(ep)
	}

	stats.Start()
}

// statsHandler handles inbound requests to `stats` endpoint
func statsHandler(req *Request) (proto.Message, errors.Error) {
	return stats.Get(), nil
}
