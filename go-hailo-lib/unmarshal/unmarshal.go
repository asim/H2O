package unmarshal

import (
	"strconv"
	"time"
)

// String gets an interface type and tries to convert it to a string
// If it can't it returns empty string
func String(v interface{}) string {
	if str, ok := v.(string); ok {
		return str
	}
	return ""
}

// Time gets an interface type and tries to convert it to a time.Time.
// If it's capable of handling string, int64 and float64. Otherwise
// it would return default value of time.Time
func Time(v interface{}) time.Time {
	var ts int64
	if str, ok := v.(string); ok {
		var err error
		ts, err = strconv.ParseInt(str, 10, 64)
		if err != nil {
			return time.Time{}
		}
	} else if val, ok := v.(int64); ok {
		ts = val
	} else if val, ok := v.(float64); ok {
		ts = int64(val)
	} else {
		return time.Time{}
	}

	return time.Unix(ts, 0)
}

// Bool converts empty string, "0" and "false" to false, every other string to
// true. And if the v is acutally a bool, returns what it is.
func Bool(v interface{}) bool {
	var b bool
	// we will allow "1" as a bool and also a non-empty jobId
	if str, ok := v.(string); ok {
		if len(str) == 0 || str == "0" || str == "false" { // for "afh"
			b = false
		} else {
			b = true
		}
	} else if val, ok := v.(bool); ok {
		b = val
	}
	return b
}

// Float64 converts string, float32 to float64.
func Float64(v interface{}) float64 {
	switch k := v.(type) {
	case string:
		ret, _ := strconv.ParseFloat(k, 64)
		return ret
	case float32:
		return float64(k)
	case float64:
		return k
	}
	return 0
}

// Int64 converts strings, float32, float64 and int32 to int64
// There could be data loss if you would like to convert from float32 or float64
func Int64(v interface{}) int64 {
	switch k := v.(type) {
	case string:
		ret, _ := strconv.ParseInt(k, 10, 64)
		return ret
	case float32:
		return int64(k)
	case float64:
		return int64(k)
	case int32:
		return int64(k)
	}

	return 0
}

// Int32 converts strings, float32, float64 and int64 to int32
// There could be data loss if you would like to convert from float32, float64
// and int64
func Int32(v interface{}) int32 {
	switch k := v.(type) {
	case string:
		ret, _ := strconv.ParseInt(k, 10, 32)
		return int32(ret)
	case float32:
		return int32(k)
	case float64:
		return int32(k)
	case int64:
		return int32(k)
	}

	return 0
}
