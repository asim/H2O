package validate

import (
	"testing"
)

type testStruct struct {
	A string
	B string
}

func (t *testStruct) GetA() string {
	return t.A
}

func (t *testStruct) GetB() string {
	return t.B
}

type hobTestStruct struct {
	Code string
}

type anotherTestStruct struct {
	X *hobTestStruct
}

func TestEmptyField(t *testing.T) {
	validator := New().CheckField("A", NotEmpty)

	testCases := []struct {
		s          interface{}
		errorCount int
	}{
		{testStruct{"a", "a"}, 0},
		{testStruct{"", "a"}, 1},
		{&testStruct{"a", "a"}, 0},
		{&testStruct{"", "a"}, 1},
		{&testStruct{" ", "a"}, 1},
		{&testStruct{" a", "a"}, 0},
	}

	for _, tc := range testCases {
		errs := validator.Validate(tc.s)
		if errs.Count() != tc.errorCount {
			t.Errorf("Expected %v errors, got %v", tc.errorCount, errs.Count())
		}
	}

}

func TestStructSanity(t *testing.T) {
	v := &testStruct{"a", "b"}
	if v.GetA() != "a" {
		t.Errorf("Test struct sanity check (A) failed")
	}
}

func TestEmptyMethod(t *testing.T) {
	validator := New().CheckMethod("GetA", NotEmpty)

	testCases := []struct {
		s          interface{}
		errorCount int
	}{
		{&testStruct{"a", "a"}, 0},
		{&testStruct{"", "a"}, 1},
	}

	for _, tc := range testCases {
		errs := validator.Validate(tc.s)
		if errs.Count() != tc.errorCount {
			t.Errorf("Expected %v errors, got %v", tc.errorCount, errs.Count())
		}
	}
}

func TestChaining(t *testing.T) {
	v1 := New().CheckField("Code", Hob)
	v2 := New().CheckField("X", NotEmpty, Chain(v1))

	testCases := []struct {
		s          *anotherTestStruct
		errorCount int
	}{
		{s: &anotherTestStruct{X: &hobTestStruct{Code: "ABC"}}, errorCount: 0},
		{s: &anotherTestStruct{X: &hobTestStruct{Code: "ABCDEF"}}, errorCount: 1},
		{s: &anotherTestStruct{X: &hobTestStruct{}}, errorCount: 1},
	}

	for _, tc := range testCases {
		errs := v2.Validate(tc.s)
		if errs.Count() != tc.errorCount {
			t.Errorf("Expected %v errors, got %v (%v) for %v", tc.errorCount, errs.Count(), errs, tc.s.X)
		}
	}
}

func TestNoErrors(t *testing.T) {
	validator := New()

	errs := validator.Validate(&testStruct{})

	if errs.Count() != 0 {
		t.Errorf("Error count should be 0, got %d", errs.Count())
	}

	if errs.AnyErrors() {
		t.Error("There should not be any errors")
	}

	if errs != nil {
		t.Errorf("Error should be nil, got %+v", errs)
	}
}
