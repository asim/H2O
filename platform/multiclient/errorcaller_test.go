package multiclient

import (
	"testing"

	"github.com/hailo-platform/H2O/platform/errors"
)

func TestErrorCallerWithNil(t *testing.T) {
	caller := ErrorCaller(nil)

	e := caller(nil, nil)
	if e == nil {
		t.Fatalf("ErrorCaller should have returned an error")
	}
	if e.Type() != errors.ErrorNotFound {
		t.Errorf("Default error expected to be NotFound")
	}
}

func TestErrorCaller(t *testing.T) {
	err := errors.Forbidden("foo.bar.baz", "Oh noes!")
	caller := ErrorCaller(err)

	e := caller(nil, nil)
	if e == nil {
		t.Fatalf("ErrorCaller should have returned the error we gave it")
	}
	if e.Code() != "foo.bar.baz" {
		t.Errorf("Error code should be foo.bar.baz")
	}
}
