package auth

import (
	"time"

	inst "github.com/hailo-platform/H2O/service/instrumentation"
)

func instTiming(key string, err error, t time.Time) {
	if err == nil {
		key += ".success"
	} else {
		key += ".failure"
	}
	inst.Timing(1.0, key, time.Since(t))
}
