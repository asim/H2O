package validate

import (
	"reflect"
	"testing"
	"time"
)

func TestHob(t *testing.T) {
	testCases := []struct {
		s     string
		valid bool
	}{
		{"LON", true},
		{"DUB", true},
		{"LO", false},
		{"LOND", false},
		{"lon", false},
		{" LON", false},
	}

	for _, tc := range testCases {
		err := Hob(reflect.ValueOf(tc.s))
		passes := err == nil
		if passes != tc.valid {
			t.Errorf("Expected %v to return %v", tc.s, tc.valid)
		}
	}
}

func TestHobId(t *testing.T) {
	testCases := []struct {
		s     string
		valid bool
	}{
		{"LON1234", true},
		{"DUB1234", true},
		{"LO", false},
		{"LOND", false},
		{"lon1234", false},
		{"LO1234", false},
		{"LONLON1234", false},
		{"AB=1234", false},
		{"", true},
	}

	for _, tc := range testCases {
		err := HobId(reflect.ValueOf(tc.s))
		passes := err == nil
		if passes != tc.valid {
			t.Errorf("Expected %v to return %v", tc.s, tc.valid)
		}
	}
}

func TestCurrencyCode(t *testing.T) {
	testCases := []struct {
		s     string
		valid bool
	}{
		{"USD", true},
		{"EUR", true},
		{"EURO", false},
		{"USD100", false},
		{"GBP", true},
		{" GBP", false},
		{"", false},
	}

	for _, tc := range testCases {
		err := CurrencyCode(reflect.ValueOf(tc.s))
		passes := err == nil
		if passes != tc.valid {
			t.Errorf("Expected %v to return %v", tc.s, tc.valid)
		}
	}
}

func TestNotEmptyString(t *testing.T) {
	testCases := []struct {
		s     string
		valid bool
	}{
		{"", false},
		{"   ", false},
		{"stuff", true},
	}

	for _, tc := range testCases {
		err := NotEmpty(reflect.ValueOf(tc.s))
		passes := err == nil
		if passes != tc.valid {
			t.Errorf("Expected %v to return %v", tc.s, tc.valid)
		}
	}
}

func TestNotEmptyTime(t *testing.T) {
	testCases := []struct {
		tm    time.Time
		valid bool
	}{
		{time.Now(), true},
		{time.Time{}, false},
	}

	for _, tc := range testCases {
		err := NotEmpty(reflect.ValueOf(tc.tm))
		passes := err == nil
		if passes != tc.valid {
			t.Errorf("Expected %v to return %v", tc.tm, tc.valid)
		}
	}
}

func TestNotEmptyPointer(t *testing.T) {
	testCases := []struct {
		p     interface{}
		valid bool
	}{
		{nil, false},
		{&testStruct{}, true},
	}

	for _, tc := range testCases {
		err := NotEmpty(reflect.ValueOf(tc.p))
		passes := err == nil
		if passes != tc.valid {
			t.Errorf("Expected %v to return %v", tc.p, tc.valid)
		}
	}
}

func TestNotEmptySlice(t *testing.T) {
	testCases := []struct {
		p     interface{}
		valid bool
	}{
		{[]int{0, 1, 2}, true},
		{nil, false},
		{[]int{}, false},
	}

	for _, tc := range testCases {
		err := NotEmpty(reflect.ValueOf(tc.p))
		passes := err == nil
		if passes != tc.valid {
			t.Errorf("Expected %v to return %v", tc.p, tc.valid)
		}
	}
}

func TestLongitude(t *testing.T) {
	testCases := []struct {
		l     interface{}
		valid bool
	}{
		{float32(0.003738), true},
		{float64(0.003738), true},
		{float64(-200), false},
		{float64(200), false},
		{float64(180), true},
		{float64(-180), true},
		{123, false},
	}

	for _, tc := range testCases {
		err := Longitude(reflect.ValueOf(tc.l))
		passes := err == nil
		if passes != tc.valid {
			t.Errorf("Expected %v to return %v", tc.l, tc.valid)
		}
	}
}

func TestLatitude(t *testing.T) {
	testCases := []struct {
		l     interface{}
		valid bool
	}{
		{float32(0.003738), true},
		{float64(0.003738), true},
		{float64(-91), false},
		{float64(105), false},
		{float64(90), true},
		{float64(-90), true},
		{80, false},
	}

	for _, tc := range testCases {
		err := Latitude(reflect.ValueOf(tc.l))
		passes := err == nil
		if passes != tc.valid {
			t.Errorf("Expected %v to return %v", tc.l, tc.valid)
		}
	}
}

func TestOneOf(t *testing.T) {
	testCases := []struct {
		v       interface{}
		allowed []interface{}
		valid   bool
	}{
		{5, []interface{}{1, 2, 3, 4, 5}, true},
		{2, []interface{}{1, 2, 3, 4, 5}, true},
		{8, []interface{}{1, 2, 3, 4, 5}, false},
		{"fred", []interface{}{"jane", "fred"}, true},
		{"kate", []interface{}{"jane", "fred"}, false},
		{9.6, []interface{}{5.4, 9.6}, true},
		{0.1, []interface{}{5.4, 9.6}, false},
		{true, []interface{}{true}, true},
		{false, []interface{}{true}, false},
		{"1", []interface{}{1, 2, 3}, false},
		{1, []interface{}{"1", 2, 3}, false},
		{"1", []interface{}{"1", 2, 3}, true},
	}

	for _, tc := range testCases {
		fn := OneOf(tc.allowed...)
		err := fn(reflect.ValueOf(tc.v))

		passes := err == nil
		if passes != tc.valid {
			t.Errorf("Expected %v to return %v", tc.v, tc.valid)
		}
	}
}

func TestNotOneOf(t *testing.T) {
	testCases := []struct {
		v          interface{}
		disallowed []interface{}
		valid      bool
	}{
		{5, []interface{}{1, 2, 3, 4, 5}, false},
		{2, []interface{}{1, 2, 3, 4, 5}, false},
		{8, []interface{}{1, 2, 3, 4, 5}, true},
		{"fred", []interface{}{"jane", "fred"}, false},
		{"kate", []interface{}{"jane", "fred"}, true},
		{9.6, []interface{}{5.4, 9.6}, false},
		{0.1, []interface{}{5.4, 9.6}, true},
		{true, []interface{}{true}, false},
		{false, []interface{}{true}, true},
		{"1", []interface{}{1, 2, 3}, true},
		{1, []interface{}{"1", 2, 3}, true},
		{"1", []interface{}{"1", 2, 3}, false},
	}

	for _, tc := range testCases {
		fn := NotOneOf(tc.disallowed...)
		err := fn(reflect.ValueOf(tc.v))

		passes := err == nil
		if passes != tc.valid {
			t.Errorf("Expected %v to return %v", tc.v, tc.valid)
		}
	}
}

func TestStringLength(t *testing.T) {
	testCases := []struct {
		v        interface{}
		min, max int
		valid    bool
	}{
		{string("FOO"), 0, 4, true},
		{string("FOOBA"), 0, 4, false},
		{string("世界世界"), 0, 4, true},
		{string("FOO"), 4, 4, false},
		{string("FOOB"), 4, 4, true},
		{float64(1.0), 0, 4, false},
		{bool(true), 0, 4, false},
		{string("世"), 1, 1, true},
	}

	for _, tc := range testCases {
		fn := StringLength(tc.min, tc.max)
		err := fn(reflect.ValueOf(tc.v))
		passes := err == nil
		if passes != tc.valid {
			t.Errorf("Expected %v to return %v", tc.v, tc.valid)
		}
	}
}
