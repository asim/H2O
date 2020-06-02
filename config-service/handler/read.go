package handler

import (
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/config-service/domain"
	read "github.com/HailoOSS/config-service/proto/read"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
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
