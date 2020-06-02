package discovery

import (
	log "github.com/cihub/seelog"
	shared "github.com/HailoOSS/discovery-service/proto"
	endpoints "github.com/HailoOSS/discovery-service/proto/endpoints"
	instances "github.com/HailoOSS/discovery-service/proto/instances"
	services "github.com/HailoOSS/discovery-service/proto/services"
	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/raven"
	"github.com/HailoOSS/hshell/login"
	"github.com/HailoOSS/hshell/util"
	"github.com/HailoOSS/protobuf/proto"
	"sort"
	"strings"
)

var AmqpHost string

var bufferedEndpoints = make(map[string][]string)

func GetEndpointsBuffered(service string) []string {

	if bufferedEndpoints[service] == nil || util.CacheReload["getendpointsbuffered"] {
		GetEndpoints(service)
	} else {
		go GetEndpoints(service)
	}

	return bufferedEndpoints[service]
}

func GetEndpoints(service string) []string {

	endpts, err := callEndpoints(service)
	if err != nil {
		log.Error("could not call discovery service endpoints")
	}
	endptsStr := make([]string, 0)
	for _, endpt := range endpts {

		endptArr := strings.Split(endpt.GetFqName(), ".")

		endptsStr = append(endptsStr, endptArr[len(endptArr)-1])
	}
	util.RemoveDuplicates(&endptsStr)
	sort.Strings(endptsStr)
	bufferedEndpoints[service] = endptsStr
	util.CacheReload["getendpointsbuffered"] = false
	return endptsStr
}

func GetProtobufForEndpoint(endpoint string) string {

	endpointArr := strings.Split(endpoint, ".")
	endpointArrLen := len(endpointArr)
	service := ""
	for i, str := range endpointArr[:(endpointArrLen - 1)] {
		service += str
		if i != endpointArrLen-2 {
			service += "."
		}
	}

	gitStr := GetServiceGit(service)
	gitStr += "/proto/" + endpointArr[endpointArrLen-1]
	return gitStr
}

func GetServiceGit(service string) string {

	srvs, err := callServices(service)
	if err != nil {
		log.Error("could not call discovery service endpoints")
	}

	git := ""
	for _, srv := range srvs {
		git = srv.GetSource()
		break
	}

	return git
}

var bufferedServices = make([]string, 0)

func GetServicesBuffered() []string {
	if len(bufferedServices) == 0 || util.CacheReload["getservicesbuffered"] {
		GetServices()
	} else {
		go GetServices()
	}
	return bufferedServices
}

func GetServices() []string {

	srvs, err := callServices("")
	if err != nil {
		log.Error("could not call discovery service endpoints")
	}

	srvsStr := make([]string, 0)

	for _, srv := range srvs {
		srvsStr = append(srvsStr, srv.GetName())
	}
	util.RemoveDuplicates(&srvsStr)
	sort.Strings(srvsStr)
	bufferedServices = srvsStr
	util.CacheReload["getservicesbuffered"] = false
	return srvsStr
}

func orderServiceName(s1, s2 *shared.Service) bool {
	return s1.GetName() < s2.GetName()
}

func orderServiceVersion(s1, s2 *shared.Service) bool {
	return s1.GetVersion() < s2.GetVersion()
}

var bufferedVersions = make(map[string][]*shared.Service)

func GetServiceVersionsBuffered(service string) []*shared.Service {
	if bufferedVersions[service] == nil || util.CacheReload["getserviceversionsbuffered"] {
		GetServiceVersions(service)
	} else {
		go GetServiceVersions(service)
	}
	return bufferedVersions[service]
}

func GetServiceVersions(service string) []*shared.Service {

	srvs, err := callServices(service)
	if err != nil {
		log.Error("could not call discovery service endpoints")
	}

	OrderedBy(orderServiceName, orderServiceVersion).Sort(srvs)
	bufferedVersions[service] = srvs
	util.CacheReload["getserviceversionsbuffered"] = false
	return srvs
}

var bufferedMachineClasses = make([]string, 0)

func GetMachineClassesBuffered(azName string) []string {
	if bufferedMachineClasses == nil || util.CacheReload["getmachineclassesbuffered"] {
		GetMachineClasses(azName)
	} else {
		go GetMachineClasses(azName)
	}
	return bufferedMachineClasses
}

func GetMachineClasses(azName string) []string {
	classArr := make([]string, 0)

	instances, err := CallInstances(azName)
	if err != nil {
		log.Errorf("could not call discovery instances endpoint")
		return []string{}
	}

	classMap := make(map[string]bool, 0)
	for _, instance := range instances {
		classMap[instance.GetMachineClass()] = true
	}

	for mc, _ := range classMap {
		classArr = append(classArr, mc)
	}

	bufferedMachineClasses = classArr
	util.CacheReload["getmachineclassesbuffered"] = false
	return classArr
}

func ReconnectClient() {
	if len(AmqpHost) != 0 {
		raven.AmqpUri = AmqpHost
	}
	<-raven.Connect()
	//raven.Disconnect()
}

func callEndpoints(service string) ([]*endpoints.Response_Endpoint, error) {

	if len(AmqpHost) != 0 {
		raven.AmqpUri = AmqpHost
	}

	request, _ := client.NewRequest(
		"com.HailoOSS.kernel.discovery",
		"endpoints",
		&endpoints.Request{
			Service: proto.String(service),
		},
	)
	request.SetSessionID(login.Session)
	request.SetFrom(login.FromService)
	response := &endpoints.Response{}
	if err := util.SendRequest(request, response); err != nil {
		log.Critical(err)
		return nil, err
	}

	return response.Endpoints, nil

}

func callServices(service string) ([]*shared.Service, error) {
	if len(AmqpHost) != 0 {
		raven.AmqpUri = AmqpHost
	}
	request, _ := client.NewRequest(
		"com.HailoOSS.kernel.discovery",
		"services",
		&services.Request{
			Service: proto.String(service),
		},
	)

	request.SetSessionID(login.Session)
	request.SetFrom(login.FromService)
	response := &services.Response{}
	if err := util.SendRequest(request, response); err != nil {
		log.Critical(err)
		return nil, err
	}

	return response.Services, nil

}

func CallInstances(az string) ([]*instances.Instance, error) {

	request, _ := client.NewRequest(
		"com.HailoOSS.kernel.discovery",
		"instances",
		&instances.Request{
			AzName: proto.String(az),
		},
	)

	request.SetSessionID(login.Session)
	request.SetFrom(login.FromService)
	response := &instances.Response{}
	if err := util.SendRequest(request, response); err != nil {
		log.Critical(err)
		return nil, err
	}

	return response.Instances, nil

}
