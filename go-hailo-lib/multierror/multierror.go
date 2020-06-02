package multierror

import (
	"fmt"
)

type Errors interface {
	error
	Errors() []error
}

type MultiError []error

func New() *MultiError {
	return new(MultiError)
}

// Add an error to this multi error
func (m *MultiError) Add(e error) {
	*m = append(*m, e)
}

// AnyErrors returns true if we have any errors
func (m *MultiError) AnyErrors() bool {
	if m == nil {
		return false
	}

	return len(*m) > 0
}

// Count the number of errors
func (m *MultiError) Count() int {
	if m == nil {
		return 0
	}

	return len(*m)
}

// Error to satisfy the Go Error interface
func (m *MultiError) Error() (msg string) {
	switch m.Count() {
	case 0:
		msg = "No errors"
	case 1:
		e := *m
		msg = e[0].Error()
	case 2:
		e := *m
		msg = e[0].Error() + fmt.Sprintf(", and 1 more error")
	default:
		e := *m
		msg = e[0].Error() + fmt.Sprintf(", and %v more errors", len(*m)-1)
	}

	return
}

// VerboseError returns an error composed by all the errors we have in the list
func (m *MultiError) VerboseError() error {
	if m == nil {
		return nil
	}
	var errr string
	for i, e := range []error(*m) {
		errr = errr + fmt.Sprintf("[%d] %s\n", i+1, e.Error())
	}
	return fmt.Errorf(errr)
}

// Errors returns all collected errors if any
func (m *MultiError) Errors() []error {
	if m == nil {
		return []error{}
	}
	return []error(*m)
}
