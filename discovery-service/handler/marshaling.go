package handler

import (
	"fmt"
	commonproto "github.com/HailoOSS/discovery-service/proto"
	endpoints "github.com/HailoOSS/discovery-service/proto/endpoints"
	instances "github.com/HailoOSS/discovery-service/proto/instances"
	register "github.com/HailoOSS/discovery-service/proto/register"
	"github.com/HailoOSS/discovery-service/registry"
	"github.com/HailoOSS/protobuf/proto"
)

// multiRegToInstance marshals a multi register request proto into an instance
func multiRegToInstance(request *register.MultiRequest) *registry.Instance {
	inst := &registry.Instance{
		Id:           request.GetInstanceId(),
		Hostname:     request.GetHostname(),
		MachineClass: request.GetMachineClass(),
		Name:         request.GetService().GetName(),
		Description:  request.GetService().GetDescription(),
		Version:      request.GetService().GetVersion(),
		AzName:       request.GetAzName(),
		Source:       request.GetService().GetSource(),
		OwnerEmail:   request.GetService().GetOwnerEmail(),
		OwnerMobile:  request.GetService().GetOwnerMobile(),
		OwnerTeam:    request.GetService().GetOwnerTeam(),
		Endpoints:    make([]*registry.Endpoint, 0),
	}
	for _, endpoint := range request.GetEndpoints() {
		inst.Endpoints = append(inst.Endpoints, &registry.Endpoint{
			Name:      endpoint.GetName(),
			Subscribe: endpoint.GetSubscribe(),
			Sla: registry.Sla{
				Mean:    uint32(endpoint.GetMean()),
				Upper95: uint32(endpoint.GetUpper95()),
			},
		})
	}
	return inst
}

// instancesToEndpointsProto extracts each endpoint from instances and marshals to proto
func instancesToEndpointsProto(insts registry.Instances) []*endpoints.Response_Endpoint {
	ret := make([]*endpoints.Response_Endpoint, 0)
	seen := make(map[string]bool)
	for _, inst := range insts {
		for _, ep := range inst.Endpoints {
			uid := fmt.Sprintf("%v.%v|%v", inst.Name, ep.Name, inst.Version)
			if seen[uid] {
				continue
			}
			ret = append(ret, &endpoints.Response_Endpoint{
				FqName:  proto.String(fmt.Sprintf("%v.%v", inst.Name, ep.Name)),
				Version: proto.Uint64(inst.Version),
				Mean:    proto.Uint32(ep.Sla.Mean),
				Upper95: proto.Uint32(ep.Sla.Upper95),
			})
			seen[uid] = true
		}
	}
	return ret
}

// instancesToServicesProto turns instances into a list of services - deduped on name + version
func instancesToServicesProto(insts registry.Instances) []*commonproto.Service {
	ret := make([]*commonproto.Service, 0)
	seen := make(map[string]bool)
	for _, inst := range insts {
		uid := fmt.Sprintf("%v|%v", inst.Name, inst.Version)
		if seen[uid] {
			continue
		}
		ret = append(ret, &commonproto.Service{
			Name:        proto.String(inst.Name),
			Description: proto.String(inst.Description),
			Version:     proto.Uint64(inst.Version),
			Source:      proto.String(inst.Source),
			OwnerEmail:  proto.String(inst.OwnerEmail),
			OwnerMobile: proto.String(inst.OwnerMobile),
			OwnerTeam:   proto.String(inst.OwnerTeam),
		})
		seen[uid] = true
	}
	return ret
}

// instancesToProto turns instances into a list of -- wait for it folks -- instances! (albeit in proto format)
func instancesToProto(insts registry.Instances) []*instances.Instance {
	ret := make([]*instances.Instance, 0)
	for _, inst := range insts {
		protoInst := &instances.Instance{
			InstanceId:         proto.String(inst.Id),
			Hostname:           proto.String(inst.Hostname),
			MachineClass:       proto.String(inst.MachineClass),
			ServiceName:        proto.String(inst.Name),
			ServiceDescription: proto.String(inst.Description),
			ServiceVersion:     proto.Uint64(inst.Version),
			AzName:             proto.String(inst.AzName),
			SubTopic:           make([]string, 0),
		}
		for _, ep := range inst.Endpoints {
			if ep.Subscribe != "" {
				protoInst.SubTopic = append(protoInst.SubTopic, ep.Subscribe)
			}
		}
		ret = append(ret, protoInst)
	}
	return ret
}
