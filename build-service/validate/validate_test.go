package validate

import (
	"testing"
)

func TestBlank(t *testing.T) {
	type testStruct struct {
		a string `validate:"nonblank"`
		b string
	}

	testCases := []struct {
		s          interface{}
		errorCount int
	}{
		{testStruct{"a", "a"}, 0},
		{testStruct{"", "a"}, 1},
		{&testStruct{"a", "a"}, 0},
		{&testStruct{"", "a"}, 1},
	}

	for _, tc := range testCases {
		errors := Validate(tc.s)
		if len(errors) != tc.errorCount {
			t.Errorf("Expected %v errors, got %v", tc.errorCount, len(errors))
		}
	}

}
