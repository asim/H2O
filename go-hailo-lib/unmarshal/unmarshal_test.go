package unmarshal

import (
	"testing"
	"time"
)

func TestString(t *testing.T) {
	tests := []struct {
		value    interface{}
		expected string
	}{
		{interface{}("1"), "1"},
		{interface{}(2), ""},
	}

	for _, v := range tests {
		s := String(v.value)

		if s != v.expected {
			t.Fatal("Expected ", v.expected, "got", s)
		}
	}
}

func TestTime(t *testing.T) {
	expected := time.Unix(1461680341, 0)
	tests := []struct {
		value    interface{}
		expected time.Time
	}{
		{interface{}("1461680341"), expected},
		{interface{}("asdasdasdasd"), time.Time{}},
		{interface{}(int64(1461680341)), expected},
		{interface{}(float64(1461680341)), expected},
	}

	for _, v := range tests {
		s := Time(v.value)

		if s != v.expected {
			t.Fatal("Expected ", v.expected, "got", s)
		}
	}
}

func TestBool(t *testing.T) {
	tests := []struct {
		value    interface{}
		expected bool
	}{
		{interface{}("0"), false},
		{interface{}("false"), false},
		{interface{}("true"), true},
		{interface{}(""), false},
		{interface{}(true), true},
		{interface{}(false), false},
	}

	for _, v := range tests {
		s := Bool(v.value)

		if s != v.expected {
			t.Fatal("Expected ", v.expected, "got", s)
		}
	}
}

func TestFloat64(t *testing.T) {
	tests := []struct {
		value    interface{}
		expected float64
	}{
		{interface{}("15.5"), float64(15.5)},
		{interface{}(float32(15.5)), float64(15.5)},
		{interface{}(float64(15.5)), float64(15.5)},
	}

	for _, v := range tests {
		s := Float64(v.value)

		if s != v.expected {
			t.Fatal("Expected ", v.expected, "got", s)
		}
	}
}

func TestInt64(t *testing.T) {
	tests := []struct {
		value    interface{}
		expected int64
	}{
		{interface{}("15"), int64(15)},
		{interface{}(float32(15.5)), int64(15)},
		{interface{}(float64(15.5)), int64(15)},
		{interface{}(int32(15)), int64(15)},
	}

	for _, v := range tests {
		s := Int64(v.value)

		if s != v.expected {
			t.Fatal("Expected ", v.expected, "got", s)
		}
	}
}

func TestInt32(t *testing.T) {
	tests := []struct {
		value    interface{}
		expected int32
	}{
		{interface{}("15"), int32(15)},
		{interface{}(float32(15.5)), int32(15)},
		{interface{}(float64(15.5)), int32(15)},
		{interface{}(int64(15)), int32(15)},
	}

	for _, v := range tests {
		s := Int32(v.value)

		if s != v.expected {
			t.Fatal("Expected ", v.expected, "got", s)
		}
	}
}
