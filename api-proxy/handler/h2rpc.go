package handler

import (
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/facebookgo/stack"
	h2error "github.com/hailo-platform/H2O/api-proxy/errors"
	"github.com/hailo-platform/H2O/api-proxy/session"
	"github.com/hailo-platform/H2O/api-proxy/stats"
	"github.com/hailo-platform/H2O/api-proxy/trace"
	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/service/auth"
	inst "github.com/hailo-platform/H2O/service/instrumentation"
)

const (
	protoMime           = "application/x-protobuf"
	formEncodedMime     = "application/x-www-form-urlencoded"
	defaultResponseMime = "application/json; charset=utf-8"
)

type caller func(req *client.Request, options ...client.Options) (*client.Response, errors.Error)

var (
	rpcCaller            caller            = client.CustomReq
	authScopeConstructor func() auth.Scope = auth.New
)

// rpcHandler handles inbound HTTP requests for H2 (RPC)
func rpcHandler(rw http.ResponseWriter, r *http.Request) {
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

	traceInfo := trace.Start(r)

	// sanity check basics - we should have a content type and requests shoudl be POSTed
	if r.Method != "POST" {
		err := &h2error.ApiError{
			ErrorType:        errors.ErrorBadRequest,
			ErrorCode:        "com.hailo-platform/H2O.api.rpc.postrequired",
			ErrorDescription: "Requests to the RPC endpoint must be POST-ed",
			ErrorContext:     []string{"15"},
			ErrorHttpCode:    http.StatusMethodNotAllowed,
			HttpHeaders: map[string]string{
				"Allow": "POST",
			},
			ErrorMultiStack: stack.CallersMulti(0),
		}
		h2error.Write(rw, err, defaultResponseMime, traceInfo)
		return
	}
	ct, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if ct == "" || err != nil {
		r.Header.Set("Content-Type", formEncodedMime)
	}

	// decide how to respond
	responseContentType := defaultResponseMime
	if r.Header.Get("Content-Type") == protoMime {
		responseContentType = protoMime
	}

	// ---

	service, request, perr := httpToH2Request(r)
	if perr != nil {
		h2error.Write(rw, perr, responseContentType, traceInfo)
		return
	}

	// test auth -- blanket block on com.hailo-platform/H2O.kernel.*
	if perr := authorisedFor(r, service); perr != nil {
		h2error.Write(rw, perr, responseContentType, traceInfo)
		return
	}

	// add scope of session and/or trace
	if traceInfo.TraceId != "" {
		request.SetTraceID(traceInfo.TraceId)
		request.SetTraceShouldPersist(traceInfo.PersistentTrace)
	}
	request.SetSessionID(session.SessionId(r))
	request.SetFrom("com.hailo-platform/H2O.hailo-2-api")
	request.SetRemoteAddr(r.RemoteAddr)

	rsp, perr := rpcCaller(request)
	if perr != nil {
		h2error.Write(rw, perr, responseContentType, traceInfo)
		return
	}

	// add any trace details to output
	trace.Write(rw, traceInfo)
	rw.Header().Set("Content-Type", responseContentType)
	rw.WriteHeader(200)
	rw.Write(rsp.Body())
	success = true
}

// httpToH2Request looks at the HTTP headers to determine what content type we are fed,
// and then constructs an appropriate H2 request
func httpToH2Request(r *http.Request) (service string, req *client.Request, perr errors.Error) {
	// extract request bytes, service name, endpoint name
	var (
		endpoint string
		reqBytes []byte
	)
	ct := r.Header.Get("Content-Type")
	switch ct {
	case protoMime: // raw bytes
		reqBytes, _ = ioutil.ReadAll(r.Body)
		service = r.URL.Query().Get("service")
		endpoint = r.URL.Query().Get("endpoint")
	default: // assume JSON is posted as a form param
		if err := r.ParseForm(); err != nil {
			perr = errors.BadRequest("com.hailo-platform/H2O.api.rpc.parseform", "Cannot parse form data.", "15")
			return
		}
		reqBytes = []byte(r.PostForm.Get("request"))
		if len(reqBytes) == 0 {
			reqBytes = []byte(`{}`)
		}
		service = r.PostForm.Get("service")
		endpoint = r.PostForm.Get("endpoint")
	}

	if service == "" {
		perr = errors.BadRequest("com.hailo-platform/H2O.api.rpc.missingservice", "Missing 'service' parameter.", "15")
		return
	}
	if endpoint == "" {
		perr = errors.BadRequest("com.hailo-platform/H2O.api.rpc.missingendpoint", "Missing 'endpoint' parameter.", "15")
		return
	}

	// mint client request now
	var reqErr error
	switch ct {
	case protoMime: // raw bytes
		req, reqErr = client.NewProtoRequest(service, endpoint, reqBytes)
	default: // assume JSON is posted as a form param
		req, reqErr = client.NewJsonRequest(service, endpoint, reqBytes)
	}

	if reqErr != nil {
		perr = errors.BadRequest("com.hailo-platform/H2O.api.rpc.badrequest", fmt.Sprintf("%v", reqErr))
		return
	}

	return
}

// authorisedFor checks if we are authorised to hit this service
func authorisedFor(r *http.Request, service string) errors.Error {
	// only bother if trying to hit kernel
	if !strings.HasPrefix(service, "com.hailo-platform/H2O.kernel.") {
		return nil
	}

	sessId := r.URL.Query().Get("session_id")
	if sessId == "" {
		sessId = r.Form.Get("session_id")
	}

	scope := auth.New()
	if sessId != "" {
		scope.RecoverSession(sessId)
	}

	if scope.IsAuth() && scope.AuthUser().HasRole("ADMIN") {
		return nil
	}

	return errors.Forbidden("com.hailo-platform/H2O.api.rpc.auth", "Permission denied.", "5")
}
