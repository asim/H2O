package cruftflake

import (
	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/server"
	cf "github.com/hailo-platform/H2O/idgen-service/proto/cruftflake"
)

// Mint a new Cruftflake ID from the IDGen Service
func Mint() (int64, error) {
	reqProto := &cf.Request{}
	req, err := server.ScopedRequest("com.hailocab.service.idgen", "cruftflake", reqProto)
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
