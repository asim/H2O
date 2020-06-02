package handler

import (
	"fmt"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/config-service/domain"
	compile "github.com/HailoOSS/config-service/proto/compile"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// Compile constructs a single, merged, view of config, combining many individual elements
func Compile(req *server.Request) (proto.Message, errors.Error) {
	request := &compile.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.config.compile", fmt.Sprintf("%v", err))
	}

	cfg, hash, err := DoCompile(request.GetId(), request.GetPath())
	if err != nil {
		return nil, err
	}

	return &compile.Response{
		Config: proto.String(cfg),
		Hash:   proto.String(hash),
	}, nil
}

// DoCompile does the real work for compile - and is implemented like this because we want
// an HTTP interface in addition to the platform interface
func DoCompile(ids []string, path string) (config, hash string, compileErr errors.Error) {
	cfg, err := domain.CompileConfig(ids, path)
	if err == domain.ErrPathNotFound {
		compileErr = errors.NotFound("com.HailoOSS.service.config.compile", fmt.Sprintf("%v", err))
		return
	}
	if err != nil {
		compileErr = errors.InternalServerError("com.HailoOSS.service.config.compile", fmt.Sprintf("%v", err))
		return
	}

	config = string(cfg)
	hash = createConfigHash(cfg)

	return
}
