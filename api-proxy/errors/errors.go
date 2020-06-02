package errors

import (
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/cihub/seelog"
	"github.com/facebookgo/stack"
	"github.com/HailoOSS/api-proxy/trace"
	perrors "github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/service/config"
	"github.com/HailoOSS/protobuf/proto"
)

const (
	profobufContentType = "application/x-protobuf"
	defaultContentType  = "application/json; charset=utf-8"
)

type ErrorBody struct {
	Status     bool     `json:"status"`
	Payload    string   `json:"payload"`
	Number     int      `json:"code"`
	DottedCode string   `json:"dotted_code"`
	Context    []string `json:"context"`
}

// ApiError implements the platform Error interface, but provides more support for HTTP-specific stuff
type ApiError struct {
	ErrorType        string
	ErrorCode        string
	ErrorDescription string
	ErrorContext     []string
	ErrorHttpCode    uint32
	HttpHeaders      map[string]string
	ErrorMultiStack  *stack.Multi
}

func (e *ApiError) Error() string {
	return e.Description()
}

func (e *ApiError) Type() string {
	return e.ErrorType
}

func (e *ApiError) Code() string {
	return e.ErrorCode
}

func (e *ApiError) Description() string {
	return e.ErrorDescription
}

func (e *ApiError) Context() []string {
	return e.ErrorContext
}

func (e *ApiError) HttpCode() uint32 {
	return e.ErrorHttpCode
}

func (e *ApiError) AddContext(s ...string) perrors.Error {
	e.ErrorContext = append(e.ErrorContext, s...)
	return e
}

func (e *ApiError) MultiStack() *stack.Multi {
	return e.ErrorMultiStack
}

// Write will write out a platform error in a standard way - appropriate for the specified mime type
func Write(rw http.ResponseWriter, err perrors.Error, mimeType string, traceInfo *trace.APITraceInfo) {
	if traceInfo != nil {
		// Add trace stuff
		trace.Write(rw, traceInfo)
	}

	// Convert the error to an ApiError (if it is not already)
	apiErr, ok := err.(*ApiError)
	if !ok {
		apiErr = &ApiError{
			ErrorType:        err.Type(),
			ErrorCode:        err.Code(),
			ErrorDescription: err.Description(),
			ErrorContext:     err.Context(),
			ErrorHttpCode:    err.HttpCode(),
			ErrorMultiStack:  stack.CallersMulti(0),
		}
	}

	// Sanitise error message if required
	if config.AtPath("hailo", "api", "sanitiseErrors").AsBool() {
		apiErr.ErrorDescription = err.Type()
	}

	switch mimeType {
	case profobufContentType:
		writeProtoError(rw, apiErr)
	default:
		writeJsonError(rw, apiErr)
	}
}

func writeProtoError(rw http.ResponseWriter, err *ApiError) {
	for k, v := range err.HttpHeaders {
		rw.Header().Set(k, v)
	}
	rw.Header().Set("Content-Type", profobufContentType)
	rw.WriteHeader(int(err.HttpCode()))
	b, _ := proto.Marshal(perrors.ToProtobuf(err))
	rw.Write(b)
}

func writeJsonError(rw http.ResponseWriter, err *ApiError) {
	for k, v := range err.HttpHeaders {
		rw.Header().Set(k, v)
	}
	rw.Header().Set("Content-Type", defaultContentType)

	errDescription := err.Description()
	if errDescription == "" {
		errDescription = "Internal low-level service failure, cannot complete request"
	}

	errNum := 0 // we'll attempt to replace this later on in this function
	errContext := err.Context()

	// see if the first entry in the context is a number and set errNum to it if it is
	if len(errContext) > 0 {
		if numFromContext, err := strconv.Atoi(errContext[0]); err == nil {
			errNum = numFromContext
			errContext = errContext[1:]
		}

	}

	// if we couldn't get the number from the context see if we can get it from err.Code()
	// once the clients have been updated this should no longer be necessary
	if errNum == 0 {
		errCode, convErr := strconv.Atoi(err.Code())
		if convErr == nil {
			errNum = errCode
		}
	}

	// log an error if we couldn't get an error number
	if errNum == 0 {
		errNum = 11
		log.Errorf("Couldn't get error number: %v", err)
	}

	e := ErrorBody{
		Status:     false,
		Payload:    errDescription,
		Number:     errNum,
		DottedCode: err.Code(),
		Context:    errContext,
	}

	b, marshalErr := json.Marshal(e)
	if marshalErr != nil {
		log.Warn("Error marshaling the error response into JSON: ", marshalErr)
	}

	rw.WriteHeader(int(err.HttpCode()))
	rw.Write(b)
}
