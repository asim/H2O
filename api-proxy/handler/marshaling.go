package handler

import (
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"strings"

	api "github.com/hailo-platform/H2O/api-proxy/proto/api"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/protobuf/proto"
)

const (
	hob       = "hob"
	city      = "city"
	sessionId = "session_id"
	apiToken  = "api_token"
)

var (
	passHeaders = map[string]bool{"Authorization": true}
)

func httpRequestToProto(r *http.Request) (*api.Request, errors.Error) {
	protoReq := &api.Request{
		Path:   proto.String(r.URL.Path),
		Verb:   proto.String(r.Method),
		Get:    make([]*api.Request_Pair, 0),
		Post:   make([]*api.Request_Pair, 0),
		Header: make([]string, 0),
	}

	// Not all clients send the correct mime type. Add it if it's missing
	var ct string
	if r.Method == "POST" || r.Method == "PUT" {
		var err error
		ct, _, err = mime.ParseMediaType(r.Header.Get("Content-Type"))
		if ct == "" || err != nil {
			ct = formEncodedMime
			r.Header.Set("Content-Type", ct)
		}
	}

	if r.Body != nil && ct != formEncodedMime {
		reqBytes, _ := ioutil.ReadAll(r.Body)
		protoReq.Body = proto.String(string(reqBytes))
	}

	// Get GET, POST or PUT parameters
	if r.Body != nil {
		err := r.ParseForm()
		if err != nil {
			return nil, errors.BadRequest(
				"11",
				fmt.Sprintf("Problem parsing HTTP request parameters: %v", err),
			)
		}
	}

	hobCode := ""

	for k, vs := range r.URL.Query() {
		if k == sessionId || k == apiToken {
			continue
		}

		switch k {
		case hob:
			hobCode = vs[0]
			continue
		case city:
			if len(vs[0]) == 3 && len(hobCode) != 3 {
				hobCode = vs[0]
			}
		}

		// pick first v
		protoReq.Get = append(protoReq.Get, &api.Request_Pair{
			Key:   proto.String(k),
			Value: proto.String(vs[0]),
		})
	}

	for k, vs := range r.PostForm {
		if k == sessionId || k == apiToken {
			continue
		}

		switch k {
		case hob:
			hobCode = vs[0]
			continue
		case city:
			if len(vs[0]) == 3 && len(hobCode) != 3 {
				hobCode = vs[0]
			}
		}

		// pick first v
		protoReq.Post = append(protoReq.Post, &api.Request_Pair{
			Key:   proto.String(k),
			Value: proto.String(vs[0]),
		})
	}

	// need to add HOB back in
	if len(hobCode) == 3 {
		protoReq.Get = append(protoReq.Get, &api.Request_Pair{
			Key:   proto.String(hob),
			Value: proto.String(hobCode),
		})

		protoReq.Post = append(protoReq.Post, &api.Request_Pair{
			Key:   proto.String(hob),
			Value: proto.String(hobCode),
		})
	}

	for k, v := range r.Header {
		if passHeaders[k] || strings.HasPrefix(k, "X-") { // only pass through custom headers
			vals := strings.Join(v, ",")
			protoReq.Header = append(protoReq.Header, fmt.Sprintf("%s: %s", k, vals))
		}
	}

	instrumentHob(r, hobCode)

	return protoReq, nil
}
