package handler

import (
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/config-service/domain"
	read "github.com/hailo-platform/H2O/config-service/proto/read"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

// Read will read a single ID config - and should only be used when editing config (use compile when reading for use)
func Read(req *server.Request) (proto.Message, errors.Error) {
	request := &read.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest(server.Name+".read", err.Error())
	}

	config, change, err := domain.ReadConfig(request.GetId(), request.GetPath())
	if err == domain.ErrPathNotFound || err == domain.ErrIdNotFound {
		return nil, errors.NotFound(server.Name+".read.notfound", err.Error())
	}
	if err != nil {
		return nil, errors.InternalServerError(server.Name+".read", err.Error())
	}

	return &read.Response{
		Config: proto.String(string(config)),
		Hash:   proto.String(createConfigHash(config)),
		Meta:   changeToProto(change),
	}, nil
}
