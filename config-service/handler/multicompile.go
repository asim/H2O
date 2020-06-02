package handler

import (
	"fmt"

	"github.com/HailoOSS/protobuf/proto"

	multicompile "github.com/HailoOSS/config-service/proto/multicompile"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// MultiCompile is an equivalent of repeated executions of "Compile". Its goal is to save traffic.
// The contract of multiconfig is as follows: For every request, compile function is executed. If the received hash is identical to the received hash, empty config will be returned to indicate that no changes were made since the previous config.
func MultiCompile(req *server.Request) (proto.Message, errors.Error) {
	request := &multicompile.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.config.multicompile", fmt.Sprintf("%v", err))
	}

	compileResponses := make([]*multicompile.Response_CompileResponse, len(request.GetCompileRequests()))
	for i, compileRequest := range request.GetCompileRequests() {
		cfg, hash, err := DoCompile(compileRequest.GetId(), compileRequest.GetPath())
		if err != nil {
			compileResponses[i] = &multicompile.Response_CompileResponse{
				Config: proto.String(""),
				Hash:   proto.String(""),
				Error:  proto.Bool(true),
			}
		} else {
			previousHash := compileRequest.GetPreviousHash()
			if previousHash != "" && previousHash == hash {
				cfg = ""
			}
			compileResponses[i] = &multicompile.Response_CompileResponse{
				Config: proto.String(cfg),
				Hash:   proto.String(hash),
				Error:  proto.Bool(false),
			}
		}
	}

	return &multicompile.Response{
		CompileResponses: compileResponses,
	}, nil
}
