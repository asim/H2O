package handler

import (
	"fmt"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/config-service/domain"
	explain "github.com/HailoOSS/config-service/proto/explain"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// Explain will compile config and then explain from which ID the "winning" piece of config came
func Explain(req *server.Request) (proto.Message, errors.Error) {
	request := &explain.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.config.explain", fmt.Sprintf("%v", err))
	}

	config, err := domain.ExplainConfig(request.GetId(), request.GetPath())
	if err == domain.ErrPathNotFound {
		return nil, errors.NotFound("com.HailoOSS.service.config.explain", fmt.Sprintf("%v", err))
	}
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.config.explain", fmt.Sprintf("%v", err))
	}

	return &explain.Response{
		Config: proto.String(string(config)),
	}, nil
}
