package handler

import (
	"net/http"
	"path"
	"strings"
	"time"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/api-proxy/controlplane"
	h2error "github.com/HailoOSS/api-proxy/errors"
	api "github.com/HailoOSS/api-proxy/proto/api"
	"github.com/HailoOSS/api-proxy/session"
	"github.com/HailoOSS/api-proxy/stats"
	"github.com/HailoOSS/api-proxy/trace"
	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/errors"
	inst "github.com/HailoOSS/service/instrumentation"
)

// h2Handler sends a request via H2, encoding the HTTP request as proto for an API-tier service
func h2Handler(rw http.ResponseWriter, r *http.Request, router controlplane.Router) {
	// map request -> proto, dispatch, map proto response -> http, respond
	start := time.Now()
	success := false

	defer func() {
		if !success {
			inst.Timing(1.0, h2_failure, time.Since(start))
			inst.Counter(1.0, h2_failure, 1)
			inst.Timing(1.0, h2_azFailure, time.Since(start))
			inst.Counter(1.0, h2_azFailure, 1)
		} else {
			inst.Timing(1.0, h2_success, time.Since(start))
			inst.Counter(1.0, h2_success, 1)
			inst.Timing(1.0, h2_azSuccess, time.Since(start))
			inst.Counter(1.0, h2_azSuccess, 1)
		}

		if r.URL.Path != "/" {
			stats.Record("/", success, time.Since(start))
		}

		stats.Record(r.URL.Path, success, time.Since(start))
	}()

	// trace this request?
	traceInfo := trace.Start(r)

	// map request to proto
	protoReq, perr := httpRequestToProto(r)
	if perr != nil {
		h2error.Write(rw, perr, "application/json", traceInfo)
		return
	}

	// dispatch request to api handler, inferred from path
	service, ep := pathToEndpoint(r.URL.Path)
	request, err := client.NewRequest(service, ep, protoReq)
	if err != nil {
		log.Debugf("Failed to translate to H2 request: %v", err)
		perr := errors.BadRequest(
			"15",
			"No handler available.",
		)
		h2error.Write(rw, perr, "application/json", traceInfo)
		return
	}

	// Add scope
	if traceInfo.TraceId != "" {
		request.SetTraceID(traceInfo.TraceId)
		request.SetTraceShouldPersist(traceInfo.PersistentTrace)
	}
	request.SetSessionID(session.SessionId(r))
	request.SetFrom("com.HailoOSS.hailo-2-api")
	request.SetRemoteAddr(r.RemoteAddr)

	rsp := &api.Response{}
	if perr := client.Req(request, rsp, client.Options{"retries": 0}); perr != nil {
		h2error.Write(rw, perr, "application/json", traceInfo)
		return
	}

	// add any trace details to output
	trace.Write(rw, traceInfo)

	for _, hdr := range rsp.GetHeader() {
		keyVal := strings.SplitN(hdr, ":", 2)
		// must be of form "key : value"
		if len(keyVal) == 2 {
			rw.Header().Add(
				strings.TrimSpace(keyVal[0]),
				strings.TrimSpace(keyVal[1]),
			)
		} else {
			log.Warnf("Malformed header %s", hdr)
		}
	}
	// default to json if no content-type set
	if rw.Header().Get("Content-Type") == "" {
		rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	// Check if we need to set the region pinning headers
	regionPinning(router, rw)

	rw.WriteHeader(int(rsp.GetStatusCode()))
	rw.Write([]byte(rsp.GetBody()))

	if rsp.GetStatusCode() < 500 {
		success = true
	}
}

func pathToEndpoint(p string) (service, endpoint string) {
	p = path.Clean(p)
	p = strings.TrimPrefix(p, "/")
	parts := strings.Split(p, "/")
	return "com.HailoOSS.api." + strings.Join(parts[:len(parts)-1], "."), parts[len(parts)-1]
}
