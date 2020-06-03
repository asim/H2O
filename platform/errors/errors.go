package errors

import (
	"fmt"

	"github.com/facebookgo/stack"
	"github.com/hailo-platform/H2O/protobuf/proto"

	pe "github.com/hailo-platform/H2O/platform/proto/error"
)

const (
	ErrorInternalServer = "INTERNAL_SERVER_ERROR"
	ErrorBadRequest     = "BAD_REQUEST"
	ErrorForbidden      = "FORBIDDEN"
	ErrorBadResponse    = "BAD_RESPONSE"
	ErrorTimeout        = "TIMEOUT"
	ErrorNotFound       = "NOT_FOUND"
	ErrorConflict       = "CONFLICT"
	ErrorUnauthorized   = "UNAUTHORIZED"
	ErrorCircuitBroken  = "CIRCUIT_BROKEN"
)

// Error represents our customer error type
type Error interface {
	Error() string
	Type() string
	Code() string
	Description() string
	HttpCode() uint32
	Context() []string
	AddContext(...string) Error
	MultiStack() *stack.Multi
}

// LocalError is a type of error we build
type LocalError struct {
	errorType   string
	code        string
	description string
	context     []string
	httpCode    uint32
	multiStack  *stack.Multi
}

// Error representation is just the description
func (self LocalError) Error() string {
	return self.Description()
}

// Type returns the type of error message
func (self LocalError) Type() string {
	return self.errorType
}

// Code returns the error code, e.g. com.hailocab.service.something.went.wrong
func (self LocalError) Code() string {
	return self.code
}

// Description returns a human readable version of the error
func (self LocalError) Description() string {
	return self.description
}

// Context contains a list of strings with more information about the error
func (self LocalError) Context() []string {
	return self.context
}

// HttpCode returns the HTTP code we should be returning back via the API
func (self LocalError) HttpCode() uint32 {
	return self.httpCode
}

func (self LocalError) AddContext(s ...string) Error {
	self.context = append(self.context, s...)
	return self
}

// MultiStack identifies the locations this error was wrapped at.
func (self LocalError) MultiStack() *stack.Multi {
	return self.multiStack
}

// FromProtobuf takes a protobuf error and returns an Error as above
func FromProtobuf(err *pe.PlatformError) Error {
	return LocalError{
		errorType:   err.Type.String(),
		code:        *err.Code,
		description: *err.Description,
		context:     err.Context,
		httpCode:    *err.HttpCode,
	}
}

// ToProtobuf takes a Error and returns a protobuf error
func ToProtobuf(err Error) *pe.PlatformError {
	return &pe.PlatformError{
		Type:        pe.PlatformError_ErrorType(pe.PlatformError_ErrorType_value[err.Type()]).Enum(),
		Code:        proto.String(err.Code()),
		Description: proto.String(err.Description()),
		Context:     err.Context(),
		HttpCode:    proto.Uint32(err.HttpCode()),
	}
}

// InternalServerError message
func InternalServerError(code string, errValue interface{}, context ...string) Error {
	return LocalError{
		errorType:   ErrorInternalServer,
		code:        code,
		description: descriptionFromErrValue(errValue),
		context:     context,
		httpCode:    500,
		multiStack:  stackFromErrValue(errValue),
	}
}

func IsInternalServerError(err error) bool {
	return isErrorOfType(err, ErrorInternalServer)
}

// BadRequest error message
func BadRequest(code string, errValue interface{}, context ...string) Error {
	return LocalError{
		errorType:   ErrorBadRequest,
		code:        code,
		description: descriptionFromErrValue(errValue),
		context:     context,
		httpCode:    400,
		multiStack:  stackFromErrValue(errValue),
	}
}

func IsBadRequest(err error) bool {
	return isErrorOfType(err, ErrorBadRequest)
}

// Forbidden error message
func Forbidden(code string, errValue interface{}, context ...string) Error {
	return LocalError{
		errorType:   ErrorForbidden,
		code:        code,
		description: descriptionFromErrValue(errValue),
		context:     context,
		httpCode:    403,
		multiStack:  stackFromErrValue(errValue),
	}
}

func IsForbidden(err error) bool {
	return isErrorOfType(err, ErrorForbidden)
}

// Unauthorized error code
func Unauthorized(code string, errValue interface{}, context ...string) Error {
	return LocalError{
		errorType:   ErrorUnauthorized,
		code:        code,
		description: descriptionFromErrValue(errValue),
		context:     context,
		httpCode:    401,
		multiStack:  stackFromErrValue(errValue),
	}
}

func IsUnauthorized(err error) bool {
	return isErrorOfType(err, ErrorUnauthorized)
}

// BadResponse error message
func BadResponse(code string, errValue interface{}, context ...string) Error {
	return LocalError{
		errorType:   ErrorBadResponse,
		code:        code,
		description: descriptionFromErrValue(errValue),
		context:     context,
		httpCode:    500,
		multiStack:  stackFromErrValue(errValue),
	}
}

func IsBadResponse(err error) bool {
	return isErrorOfType(err, ErrorBadResponse)
}

// Timeout error message
func Timeout(code string, errValue interface{}, context ...string) Error {
	return LocalError{
		errorType:   ErrorTimeout,
		code:        code,
		description: descriptionFromErrValue(errValue),
		context:     context,
		httpCode:    504,
		multiStack:  stackFromErrValue(errValue),
	}
}

func IsTimeout(err error) bool {
	return isErrorOfType(err, ErrorTimeout)
}

// NotFound error message
func NotFound(code string, errValue interface{}, context ...string) Error {
	return LocalError{
		errorType:   ErrorNotFound,
		code:        code,
		description: descriptionFromErrValue(errValue),
		context:     context,
		httpCode:    404,
		multiStack:  stackFromErrValue(errValue),
	}
}

func IsNotFound(err error) bool {
	return isErrorOfType(err, ErrorNotFound)
}

// Conflict error message
func Conflict(code string, errValue interface{}, context ...string) Error {
	return LocalError{
		errorType:   ErrorConflict,
		code:        code,
		description: descriptionFromErrValue(errValue),
		context:     context,
		httpCode:    409,
		multiStack:  stackFromErrValue(errValue),
	}
}

func IsConflict(err error) bool {
	return isErrorOfType(err, ErrorConflict)
}

func CircuitBroken(code string, errValue interface{}, context ...string) Error {
	return LocalError{
		errorType:   ErrorCircuitBroken,
		code:        code,
		description: descriptionFromErrValue(errValue),
		context:     context,
		httpCode:    500,
		multiStack:  stackFromErrValue(errValue),
	}
}

func IsCircuitBroken(err error) bool {
	return isErrorOfType(err, ErrorCircuitBroken)
}

func isErrorOfType(err error, errorType string) bool {
	localError, ok := err.(LocalError)
	if !ok {
		return false
	}
	return localError.Type() == errorType
}

type stackErr interface {
	MultiStack() *stack.Multi
}

func stackFromErrValue(errValue interface{}) *stack.Multi {
	if stackErr, ok := errValue.(stackErr); ok {
		return stackErr.MultiStack()
	} else {
		// skip 2 stack frames such that top frame is the service handler
		return stack.CallersMulti(2)
	}
}

type errWrapper interface {
	Underlying() error
}

func descriptionFromErrValue(errValue interface{}) string {
	switch err := errValue.(type) {
	case errWrapper:
		return err.Underlying().Error()
	case error:
		return err.Error()
	case string:
		return err
	default:
		return fmt.Sprintf("%v", errValue)
	}
}
