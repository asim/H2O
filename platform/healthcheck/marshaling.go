package healthcheck

import (
	hcproto "github.com/hailo-platform/H2O/platform/proto/healthcheck"
	"github.com/hailo-platform/H2O/protobuf/proto"
)

func healthCheckSampleToProto(hc *HealthCheck, sample *Sample) *hcproto.HealthCheck {
	return &hcproto.HealthCheck{
		Timestamp:        proto.Int64(sample.At.Unix()),
		HealthCheckId:    proto.String(hc.Id),
		ServiceName:      proto.String(hc.ServiceName),
		ServiceVersion:   proto.Uint64(hc.ServiceVersion),
		Hostname:         proto.String(hc.Hostname),
		InstanceId:       proto.String(hc.InstanceId),
		IsHealthy:        proto.Bool(sample.IsHealthy),
		ErrorDescription: proto.String(sample.ErrorDescription),
		Measurements:     mapToProto(sample.Measurements),
		Priority:         hcproto.HealthCheck_Priority(hc.Priority).Enum(),
	}
}

func mapToProto(m map[string]string) []*hcproto.HealthCheck_KeyValue {
	if m == nil {
		return []*hcproto.HealthCheck_KeyValue{}
	}
	ret := make([]*hcproto.HealthCheck_KeyValue, len(m))
	i := 0
	for k, v := range m {
		ret[i] = &hcproto.HealthCheck_KeyValue{
			Key:   proto.String(k),
			Value: proto.String(v),
		}
		i++
	}
	return ret
}
