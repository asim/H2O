package graphite

import (
	"strings"
)

type Result struct {
	results []*Series
	cursor  int
	curr    *Series
}

type Series struct {
	cursor     int
	curr       *DataPoint
	Name       string
	DataPoints []DataPoint
}

type DataPoint struct {
	X, Y *float64
}

// NextItem can be called inside `for` and will populate the response to Current()
// Responds with true if a new Current() was found (hence you can call Current() to get it)
func (r *Result) Next() bool {
	if r.cursor < len(r.results) {
		r.curr = r.results[r.cursor]
	} else {
		r.curr = nil
	}
	r.cursor++
	return r.curr != nil
}

// Current returns the current item within the iterator
func (r *Result) Current() *Series {
	return r.curr
}

// Rewind takes the result set iterator back to the start
func (r *Result) Rewind() {
	r.curr = nil
	r.cursor = 0
}

// Count returns how many series we have within this results set
func (r *Result) Count() int {
	return len(r.results)
}

// Target attempts to find the result for a particular target, returning nil if not found
func (r *Result) Target(s string) *Series {
	for _, tr := range r.results {
		if tr.Name == s {
			return tr
		}
	}
	return nil
}

// TargetLike attempts to find the result for a particular target, matching where the supplied
// string is _anywhere_ within the target name - eg: partial match. Returns nil if not found
// This will always return the _first_ thing found that matches, where multiple matches exist
func (r *Result) TargetLike(s string) *Series {
	for _, tr := range r.results {
		if strings.Contains(tr.Name, s) {
			return tr
		}
	}
	return nil
}

// TargetN returns the Nth results series, 0 indexed, or nil if out of bounds
func (r *Result) TargetN(i int) *Series {
	if i < 0 || i >= len(r.results) {
		return nil
	}
	return r.results[i]
}

// Len returns how many data points in a series
func (tr *Series) Len() int {
	if tr == nil {
		return 0
	}
	return len(tr.DataPoints)
}

// Sum will aggregate all Y values from data points within the series
func (tr *Series) Sum() float64 {
	if tr == nil {
		return 0.0
	}
	var r float64
	for _, dp := range tr.DataPoints {
		if dp.Y == nil {
			continue
		}

		r += *dp.Y
	}
	return r
}

// Max finds the maximum Y value from the data points within the series
// Will return 0 if no datapoints
func (tr *Series) Max() float64 {
	if tr == nil {
		return 0.0
	}
	var r float64
	for _, dp := range tr.DataPoints {
		if dp.Y == nil {
			continue
		}

		if *dp.Y > r {
			r = *dp.Y
		}
	}
	return r
}

// Min finds the minimunm Y value from the data points within the series
// Will return 0 if no datapoints
func (tr *Series) Min() float64 {
	if tr == nil {
		return 0.0
	}
	var (
		r     float64
		first bool = true
	)
	for _, dp := range tr.DataPoints {
		if dp.Y == nil {
			continue
		}

		if first || *dp.Y < r {
			r = *dp.Y
			first = false
		}
	}
	return r
}

// NextItem can be called inside `for` and will populate the response to Current()
// Responds with true if a new Current() was found (hence you can call Current() to get it)
func (tr *Series) Next() bool {
	if tr == nil {
		return false
	}
	if tr.cursor < len(tr.DataPoints) {
		tr.curr = &tr.DataPoints[tr.cursor]
	} else {
		tr.curr = nil
	}
	tr.cursor++
	return tr.curr != nil
}

// Current returns the current data point within the iterator
func (tr *Series) Current() DataPoint {
	if tr == nil || tr.curr == nil {
		return DataPoint{}
	}
	return *tr.curr
}

// Rewind takes the result set iterator back to the start
func (tr *Series) Rewind() {
	if tr == nil {
		return
	}
	tr.curr = nil
	tr.cursor = 0
}
