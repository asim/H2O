package timeband

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TimeFromTime mints a new Time object (representing JUST the time component) from a golang time (representing a point in time)
func TimeFromTime(t time.Time) *Time {
	h, m, s := t.Clock()
	return &Time{
		h: int32(h),
		m: int32(m),
		s: int32(s),
	}
}

// UnmarshalJSON satisifies the JSON unmarshaler interface in order to interpret "HH:MM" or "HH:MM:SS" json
func (t *Time) UnmarshalJSON(b []byte) error {
	var err error

	var s string
	err = json.Unmarshal(b, &s)
	if err != nil {
		return fmt.Errorf("Time must be HH:MM or HH:MM:SS as JSON string")
	}

	p := strings.Split(s, ":")
	if len(p) < 2 || len(p) > 3 {
		return fmt.Errorf("Time must be HH:MM or HH:MM:SS")
	}
	t.h, err = extractInt32(p[0], 0, 23)
	if err != nil {
		return fmt.Errorf("Invalid hour: %v", err)
	}
	t.m, err = extractInt32(p[1], 0, 59)
	if err != nil {
		return fmt.Errorf("Invalid minute: %v", err)
	}
	if len(p) == 3 {
		t.s, err = extractInt32(p[2], 0, 59)
		if err != nil {
			return fmt.Errorf("Invalid second: %v", err)
		}
	}
	return nil
}

// Gte tests t is greater than or equal to u (same time or later than it)
func (t *Time) Gte(u *Time) bool {
	if t.h < u.h {
		return false
	}
	if t.h == u.h && t.m < u.m {
		return false
	}
	if t.h == u.h && t.m == u.m && t.s < u.s {
		return false
	}
	return true
}

// Lte tests t is less than or equal to u (same time or earlier than it)
func (t *Time) Lte(u *Time) bool {
	if t.h > u.h {
		return false
	}
	if t.h == u.h && t.m > u.m {
		return false
	}
	if t.h == u.h && t.m == u.m && t.s > u.s {
		return false
	}
	return true
}

// String satisifes Stringer
func (t *Time) String() string {
	return fmt.Sprintf("%02v:%02v:%02v", t.h, t.m, t.s)
}

// ---

// UnmarshalJSON satisifies the JSON unmarshaler interface in order to interpret "MON" (or "mon" or "Mon") to "SUN" json
func (d *Day) UnmarshalJSON(b []byte) error {
	var err error

	var s string
	err = json.Unmarshal(b, &s)
	if err != nil {
		return fmt.Errorf("Day must be MON through SUN as a string")
	}
	switch strings.ToUpper(s) {
	case "MON":
		*d = Day("MON")
	case "TUE":
		*d = Day("TUE")
	case "WED":
		*d = Day("WED")
	case "THU":
		*d = Day("THU")
	case "FRI":
		*d = Day("FRI")
	case "SAT":
		*d = Day("SAT")
	case "SUN":
		*d = Day("SUN")
	default:
		return fmt.Errorf("Day must be MON through SUN as a string")
	}
	return nil
}

// Weekday returns as time.Weekday
func (d Day) Weekday() time.Weekday {
	switch d {
	case Day("MON"):
		return time.Monday
	case Day("TUE"):
		return time.Tuesday
	case Day("WED"):
		return time.Wednesday
	case Day("THU"):
		return time.Thursday
	case Day("FRI"):
		return time.Friday
	case Day("SAT"):
		return time.Saturday
	case Day("SUN"):
		return time.Sunday
	}
	panic(fmt.Sprintf("Invalid day %v", d))
}

// String satisfies Stringer
func (d Day) String() string {
	return string(d)
}

// MatchesDow tests if the day-of-week of the supplied time falls on this day
func (d Day) MatchesDow(t time.Time) bool {
	return d.Weekday() == t.Weekday()
}

// ---

// MatchesDow tests if the day-of-week of the supplied time falls within any day
func (days Days) MatchesDow(t time.Time) bool {
	for _, d := range days {
		if d.MatchesDow(t) {
			return true
		}
	}
	return false
}

// ---

// Spans tests whether a day and time, as extracted from t, are within this band
// We are only checking here on day of week and time
func (tb *TimeBand) Spans(t time.Time) bool {
	if !tb.Days.MatchesDow(t) {
		return false
	}
	// extract time
	u := TimeFromTime(t)
	return tb.StartTime.Lte(u) && tb.EndTime.Gte(u)
}

// ---

// Spans tests whether a day and time, as extracted from t, are within ANY band
func (tbs TimeBands) Spans(t time.Time) bool {
	for _, tb := range tbs {
		if tb.Spans(t) {
			return true
		}
	}
	return false
}

// ---

// extractInt32 turns string to int32 and constrains within bounds
func extractInt32(s string, min, max int32) (int32, error) {
	i, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return min, err
	}
	i32 := int32(i)
	if i32 < min || i32 > max {
		return min, fmt.Errorf("Value %v is not within bounds %v < N < %v", s, min, max)
	}
	return i32, nil
}
