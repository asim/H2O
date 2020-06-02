package handler

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/api-proxy/controlplane"
	"github.com/HailoOSS/platform/util"
)

const (
	h1_success           = "handler.h1.success"
	h1_failure           = "handler.h1.failure"
	h1_azSuccessTemplate = "handler.per-az.%s.h1.success"
	h1_azFailureTemplate = "handler.per-az.%s.h1.failure"
	h2_success           = "handler.h2.success"
	h2_failure           = "handler.h2.failure"
	throttle             = "handler.throttle"
	deprecate            = "handler.deprecate"
	h2_azSuccessTemplate = "handler.per-az.%s.h2.success"
	h2_azFailureTemplate = "handler.per-az.%s.h2.failure"
)

var (
	az                         string
	h1_azSuccess, h1_azFailure string
	h2_azSuccess, h2_azFailure string
)

func init() {
	az, _ = util.GetAwsAZName()
	if len(az) == 0 {
		az = "unknown"
	}
	h1_azSuccess = fmt.Sprintf(h1_azSuccessTemplate, az)
	h1_azFailure = fmt.Sprintf(h1_azFailureTemplate, az)
	h2_azSuccess = fmt.Sprintf(h2_azSuccessTemplate, az)
	h2_azFailure = fmt.Sprintf(h2_azFailureTemplate, az)

	setupProxy()
}

// Determine whether the request should be "pinned" to a different hostname (with use of the X-H-ENDPOINT-foo headers),
// adding these headers if necessary.
func maybePinRequestToHostname(router controlplane.Router, rw http.ResponseWriter) {
	err, isCorrectHostname, urls, version := router.CorrectHostname(rw)
	if err == nil && !isCorrectHostname {
		pinRequestToHostname(rw, version, urls)
	}
}

// Send the X-H-ENDPOINT-foo headers
func pinRequestToHostname(rw http.ResponseWriter, version int64, urls controlplane.Urls) {
	rw.Header().Set("X-H-ENDPOINT-TIMESTAMP", fmt.Sprintf("%d", version))
	for k, v := range urls {
		rw.Header().Set("X-H-ENDPOINT-"+strings.ToUpper(strings.Replace(k, "_", "-", -1)), v)
	}
}

func regionPinning(router controlplane.Router, rw http.ResponseWriter) {
	// check if already set
	hob := rw.Header().Get("X-H-Hob")

	if len(hob) == 0 && rw.Header().Get("X-H-Endpoint-Timestamp") != "" {
		return
	}

	if hob != "" && rw.Header().Get("X-H-Pinning") == "1" {
		router.SetHob(hob)
		err, _, urls, version := router.CorrectHostname(rw)
		if err == nil {
			pinRequestToHostname(rw, version, urls)
		}
	}
}

// Handler will handle an HTTP request, deciding what to do with it
func Handler(srv *HailoServer) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		router := srv.Control.Router(r)
		route := router.Route()

		maybePinRequestToHostname(router, rw)

		if hobMode := router.GetHobMode(); len(hobMode) > 0 {
			rw.Header().Set("X-H-Mode", hobMode)
		}

		if route == nil {
			log.Tracef("[Handler] No route available; defaulting to H2")
			rw.Header().Set("X-Hailo-Route", controlplane.ActionSendToH2.String())
			h2Handler(rw, r, router)
			return
		}

		rw.Header().Set("X-Hailo-Route", route.Action.String())
		switch route.Action {
		case controlplane.ActionProxyToH1:
			log.Trace("[Handler] Matched H1 proxy route")
			h1Handler(rw, r)
		case controlplane.ActionThrottle:
			log.Trace("[Handler] Matched throttle route")
			throttleHandler(rw, r, route)
		case controlplane.ActionDeprecate:
			log.Trace("[Handler] Matched deprecate route")
			deprecateHandler(rw, r, route)
		case controlplane.ActionSendToH2:
			log.Trace("[Handler] Matched H2 route")
			h2Handler(rw, r, router)
		default:
			log.Errorf("[Handler] Unknown route action %v", route.Action)
			h2Handler(rw, r, router)
		}
	}
}

// RpcHandler will handle an HTTP request to /rpc (specifically H2 RPC)
func RpcHandler(srv *HailoServer) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		router := srv.Control.Router(r)
		maybePinRequestToHostname(router, rw)
		rpcHandler(rw, r)
	}
}
