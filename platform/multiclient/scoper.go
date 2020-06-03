package multiclient

import (
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/platform/client"
)

// Scoper represents something that generate us a "scoped" request
// A "scoped" request includes some context, either taken from the "server" (who we are) or an inbound request
type Scoper interface {
	// Name returns the scope name, for example the server name, or scoping request name,
	// which can be used to give context to error messages etc.
	Context() string
	// ScopedRequest yields a new RPC request, preserving scope of trace/auth etc.
	ScopedRequest(service, endpoint string, payload proto.Message) (*client.Request, error)
}

// ExplicitScoper mints a Scoper that we can define explicitly things like SessionId and TraceId
func ExplicitScoper() *explicitScoper {
	return &explicitScoper{}
}

type explicitScoper struct {
	SessionId, TraceId string
	contextValue       string
}

// SetSessionId for this scoper
func (es *explicitScoper) SetSessionId(s string) *explicitScoper {
	es.SessionId = s
	return es
}

// SetTraceId for this scoper
func (es *explicitScoper) SetTraceId(t string) *explicitScoper {
	es.TraceId = t
	return es
}

// SetContext for this scoper
func (es *explicitScoper) SetContext(s string) *explicitScoper {
	es.contextValue = s
	return es
}

// ScopedRequest to satisfy Scoper
func (es *explicitScoper) ScopedRequest(service, endpoint string, payload proto.Message) (*client.Request, error) {
	req, err := client.NewRequest(service, endpoint, payload)
	if err != nil {
		return nil, err
	}
	req.SetSessionID(es.SessionId)
	req.SetTraceID(es.TraceId)
	return req, nil
}

// ScopedRequest to satisfy Scoper
func (es *explicitScoper) Context() string {
	return es.contextValue
}
