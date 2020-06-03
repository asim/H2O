package multiclient

import (
	"github.com/facebookgo/stack"

	"github.com/hailo-platform/H2O/platform/errors"
)

type scopedErr struct {
	errorType   string
	code        string
	description string
	context     []string
	httpCode    uint32
	multiStack  *stack.Multi
}

func (e scopedErr) Description() string {
	return e.description
}

func (e scopedErr) Error() string {
	return e.Description()
}

func (e scopedErr) Type() string {
	return e.errorType
}

func (e scopedErr) Code() string {
	return e.code
}

func (e scopedErr) Context() []string {
	return e.context
}

func (e scopedErr) HttpCode() uint32 {
	return e.httpCode
}

func (e scopedErr) AddContext(s ...string) errors.Error {
	e.context = append(e.context, s...)
	return e
}

func (e scopedErr) MultiStack() *stack.Multi {
	return e.multiStack
}
