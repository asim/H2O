package timeband

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUnmarshal(t *testing.T) {
	s := `{"startTime":"12:00:00","endTime":"13:00:00","days":["MON","TUE","THU"]}`
	tb := &TimeBand{}
	err := json.Unmarshal([]byte(s), tb)
	if err != nil {
		t.Errorf("Unmarshal failed at JSON parsing: %v", err)
	}
	if tb.StartTime.String() != "12:00:00" {
		t.Errorf("Unmarshal failed to yield expected StartTime 12:00:00 (got %v)", tb.StartTime.String())
	}
	if tb.EndTime.String() != "13:00:00" {
		t.Errorf("Unmarshal failed to yield expected EndTime 13:00:00 (got %v)", tb.EndTime.String())
	}
	if len(tb.Days) != 3 {
		t.Errorf("Unmarshal failed to yield expected 3 days (got %v)", len(tb.Days))
	}
	m := make(map[string]bool)
	for _, d := range tb.Days {
		m[string(d)] = true
	}
	expected := []string{"MON", "TUE", "THU"}
	for _, e := range expected {
		if !m[e] {
			t.Errorf("Unmarshal failed to yield expected day %v (missing)", e)
		}
	}
}

func TestSpans(t *testing.T) {
	s := `{"startTime":"12:00:00","endTime":"13:00:00","days":["MON","TUE","THU"]}`
	tb := &TimeBand{}
	err := json.Unmarshal([]byte(s), tb)
	if err != nil {
		t.Errorf("Unmarshal failed at JSON parsing: %v", err)
	}

	testCases := []struct {
		dateTime   string
		withinBand bool
	}{
		{"2013-11-06 12:01:00", false}, // wrong day - WED
		{"2013-11-05 12:01:00", true},  // right day - TUE
		{"2013-11-05 11:59:59", false}, // 1 min shy
		{"2013-11-05 12:00:00", true},  // *just* in band
		{"2013-11-05 13:00:00", true},  // *just* in band
		{"2013-11-05 13:00:01", false}, // *just* out of band
		{"2013-11-07 13:00:00", true},  // *just* in band on THU
		{"2013-11-04 13:00:00", true},  // *just* in band on MON
	}

	for _, tc := range testCases {
		u := parseTestDateTime(tc.dateTime)
		outcome := tb.Spans(u)
		if outcome != tc.withinBand {
			t.Errorf("Time %v does not match expected outcome %v when testing in band", u, tc.withinBand)
		}
	}
}

func parseTestDateTime(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		panic(err)
	}
	return t
}
