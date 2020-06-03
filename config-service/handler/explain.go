package handler

import (
	"fmt"

	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/config-service/domain"
	explain "github.com/hailo-platform/H2O/config-service/proto/explain"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

// Explain will compile config and then explain from which ID the "winning" piece of config came
func Explain(req *server.Request) (proto.Message, errors.Error) {
	request := &explain.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.hailocab.service.config.explain", fmt.Sprintf("%v", err))
	}

	config, err := domain.ExplainConfig(request.GetId(), request.GetPath())
	if err == domain.ErrPathNotFound {
		return nil, errors.NotFound("com.hailocab.service.config.explain", fmt.Sprintf("%v", err))
	}
	if err != nil {
		return nil, errors.InternalServerError("com.hailocab.service.config.explain", fmt.Sprintf("%v", err))
	}

	return &explain.Response{
		Config: proto.String(string(config)),
	}, nil
}
