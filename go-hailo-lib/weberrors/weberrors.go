// Package weberrors aims to ease error reporting for webish service programming.
// The error type here implements the github.com/HailoOSS/platform/errors Errors interface
// If in doubt, ask author (John Dobronszki)
package weberrors

import (
	"github.com/facebookgo/stack"
	"github.com/HailoOSS/platform/errors"
)

const (
	BadRequest          = "BAD_REQUEST"
	InternalServerError = "INTERNAL_SERVER_ERROR"
)

// Error implements the errors.Error() interface.
type Error struct {
	c          map[string]string // Context, if the error is field validation related.
	d          string            // Description, if the error is related to something else.
	code       string
	multiStack *stack.Multi
}

func (e *Error) MultiStack() *stack.Multi {
	return e.multiStack
}

// New returns a new *Error, without containing any errors
// (ie. .HasErrors() returns false on a newly created one).
func New() *Error {
	return &Error{
		multiStack: stack.CallersMulti(2),
		c:          map[string]string{},
	}
}

// SetD sets the error description.
// Note that you should not to set the error description
// if you are failing validation. If you set the description, you will get
// back an internal server error (500) instead of a client error (400).
func (e *Error) SetD(s string) *Error {
	e.d = s
	return e
}

// SetC sets key and value of the context map.
func (e *Error) SetC(k, v string) *Error {
	e.c[k] = v
	return e
}

// UnsetC unsets a field in the error context.
func (e *Error) UnsetC(k string) *Error {
	delete(e.c, k)
	return e
}

// SetCode sets the error code. (eg. com.HailoOSS.service.organization.whatever)
func (e *Error) SetCode(c string) *Error {
	e.code = c
	return e
}

// HasErrors returns true if Error contains any error.
func (e *Error) HasErrors() bool {
	return len(e.d) > 0 || len(e.c) > 0
}

func (e *Error) isClientErr() bool {
	return len(e.d) == 0
}

// Context implements the errors.Error interface
func (e *Error) Context() []string {
	r := []string{}
	for k, v := range e.c {
		r = append(r, k)
		r = append(r, v)
	}
	return r
}

func (e *Error) AddContext(s ...string) errors.Error {
	for i := 0; i+1 < len(s); i += 2 {
		e.c[s[i]] = s[i+1]
	}
	return e
}

// HttpCode returns either 400 if it is a client bad request or 500 if it is a server error
// This will probably be improvde in the future
func (e *Error) HttpCode() uint32 {
	if e.isClientErr() {
		return 400 // Bad request.
	}
	return 500 // Internal server error.
}

// Code returns code from a Error struct
func (e *Error) Code() string {
	return e.code
}

// Type returns type from a Error struct
func (e *Error) Type() string {
	if e.isClientErr() {
		return BadRequest
	}
	return InternalServerError
}

// Error returns description from an Error struct
func (e *Error) Error() string {
	return e.Description()
}

// Description returns a web-friendly description or the actual description from the struct
func (e *Error) Description() string {
	if e.isClientErr() {
		return "Field validation failed, see context."
	}
	return e.d
}
