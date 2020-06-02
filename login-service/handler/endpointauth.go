package handler

import (
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	protoep "github.com/HailoOSS/login-service/proto"
	endpointauth "github.com/HailoOSS/login-service/proto/endpointauth"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// EndpointAuth will return a list of services that are allowed to talk to endpoints
// on a given service
func EndpointAuth(req *server.Request) (proto.Message, errors.Error) {
	request := &endpointauth.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.login.endpointauth.unmarshal", err.Error())
	}

	// authorise by service-to-service header
	s := request.GetService()
	if req.From() != s {
		if !req.Auth().IsAuth() || !req.Auth().AuthUser().HasRole("ADMIN") {
			return nil, errors.Forbidden("com.HailoOSS.service.login.endpointauth.auth", "Permission denied (unauthorised role)")
		}
	}

	authorised, err := dao.ReadEndpointAuth(s)
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.login.endpointauth.dao", err.Error())
	}

	rsp := &endpointauth.Response{
		Endpoints: endpointAuthsToProto(authorised),
	}

	return rsp, nil
}

// endpointAuthsToProto marshals domain -> proto
func endpointAuthsToProto(endpointAuths []*domain.EndpointAuth) []*protoep.Endpoint {
	ret := make([]*protoep.Endpoint, 0)

	endpoints := make(map[string]*protoep.Endpoint)
	for _, epa := range endpointAuths {
		fqep := epa.FqEndpoint()
		if _, ok := endpoints[fqep]; !ok {
			endpoints[fqep] = &protoep.Endpoint{
				Service:  proto.String(epa.ServiceName),
				Endpoint: proto.String(epa.EndpointName),
				Granted:  make([]*protoep.Service, 0),
			}
		}
		// now add this specific allowed service to the list
		endpoints[fqep].Granted = append(endpoints[fqep].Granted, &protoep.Service{
			Name: proto.String(epa.AllowedService),
			Role: proto.String(epa.Role),
		})
	}

	// now rejig into a slice
	for _, protoep := range endpoints {
		ret = append(ret, protoep)
	}

	return ret
}
