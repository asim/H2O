package circuitbreaker

import (
	"time"

	cb "github.com/andreas/circuitbreaker"
	"github.com/cenkalti/backoff"
	"github.com/facebookgo/clock"
)

var (
	defaultClock = clock.New() // Used for testing
)

type DefaultCircuit struct {
	disabled bool
	circuit  *cb.Breaker
}

func (r *DefaultCircuit) Open() bool {
	if r.disabled {
		return false
	}
	return !r.circuit.Ready()
}

func (r *DefaultCircuit) Result(err error) {
	if err != nil {
		r.circuit.Fail()
	} else {
		r.circuit.Success()
	}

}

type Options struct {
	Disabled bool `json:"disabled,omitempty"`

	// Rate threshold config
	Threshold  float64 `json:"threshold,omitempty"`
	MinSamples int64   `json:"minSamples,omitempty"`

	// Backoff config
	Multiplier          float64 `json:"multiplier,omitempty"`
	RandomizationFactor float64 `json:"randomizationFactor,omitempty"`
	InitialIntervalMs   int64   `json:"initialIntervalMs,omitempty"`
	MaxIntervalMs       int64   `json:"maxIntervalMs,omitempty"`
	MaxElapsedTimeMs    int64   `json:"maxElapsedTimeMs,omitempty"`
}

func NewDefaultCircuit(opts Options) *DefaultCircuit {
	b := &backoff.ExponentialBackOff{
		RandomizationFactor: opts.RandomizationFactor,
		Multiplier:          opts.Multiplier,
		InitialInterval:     time.Duration(opts.InitialIntervalMs) * time.Millisecond,
		MaxInterval:         time.Duration(opts.MaxIntervalMs) * time.Millisecond,
		MaxElapsedTime:      time.Duration(opts.MaxElapsedTimeMs) * time.Millisecond,
		Clock:               defaultClock,
	}
	b.Reset()

	circuit := cb.NewBreakerWithOptions(&cb.Options{
		BackOff:    b,
		Clock:      defaultClock,
		ShouldTrip: cb.RateTripFunc(opts.Threshold, opts.MinSamples),
	})

	return &DefaultCircuit{
		disabled: opts.Disabled,
		circuit:  circuit,
	}
}
