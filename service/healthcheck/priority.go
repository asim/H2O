//go:generate jsonenums -type=Priority
// Uses github.com/campoy/jsonenums

package healthcheck

// Priority defines the alert level for a healthcheck

type Priority int32

const (
	// 0 reserved for catastrophic failure. Not yet utilized.
	Pager   Priority = 1
	Email   Priority = 2
	Warning Priority = 3
)
