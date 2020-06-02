package handler

import (
	"fmt"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/multiclient"
	"github.com/HailoOSS/platform/server"
)

// throttlingSyncScoper is a pretty gross piece of code to get around an issue. Requests made to their API throttling
// service (rightly) require ADMIN-level priviledges, but we cannot grant an S2S auth rule for this purpose. Instead, we
// fake the service that the requests to it come from (which will be com.HailoOSS.api-proxy.throttlesync).
//
// @TODO: Figure out a better way of handling this that won't break if/when we implement cryptographic caller identity.
type throttleSyncScoper struct {
	scoperImpl multiclient.Scoper
}

func newthrottleSyncScoper() multiclient.Scoper {
	return &throttleSyncScoper{
		scoperImpl: server.Scoper(),
	}
}

func (ts *throttleSyncScoper) ScopedRequest(service, endpoint string, payload proto.Message) (*client.Request, error) {
	req, err := ts.scoperImpl.ScopedRequest(service, endpoint, payload)
	if err == nil {
		req.SetFrom(fmt.Sprintf("%s.throttlesync", req.From()))
	}
	return req, err
}

func (ts *throttleSyncScoper) Context() string {
	return fmt.Sprintf("%s.throttlesync", server.Name)
}
