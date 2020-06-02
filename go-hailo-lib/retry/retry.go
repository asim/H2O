// Package retry wraps retry logic around a a function call.
package retry

import (
	"time"
)

type sleeper interface {
	Sleep(time.Duration)
}

type standardSleeper struct{}

func (s standardSleeper) Sleep(d time.Duration) {
	time.Sleep(d)
}

var defaultSleeper sleeper = standardSleeper{}

type BackoffType uint8

const (
	// Delay does not increase between retries.
	BackoffConstant BackoffType = iota
	// Delay is increased by by itself after each attempt.
	BackoffLinear
	// Delay is doubled after each attempt.
	BackoffExponential
)

type ErrorHandler func(error)

// Retrier will try to call a function a number of times
// before giving up.
type Retrier struct {
	// Number of times to attempt the function call.
	attempts uint
	// Current delay between function calls
	delay time.Duration
	// Maximum time to delay between function calls
	maxDelay time.Duration
	// The backoff strategy to use.
	backoffType BackoffType
	// ErrorHandler is a function that will be called if any errors occur before
	// the Retrier gives up.
	// It is safe to set this to nil.
	errorHandler ErrorHandler
}

// NewSimpleRetrier will create a new Retrier instance with no maximum delay or error handler.
//
// backoffType: The backoff strategy to use.
//
// attempts: The number of times to attempt the function call
//
// delay: The delay to wait after an unsuccesful call.
func NewSimpleRetrier(backoffType BackoffType, attempts uint, delay time.Duration) Retrier {
	return NewRetrier(backoffType, attempts, delay, 0, nil)
}

// NewRetrier will create a new Retrier instance.
//
// The first three arguments are the same as NewSimpleRetrier.
//
// maxDelay: The maximum delay to wait after an unsuccesful call.
// Set it to 0 if you don't want a maximum.
//
// handler: a function that will be called if any errors occur before the Retrier gives up.
// It is safe to set this to nil.
func NewRetrier(backoffType BackoffType, attempts uint, delay time.Duration, maxDelay time.Duration, errorHandler ErrorHandler) Retrier {
	return Retrier{
		attempts:     attempts,
		delay:        delay,
		maxDelay:     maxDelay,
		backoffType:  backoffType,
		errorHandler: errorHandler,
	}
}

// Try will attempt to call f using the current Retrier settings.
// Since f does not take any arguments you may need to use a function literal
// to capture the arguments
//     r.Try(func() error { return someFunc(foo) })
func (r Retrier) Try(f func() error) error {
	delay := r.delay
	attempts := uint(0)

	for {
		err := f()
		attempts++
		if err == nil {
			return nil
		}
		if r.errorHandler != nil {
			r.errorHandler(err)
		}
		if attempts == r.attempts {
			return err
		}

		defaultSleeper.Sleep(delay)

		switch r.backoffType {
		case BackoffLinear:
			delay += r.delay
		case BackoffExponential:
			delay = delay * 2
		}

		if r.maxDelay > 0 && delay > r.maxDelay {
			delay = r.maxDelay
		}
	}
}
