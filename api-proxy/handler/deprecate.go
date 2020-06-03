package handler

import (
	"net/http"

	"github.com/hailo-platform/H2O/api-proxy/controlplane"
	inst "github.com/hailo-platform/H2O/service/instrumentation"
)

var (
	deprecatePayload = []byte(`{"status":false,"payload":"Deprecated","code":11}`)
)

// deprecateHandler is responsible for shedding traffic, by returning standard response
func deprecateHandler(rw http.ResponseWriter, r *http.Request, rule *controlplane.Rule) {
	// count hits
	inst.Counter(1.0, deprecate, 1)

	if rule.Payload == nil {
		rw.WriteHeader(410)
		rw.Write(deprecatePayload)
	} else {
		for k, v := range rule.Payload.Headers {
			rw.Header().Add(k, v)
		}
		rw.WriteHeader(rule.Payload.HttpStatus)
		rw.Write([]byte(rule.Payload.Body))
	}
}
