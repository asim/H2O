package handler

import (
	log "github.com/cihub/seelog"
	"github.com/HailoOSS/binding-service/dao"
	"github.com/HailoOSS/binding-service/domain"
	"github.com/HailoOSS/binding-service/event"
	rule "github.com/HailoOSS/binding-service/proto"
	createrule "github.com/HailoOSS/binding-service/proto/createrule"
	deleterule "github.com/HailoOSS/binding-service/proto/deleterule"
	listrules "github.com/HailoOSS/binding-service/proto/listrules"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
)

func getUser(req *server.Request) string {
	if req.Auth() != nil && req.Auth().AuthUser() != nil {
		return req.Auth().AuthUser().Id
	}

	return "unknown"
}

func CreateBindingRuleHandler(req *server.Request) (proto.Message, errors.Error) {
	request := &createrule.Request{}
	err := req.Unmarshal(request)
	if err != nil {
		return nil, errors.BadRequest("com.HailoOSS.kernel.binding.createrule", err.Error())
	}
	ruleReq := request.GetRule()
	rule := &domain.Rule{Service: ruleReq.GetService(), Version: ruleReq.GetVersion(), Weight: ruleReq.GetWeight()}
	err = dao.CreateRule(rule)
	if err != nil {
		log.Errorf("Error creating rule %+v", err)
		return nil, errors.InternalServerError("com.HailoOSS.kernel.binding.createrule", err.Error())
	}

	event.PubRuleChange(ruleReq.GetService(), ruleReq.GetVersion(), event.CreateRule, getUser(req), ruleReq.GetWeight())

	// once a rule is created you need to rebind everything - just wait for the periodic refresh to pick it up
	return &createrule.Response{Ok: proto.Bool(true)}, nil
}

func DeleteBindingRuleHandler(req *server.Request) (proto.Message, errors.Error) {
	request := &deleterule.Request{}
	err := req.Unmarshal(request)
	if err != nil {
		return nil, errors.BadRequest("com.HailoOSS.kernel.binding.deleterule", err.Error())
	}
	ruleReq := request.GetRule()
	rule := &domain.Rule{Service: ruleReq.GetService(), Version: ruleReq.GetVersion(), Weight: ruleReq.GetWeight()}
	err = dao.DeleteRule(rule)
	if err != nil {
		log.Errorf("Error deleting rule %+v", err)
		return nil, errors.InternalServerError("com.HailoOSS.kernel.binding.deleterule", err.Error())
	}

	event.PubRuleChange(ruleReq.GetService(), ruleReq.GetVersion(), event.DeleteRule, getUser(req), ruleReq.GetWeight())

	// once a rule is deleted you need to rebind everything - just wait for the periodic refresh to pick it up
	return &deleterule.Response{Ok: proto.Bool(true)}, nil
}

func ListBindingRulesHandler(req *server.Request) (proto.Message, errors.Error) {
	request := &listrules.Request{}
	err := req.Unmarshal(request)
	if err != nil {
		return nil, errors.BadRequest("com.HailoOSS.kernel.binding.listrules", err.Error())
	}
	rules, err := dao.GetRules(request.GetService())
	if err != nil {
		log.Errorf("Error listing rules %+v", err)
		return nil, errors.InternalServerError("com.HailoOSS.kernel.binding.listrules", err.Error())
	}
	ret := make([]*rule.BindingRule, 0)
	for _, r := range rules {
		ret = append(ret, &rule.BindingRule{Service: &r.Service, Version: &r.Version, Weight: &r.Weight})
	}
	return &listrules.Response{Rules: ret}, nil
}
