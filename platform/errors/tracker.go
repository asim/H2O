package errors

import (
	"strings"
	"sync"
	"time"
)

type counters map[string]int

type tracker struct {
	sync.RWMutex
	errors  map[string]counters
	cleared time.Time
}

var (
	defaultTracker = newTracker()
)

func counterName(context ...string) string {
	return strings.Join(context, ":")
}

func newTracker() *tracker {
	return &tracker{
		cleared: time.Now(),
		errors:  make(map[string]counters),
	}
}

// Clear clears the counters for an error
func Clear(code string, context ...string) {
	defaultTracker.clearCounters(code, context...)
}

// Count returns the count for an error
func Count(code string, context ...string) int {
	return defaultTracker.getCount(code, context...)
}

func Cleared() time.Time {
	return defaultTracker.getCleared()
}

// Get returns a counter for an error
func Get(code string, context ...string) counters {
	return defaultTracker.getCounters(code, context...)
}

// Track increments the count for an error
func Track(code string, context ...string) {
	defaultTracker.incrementCounter(code, context...)
}

func (t *tracker) clearCounters(code string, context ...string) {
	t.Lock()
	defer t.Unlock()

	t.cleared = time.Now()

	if len(context) > 0 {
		counter := counterName(context...)

		if _, ok := t.errors[code]; ok {
			t.errors[code][counter] = 0
		}

		return
	}

	for counter := range t.errors[code] {
		t.errors[code][counter] = 0
	}
}

func (t *tracker) getCleared() time.Time {
	t.RLock()
	defer t.RUnlock()
	return t.cleared
}

func (t *tracker) getCount(code string, context ...string) int {
	t.RLock()
	defer t.RUnlock()

	if len(context) > 0 {
		counter := counterName(context...)
		if _, ok := t.errors[code]; ok {
			return t.errors[code][counter]
		}
		return 0
	}

	count := 0

	for _, counter := range t.errors[code] {
		count += counter
	}

	return count
}

func (t *tracker) getCounters(code string, context ...string) counters {
	t.RLock()
	defer t.RUnlock()

	if len(context) > 0 {
		counts := make(counters)
		counter := counterName(context...)
		if _, ok := t.errors[code]; ok {
			counts[counter] = t.errors[code][counter]
		}
		return counts
	}

	return t.errors[code]
}

func (t *tracker) incrementCounter(code string, context ...string) {
	t.Lock()
	defer t.Unlock()

	counter := counterName(context...)

	if _, ok := t.errors[code]; !ok {
		t.errors[code] = make(counters)
	}

	t.errors[code][counter]++
}
