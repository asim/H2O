package schema

import (
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"

	"github.com/hailo-platform/H2O/go-hailo-lib/schema"
	schemaProto "github.com/hailo-platform/H2O/platform/proto/schema"
	service "github.com/hailo-platform/H2O/platform/server"
)

func Endpoint(name string, configStruct interface{}) *service.Endpoint {
	handler := func(req *server.Request) (proto.Message, errors.Error) {
		return &schemaProto.Response{
			Schema: proto.String(schema.Of(configStruct).String()),
		}, nil
	}

	return &server.Endpoint{
		Name:       name,
		Mean:       200,
		Upper95:    400,
		Handler:    handler,
		Authoriser: service.OpenToTheWorldAuthoriser(),
	}
}
