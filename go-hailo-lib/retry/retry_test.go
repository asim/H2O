package retry

import (
	"fmt"
	"testing"
	"time"
)

type testSleeper struct {
	totalSlept time.Duration
}

func (s *testSleeper) Sleep(d time.Duration) {
	s.totalSlept = s.totalSlept + d
}

func TestRetrierWithSuccess(t *testing.T) {
	attempsDone := uint16(0)

	ts := &testSleeper{}
	defaultSleeper = ts
	r := NewRetrier(BackoffConstant, 3, 1*time.Millisecond, 0, nil)

	f := func() error {
		attempsDone++
		return nil
	}
	err := r.Try(f)
	if err != nil {
		t.Fatal(err)
	}
	if attempsDone != 1 {
		t.Fatalf("Expected 1 attempt, got %d", attempsDone)
	}
	if ts.totalSlept != 0*time.Millisecond {
		t.Fatalf("Time slept should have been 1 ms, took %v", ts.totalSlept)
	}
}

func TestRetrierSucceedsEventually(t *testing.T) {
	attempsDone := uint16(0)

	ts := &testSleeper{}
	defaultSleeper = ts
	r := NewRetrier(BackoffConstant, 3, 1*time.Millisecond, 0, nil)

	// Fail first time, then succeed
	f := func() error {
		attempsDone++
		if attempsDone > 1 {
			return nil
		}
		return fmt.Errorf("Oops")
	}
	err := r.Try(f)
	if err != nil {
		t.Fatal(err)
	}
	if attempsDone != 2 {
		t.Fatalf("Expected 2 attempts, got %d", attempsDone)
	}
	if ts.totalSlept != 1*time.Millisecond {
		t.Fatalf("Time slept should been 1 ms, took %v", ts.totalSlept)
	}
}

func TestRetrierGivesUpEventually(t *testing.T) {
	attempsDone := uint16(0)

	ts := &testSleeper{}
	defaultSleeper = ts
	r := NewRetrier(BackoffConstant, 3, 1*time.Millisecond, 0, nil)

	f := func() error {
		attempsDone++
		return fmt.Errorf("Oops")
	}
	err := r.Try(f)
	if err == nil {
		t.Fatalf("Expected an error, got nil instead")
	}
	if attempsDone != 3 {
		t.Fatalf("Expected 3 attempts, got %d", attempsDone)
	}
	if ts.totalSlept != 2*time.Millisecond {
		t.Fatalf("Time slept should have been 2 ms, took %v", ts.totalSlept)
	}
}

func TestErrorHandlerCalled(t *testing.T) {
	handled := 0

	ts := &testSleeper{}
	defaultSleeper = ts
	r := NewRetrier(BackoffConstant, 3, 1*time.Millisecond, 0, func(err error) { handled++ })

	_ = r.Try(func() error { return fmt.Errorf("Oops") })
	if handled != 3 {
		t.Errorf("Expected error handler to be called 3 times. Got %d", handled)
	}
}

func TestRetrierBackoff(t *testing.T) {
	// Assume delay is 1 ms
	testCases := []struct {
		BackoffType BackoffType
		Attempts    uint
		MaxDelay    time.Duration
		TimeTaken   time.Duration
	}{
		// The last attempt won't wait since we have attempted the max number of times

		// 1ms + 1ms + 1ms + 0ms
		{BackoffType: BackoffConstant, Attempts: 4, MaxDelay: 0, TimeTaken: 3 * time.Millisecond},
		// 1ms + 2ms + 3ms + 0ms
		{BackoffType: BackoffLinear, Attempts: 4, MaxDelay: 0, TimeTaken: 6 * time.Millisecond},
		// 1ms + 2ms + 4ms + 0ms
		{BackoffType: BackoffExponential, Attempts: 4, MaxDelay: 0, TimeTaken: 7 * time.Millisecond},

		// With MaxDelay
		// 1ms + 1ms + 1ms + 0ms
		{BackoffType: BackoffConstant, Attempts: 4, MaxDelay: 2 * time.Millisecond, TimeTaken: 3 * time.Millisecond},
		// 1ms + 2ms + 2ms + 0ms
		{BackoffType: BackoffLinear, Attempts: 4, MaxDelay: 2 * time.Millisecond, TimeTaken: 5 * time.Millisecond},
		// 1ms + 2ms + 2ms + 0ms
		{BackoffType: BackoffExponential, Attempts: 4, MaxDelay: 2 * time.Millisecond, TimeTaken: 5 * time.Millisecond},
	}

	for i, tc := range testCases {
		attempsDone := uint(0)

		ts := &testSleeper{}
		defaultSleeper = ts
		r := NewRetrier(tc.BackoffType, tc.Attempts, 1*time.Millisecond, tc.MaxDelay, nil)

		f := func() error {
			attempsDone++
			return fmt.Errorf("Oops")
		}
		err := r.Try(f)
		if err == nil {
			t.Errorf("Expected an error, got nil instead (%d)", i)
			continue
		}
		if attempsDone != tc.Attempts {
			t.Errorf("Expected %d attempts, got %d (%d)", tc.Attempts, attempsDone, i)
			continue
		}
		if ts.totalSlept != tc.TimeTaken {
			t.Errorf("Time taken should have been %v, took %v (%d)", tc.TimeTaken, ts.totalSlept, i)
			continue
		}
	}
}
