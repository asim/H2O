package client

import (
	"testing"
	"time"
)

func TestGetBacksOff(t *testing.T) {
	timeout := NewTimeout(DefaultClient)
	timeout.add("foo", "bar")
	timeout.endpoints["foo"]["bar"] = time.Second

	testCases := []struct {
		attempt          int
		expectedDuration time.Duration
	}{
		{1, time.Second},
		{2, time.Second * 2},
		{3, time.Second * 3},
		{66, time.Second * 60}, // constrained by max
	}

	for _, tc := range testCases {
		d := timeout.Get("foo", "bar", tc.attempt)
		if d != tc.expectedDuration {
			t.Fatalf("Expecting %v duration, got %v", tc.expectedDuration, d)
		}
	}
}

func TestMsToDuration(t *testing.T) {
	testCases := []struct {
		ms       uint32
		duration time.Duration
	}{
		{1000, time.Second},
		{2000, time.Second * 2},
		{100, time.Millisecond * 100},
		{10, time.Millisecond * 10},
		{50000, time.Second * 50},
	}

	for _, tc := range testCases {
		d := msToDuration(tc.ms)
		if d != tc.duration {
			t.Fatalf("msToDuration - expecting %v, got %v", tc.duration, d)
		}
	}
}
