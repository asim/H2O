package hob

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/platform/multiclient"
	multiconfig "github.com/HailoOSS/hob-service/proto/multiconfig"
	readhob "github.com/HailoOSS/hob-service/proto/readhob"
	readservicetypes "github.com/HailoOSS/hob-service/proto/readservicetypes"
)

type HobService interface {
	ReadHob(hob string) (*Hob, error)
	ReadServiceTypes(hob string) (ServiceTypes, error)
	Multiconfig(hobs, hobHashes, serviceTypesHashes []string) (newHobs, newHobsConfigs, newHobsHashes, newServiceTypesConfigs, newServiceTypesHashes []string, err error)
}

var h2HobService HobService = &H2HobService{}

type H2HobService struct{}

func (c *H2HobService) ReadHob(hob string) (*Hob, error) {
	rsp := &readhob.Response{}
	cl := multiclient.New().DefaultScopeFrom(serverScope)
	cl.AddScopedReq(
		&multiclient.ScopedReq{
			Uid:      "readhob",
			Service:  "com.HailoOSS.service.hob",
			Endpoint: "readhob",
			Req:      &readhob.Request{Hob: proto.String(hob)},
			Rsp:      rsp,
		})
	if err := cl.Execute().Succeeded("readhob"); err != nil {
		log.Errorf("error while doing readhob: %v", err)
		return nil, err
	}
	log.Debugf("Config read for hob:%v", hob)
	return ProtoToHob(rsp.GetHob()), nil
}

func (c *H2HobService) ReadServiceTypes(hob string) (ServiceTypes, error) {
	rsp := &readservicetypes.Response{}
	cl := multiclient.New().DefaultScopeFrom(serverScope)
	cl.AddScopedReq(
		&multiclient.ScopedReq{
			Uid:      "readservicetypes",
			Service:  "com.HailoOSS.service.hob",
			Endpoint: "readservicetypes",
			Req:      &readservicetypes.Request{Hob: proto.String(hob)},
			Rsp:      rsp,
		})
	if err := cl.Execute().Succeeded("readservicetypes"); err != nil {
		log.Errorf("error while doing readservicetypes: %v", err)
		return nil, err
	}
	result := make(ServiceTypes)
	for _, serviceType := range rsp.GetServiceTypes() {
		result[serviceType.GetId()] = ProtoToServiceType(serviceType)
	}
	log.Debugf("Config read for ServiceTypes in hob:%v", hob)
	return ServiceTypes(result), nil
}

func (c *H2HobService) Multiconfig(hobs, hobHashes, serviceTypesHashes []string) (newHobs, newHobsConfigs, newHobsHashes, newServiceTypesConfigs, newServiceTypesHashes []string, err error) {
	log.Debugf("Background loading of hobs %v hobHashes:%v serviceTypeHashes:%v", hobs, hobHashes, serviceTypesHashes)
	if len(hobs) != len(hobHashes) {
		hobHashes = make([]string, len(hobs))
	}
	if len(hobs) != len(serviceTypesHashes) {
		serviceTypesHashes = make([]string, len(hobs))
	}
	rsp := &multiconfig.Response{}
	hobIds := make([]*multiconfig.Request_HobId, len(hobs))
	for i, hob := range hobs {
		hobIds[i] = &multiconfig.Request_HobId{
			Code:             proto.String(hob),
			HobHash:          proto.String(hobHashes[i]),
			ServiceTypesHash: proto.String(serviceTypesHashes[i]),
		}
	}

	cl := multiclient.New().DefaultScopeFrom(serverScope)
	cl.AddScopedReq(
		&multiclient.ScopedReq{
			Uid:      "multiconfig",
			Service:  "com.HailoOSS.service.hob",
			Endpoint: "multiconfig",
			Req: &multiconfig.Request{
				Ids: hobIds,
			},
			Rsp: rsp,
		})
	if err := cl.Execute().Succeeded("multiconfig"); err != nil {
		log.Errorf("error while reading hobs via multiconfig: %v", err)
		return nil, nil, nil, nil, nil, fmt.Errorf("error while reading hobs via multiconfig: %v", err)
	}
	hobsCount := len(rsp.GetHobConfigs())
	newHobs = make([]string, hobsCount)
	newHobsConfigs = make([]string, hobsCount)
	newHobsHashes = make([]string, hobsCount)
	newServiceTypesConfigs = make([]string, hobsCount)
	newServiceTypesHashes = make([]string, hobsCount)
	for i, hobConfig := range rsp.GetHobConfigs() {
		newHobs[i] = hobConfig.GetHob()
		newHobsConfigs[i] = hobConfig.GetHobConfig()
		newHobsHashes[i] = hobConfig.GetHobHash()
		newServiceTypesConfigs[i] = hobConfig.GetServiceTypesConfig()
		newServiceTypesHashes[i] = hobConfig.GetServiceTypesHash()
	}
	return newHobs, newHobsConfigs, newHobsHashes, newServiceTypesConfigs, newServiceTypesHashes, nil
}
