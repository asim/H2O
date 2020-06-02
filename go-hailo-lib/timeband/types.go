package timeband

// Time represents a time in HH:MM:SS format
type Time struct {
	h, m, s int32
}

// Day represents a day of the week - MON through SUN
type Day string

// Days represents a number of days
type Days []Day

// TimeBand represents a time range, on a repeating number of days
type TimeBand struct {
	StartTime Time `json:"startTime" schema-type:"string"`
	EndTime   Time `json:"endTime" schema-type:"string"`
	Days      Days `json:"days" format:"table"`
}

// TimeBands represents a collection of time bands
type TimeBands []*TimeBand
