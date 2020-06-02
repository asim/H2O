package time

import (
	"time"

	"github.com/HailoOSS/monday"
)

// Format prints a localised string representing the given time=
func Format(t time.Time, locale string, location *time.Location) string {
	return monday.Format(t.In(location), "15:04, Monday 2 January", monday.Locale(locale))
}
