package graphite

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	log "github.com/cihub/seelog"
)

const graphiteLayout = "15:04 20060102"

type Query struct {
	from, to time.Time
	targets  []string
}

type graphiteResult struct {
	Target     string       `json:"target"`
	DataPoints [][]*float64 `json:"datapoints"`
}

// NewQuery mints a new Graphite query from 24 hours ago with no targets (need to be added)
func NewQuery() *Query {
	return &Query{targets: make([]string, 0)}
}

// Range zeros in on a specific start/end time
func (q *Query) Range(from, to time.Time) *Query {
	q.from, q.to = from, to
	return q
}

// From sets the from time, with an open-ended to
func (q *Query) From(from time.Time) *Query {
	q.from, q.to = from, time.Time{}
	return q
}

// AddTarget adds a single target metric (incl. Graphite function(s)) to the query
func (q *Query) AddTarget(t string) *Query {
	q.targets = append(q.targets, t)
	return q
}

// Execute hits Graphite and returns results
func (q *Query) Execute() (*Result, error) {
	v := url.Values{}
	v.Set("format", "json")
	if q.from.IsZero() {
		v.Set("from", "-1day")
	} else {
		v.Set("from", timeToGraphite(q.from))
	}
	if !q.to.IsZero() {
		v.Set("until", timeToGraphite(q.to))
	}
	for _, t := range q.targets {
		v.Add("target", t)
	}

	b, err := DefaultConnection.Query("/render/", v)
	if err != nil {
		return &Result{}, fmt.Errorf("Query execution error: %v", err)
	}

	r, err := unmarshalResult(b)
	return r, err
}

func timeToGraphite(t time.Time) string {
	return strings.Replace(t.Format(graphiteLayout), " ", "_", -1)
}

func unmarshalResult(b []byte) (*Result, error) {
	r := &Result{}
	var grRes []*graphiteResult = make([]*graphiteResult, 0)
	if err := json.Unmarshal(b, &grRes); err != nil {
		return r, fmt.Errorf("Query result unmarshaling error: %v", err)
	}

	// convert to the format we want (sane format)
	for _, gr := range grRes {
		tr := &Series{
			Name:       gr.Target,
			DataPoints: make([]DataPoint, len(gr.DataPoints)),
		}
		for i, dp := range gr.DataPoints {
			tr.DataPoints[i] = DataPoint{
				X: dp[1],
				Y: dp[0],
			}
		}
		r.results = append(r.results, tr)
		log.Debugf("Unmarshaled data series from graphite with target name '%v'", gr.Target)
	}
	return r, nil
}
