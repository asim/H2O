package handler

import (
	"net/http"

	"github.com/hailo-platform/H2O/api-proxy/controlplane"
	inst "github.com/hailo-platform/H2O/service/instrumentation"
)

var (
	throttlePayload = []byte(`{"status":false,"payload":"Throttled request","code":11}`)
)

// throttleHandler is responsible for shedding traffic, by returning standard response
func throttleHandler(rw http.ResponseWriter, r *http.Request, rule *controlplane.Rule) {
	// count hits
	inst.Counter(1.0, throttle, 1)

	if rule.Payload == nil {
		rw.WriteHeader(503)
		rw.Write(throttlePayload)
	} else {
		for k, v := range rule.Payload.Headers {
			rw.Header().Add(k, v)
		}
		rw.WriteHeader(rule.Payload.HttpStatus)
		rw.Write([]byte(rule.Payload.Body))
	}
}
