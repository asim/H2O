package binding

import (
	shared "github.com/HailoOSS/binding-service/proto"
	createrule "github.com/HailoOSS/binding-service/proto/createrule"
	deleterule "github.com/HailoOSS/binding-service/proto/deleterule"
	listrules "github.com/HailoOSS/binding-service/proto/listrules"
	"github.com/HailoOSS/hshell/login"
	"github.com/HailoOSS/hshell/util"
	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/protobuf/proto"
	log "github.com/cihub/seelog"
)

var bufferedRules map[string][]*shared.BindingRule = make(map[string][]*shared.BindingRule)

func ListRulesBuffered(service string) []*shared.BindingRule {

	if bufferedRules[service] == nil || util.CacheReload["listrulesbuffered"] {
		ListRules(service)
	} else {
		go ListRules(service)
	}

	return bufferedRules[service]
}
func ListRules(service string) ([]*shared.BindingRule, error) {
	rules, err := callListRules(service)
	bufferedRules[service] = rules
	util.CacheReload["listrulesbuffered"] = false
	return rules, err
}

func DeleteRule(service string, version string, weight int) error {
	return callDeleteRule(service, version, weight)
}

func CreateRule(service string, version string, weight int) error {
	return callCreateRule(service, version, weight)
}

func callCreateRule(service string, version string, weight int) error {
	request, _ := client.NewRequest(
		"com.HailoOSS.kernel.binding",
		"createrule",
		&createrule.Request{
			Rule: &shared.BindingRule{
				Service: proto.String(service),
				Version: proto.String(version),
				Weight:  proto.Int32(int32(weight)),
			},
		},
	)
	request.SetSessionID(login.Session)
	request.SetFrom(login.FromService)
	response := &createrule.Response{}
	if err := util.SendRequest(request, response); err != nil {
		log.Critical(err)
		return err
	}

	return nil
}

func callDeleteRule(service string, version string, weight int) error {

	request, _ := client.NewRequest(
		"com.HailoOSS.kernel.binding",
		"deleterule",
		&deleterule.Request{
			Rule: &shared.BindingRule{
				Service: proto.String(service),
				Version: proto.String(version),
				Weight:  proto.Int32(int32(weight)),
			},
		},
	)
	request.SetSessionID(login.Session)
	request.SetFrom(login.FromService)
	response := &deleterule.Response{}
	if err := util.SendRequest(request, response); err != nil {
		log.Critical(err)
		return err
	}

	return nil
}

func callListRules(service string) ([]*shared.BindingRule, error) {

	request, _ := client.NewRequest(
		"com.HailoOSS.kernel.binding",
		"listrules",
		&listrules.Request{
			Service: proto.String(service),
		},
	)
	request.SetSessionID(login.Session)
	request.SetFrom(login.FromService)
	response := &listrules.Response{}
	if err := util.SendRequest(request, response); err != nil {
		log.Critical(err)
		return nil, err
	}

	return response.GetRules(), nil
}
