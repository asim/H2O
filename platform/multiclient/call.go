package multiclient

import (
	"github.com/hailo-platform/H2O/protobuf/proto"
)

func Call(scope Scoper, service string, endpoint string, request proto.Message, response proto.Message) error {
	cl := New().DefaultScopeFrom(scope)
	cl.AddScopedReq(&ScopedReq{
		Service:  service,
		Endpoint: endpoint,
		Req:      request,
		Rsp:      response,
	})
	return cl.Execute().Succeeded("")
}
