package server

import (
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/platform/client"
)

var defaultScoper = Scoper()

// Scoper mints something that is able to yield a scoped request for this server
func Scoper(req ...*Request) *serverScoper {
	scoper := &serverScoper{}
	if len(req) == 1 {
		scoper.parent = req[0]
	}

	return scoper
}

type serverScoper struct {
	parent *Request // Optional parent request
}

// ScopedRequest yields a new request from our server's scope (setting "from" details)
func (ss *serverScoper) ScopedRequest(service, endpoint string, payload proto.Message) (request *client.Request, err error) {
	if ss.parent != nil {
		request, err = ss.parent.ScopedRequest(service, endpoint, payload)

		// Remove any session attached to the request
		request.SetSessionID("")
		request.SetAuthorised(true)
	} else {
		request, err = ScopedRequest(service, endpoint, payload)
	}

	return
}

// Context returns some context to base error messages from, eg: server name
func (ss *serverScoper) Context() string {
	return Name
}
