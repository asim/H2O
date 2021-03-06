package handler

import (
	"fmt"

	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/login-service/dao"
	grantservice "github.com/hailo-platform/H2O/login-service/proto/grantservice"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

const (
	thinAPI = "com.hailocab.hailo-2-api"
)

// GrantService will authorise a specific service to be authorised to talk to a specific endpoint
// on another service, automatically granting a specific role
func GrantService(req *server.Request) (proto.Message, errors.Error) {
	request := &grantservice.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest(server.Name+".grantservice.unmarshal", err.Error())
	}

	if len(request.GetEndpoint().GetGranted()) == 0 {
		return nil, errors.BadRequest(server.Name+".grantservice.empty", "No services in granted list")
	}

	// make sure we do not allow granting access for thin API, since this can open services to the world
	for _, gs := range request.GetEndpoint().GetGranted() {
		if gs.GetName() == thinAPI {
			return nil, errors.BadRequest(server.Name+".grantservice.thinapi", fmt.Sprintf("Not allowed to add auth for %s", thinAPI))
		}
	}

	epas := protoToEndpointAuth(request.GetEndpoint())

	if err := dao.WriteEndpointAuths(epas); err != nil {
		return nil, errors.InternalServerError(server.Name+".grantservice.dao", err.Error())
	}

	return &grantservice.Response{}, nil
}
