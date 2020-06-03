package errors

import (
	"errors"
	"reflect"
	"testing"
)

func TestInternalServerError(t *testing.T) {
	err := InternalServerError("com.hailocab.test", "Something went wrong")

	if err.Type() != "INTERNAL_SERVER_ERROR" {
		t.Errorf("Wrong error type: %v", err.Type())
	}

	if err.Code() != "com.hailocab.test" {
		t.Errorf("Wrong error code: %v", err.Code())
	}

	if err.Description() != "Something went wrong" {
		t.Errorf("Wrong error description: %v", err.Description())
	}

	if err.HttpCode() != 500 {
		t.Errorf("Wrong http code: %v", err.HttpCode())
	}

	if err.Description() != err.Error() {
		t.Error("Description should be default error")
	}
}

func TestConversion(t *testing.T) {
	err := InternalServerError("com.hailocab.test", "Something went wrong")
	e := ToProtobuf(err)
	err2 := FromProtobuf(e)

	if err.Error() != err2.Error() {
		t.Errorf("Error() does not match: %#v vs %#v", err, err2)
	}

	if err.Code() != err2.Code() {
		t.Errorf("Code() does not match: %#v vs %#v", err, err2)
	}

	if err.Description() != err2.Description() {
		t.Errorf("Description() does not match: %#v vs %#v", err, err2)
	}

	if err.HttpCode() != err2.HttpCode() {
		t.Errorf("HttpCode() does not match: %#v vs %#v", err, err2)
	}

	if !reflect.DeepEqual(err.Context(), err2.Context()) {
		t.Errorf("Context() does not match: %#v vs %#v", err, err2)
	}
}

func TestErrTypeCheck(t *testing.T) {
	testCases := []struct {
		errCreator func(code string, errValue interface{}, context ...string) Error
		errChecker func(error) bool
	}{
		{
			errCreator: InternalServerError,
			errChecker: IsInternalServerError,
		},
		{
			errCreator: BadRequest,
			errChecker: IsBadRequest,
		},
		{
			errCreator: Forbidden,
			errChecker: IsForbidden,
		},
		{
			errCreator: BadResponse,
			errChecker: IsBadResponse,
		},
		{
			errCreator: Timeout,
			errChecker: IsTimeout,
		},
		{
			errCreator: NotFound,
			errChecker: IsNotFound,
		},
	}

	randomError := errors.New("Random")

	for i, tc := range testCases {
		if tc.errChecker(randomError) == true {
			t.Errorf("Should not pass for random error (%d)", i)
			continue
		}
		created := tc.errCreator("abc", "abc")
		if tc.errChecker(created) == false {
			t.Errorf("Should pass for correct error type (%d)", i)
		}
	}
}

func TestAddContext(t *testing.T) {
	e := InternalServerError("com.hailocab.test", "Something went wrong", "Some context").AddContext("1234")

	if l := len(e.Context()); l != 2 {
		t.Fatalf("Expected 2 parts in context, found %d", l)
	}

	if e.Context()[1] != "1234" {
		t.Fatalf("Expected error code 1234, found %s", e.Context()[1])
	}
}
