package multiclient

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/protobuf/proto"
	"github.com/mreiferson/go-httpclient"

	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/errors"

	protoerror "github.com/hailo-platform/H2O/platform/proto/error"
)

const (
	protoContentType       = "application/x-protobuf"
	jsonContentType        = "application/json"
	formEncodedContentType = "application/x-www-form-urlencoded"
)

type errorBody struct {
	Status     bool     `json:"status"`
	Payload    string   `json:"payload"`
	Number     int      `json:"code"`
	DottedCode string   `json:"dotted_code"`
	Context    []string `json:"context"`
}

type Options struct {
	BaseUrl               string
	TlsSkipVerify         bool
	ConnectTimeout        time.Duration
	RequestTimeout        time.Duration
	ResponseHeaderTimeout time.Duration
}

// HttpCaller returns a caller that hits a "thin API" by HTTP, where baseUrl is
// provided and is something like 'https://api-staging.elasticride.com'
func HttpCaller(baseUrl string) Caller {
	return ConfiguredHttpCaller(Options{BaseUrl: baseUrl})
}

// ConfiguredHttpCaller with more explicit configuration options than simple HttpCaller
func ConfiguredHttpCaller(opts Options) Caller {
	tp := &httpclient.Transport{
		ConnectTimeout:        durationOrDefault(opts.ConnectTimeout, 5*time.Second),
		RequestTimeout:        durationOrDefault(opts.RequestTimeout, 5*time.Second),
		ResponseHeaderTimeout: durationOrDefault(opts.ResponseHeaderTimeout, 5*time.Second),
	}
	if opts.TlsSkipVerify {
		tp.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	httpClient := &http.Client{Transport: tp}

	return func(req *client.Request, rsp proto.Message) errors.Error {
		u, err := url.Parse(opts.BaseUrl)

		q := u.Query()
		q.Set("session_id", req.SessionID())
		q.Set("service", req.Service())
		q.Set("endpoint", req.Endpoint())
		u.Path = "/rpc"
		u.RawQuery = q.Encode()

		var httpReq *http.Request

		// send JSON req content-type to thin API as form-encoded data
		// send proto req content-type directly as bytes, with proto content type
		if req.ContentType() == jsonContentType {
			values := make(url.Values)
			values.Set("service", req.Service())
			values.Set("endpoint", req.Endpoint())
			values.Set("request", string(req.Payload()))
			httpReq, _ = http.NewRequest("POST", u.String(), bytes.NewReader([]byte(values.Encode())))
			httpReq.Header.Set("Content-Type", formEncodedContentType)
		} else {
			httpReq, _ = http.NewRequest("POST", u.String(), bytes.NewReader(req.Payload()))
			httpReq.Header.Set("Content-Type", protoContentType)
		}

		log.Tracef("[Multiclient] HTTP caller - calling '%s' : content-type '%s'", u.String(), req.ContentType())

		httpRsp, err := httpClient.Do(httpReq)
		if err != nil {
			log.Warnf("[Multiclient] HTTP caller error calling %s.%s via %s : %s", req.Service(), req.Endpoint(), u.String(), err)
			return errors.InternalServerError("multiclienthttp.postform", fmt.Sprintf("Error calling %s.%s via %s : %s", req.Service(), req.Endpoint(), u.String(), err))
		}

		defer httpRsp.Body.Close()
		rspBody, err := ioutil.ReadAll(httpRsp.Body)
		if err != nil {
			return errors.BadResponse("multiclienthttp.readresponse", fmt.Sprintf("Error reading response bytes: %v", err))
		}

		// what status code?
		if httpRsp.StatusCode != 200 {
			// deal with error
			e := &protoerror.PlatformError{}
			var err error
			if req.ContentType() == jsonContentType {
				jsonErr := &errorBody{}
				err = json.Unmarshal(rspBody, jsonErr)
				e.Code = proto.String(jsonErr.DottedCode)
				e.Context = jsonErr.Context
				e.Description = proto.String(jsonErr.Payload)
				e.HttpCode = proto.Uint32(uint32(httpRsp.StatusCode))
				// this conversion is lossy, since the JSON response for errors, as crafted
				// by the "thin API", does not currently include the error type, so we have
				// to guess from HTTP status code, but there is no distinct code for "BAD_RESPONSE"
				switch httpRsp.StatusCode {
				case 400:
					e.Type = protoerror.PlatformError_BAD_REQUEST.Enum()
				case 403:
					e.Type = protoerror.PlatformError_FORBIDDEN.Enum()
				case 404:
					e.Type = protoerror.PlatformError_NOT_FOUND.Enum()
				case 500:
					e.Type = protoerror.PlatformError_INTERNAL_SERVER_ERROR.Enum()
				case 504:
					e.Type = protoerror.PlatformError_TIMEOUT.Enum()
				}
			} else {
				err = proto.Unmarshal(rspBody, e)
			}
			// some issue understanding error rsp
			if err != nil {
				return errors.BadResponse("multiclienthttp.unmarshalerr", fmt.Sprintf("Error unmarshaling error response '%s': %v", string(rspBody), err))
			}
			return errors.FromProtobuf(e)
		}

		// unmarshal response
		if req.ContentType() == jsonContentType {
			err = json.Unmarshal(rspBody, rsp)
		} else {
			err = proto.Unmarshal(rspBody, rsp)
		}
		if err != nil {
			return errors.BadResponse("multiclienthttp.unmarshal", fmt.Sprintf("Error unmarshaling response: %v", err))
		}

		return nil
	}
}

func durationOrDefault(v, def time.Duration) time.Duration {
	if int64(v) == 0 {
		return def
	}
	return v
}
