package cruftflake

import (
	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/server"
	cf "github.com/HailoOSS/idgen-service/proto/cruftflake"
)

// Mint a new Cruftflake ID from the IDGen Service
func Mint() (int64, error) {
	reqProto := &cf.Request{}
	req, err := server.ScopedRequest("com.HailoOSS.service.idgen", "cruftflake", reqProto)
	if err != nil {
		return 0, err
	}

	rsp := &cf.Response{}
	if err := client.Req(req, rsp); err != nil {
		return 0, err
	}

	id := rsp.GetId()
	return id, nil
}
