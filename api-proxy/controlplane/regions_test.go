package controlplane

import (
	"testing"
)

func TestIsOnline(t *testing.T) {
	testCases := []struct {
		status   string
		isOnline bool
	}{
		{"ONLINE", true},
		{"OFFLINE", false},
		{"OFLINE", true}, // misspelled - default to online
		{"", true},
	}

	for _, tc := range testCases {
		r := &Region{Status: tc.status}
		if r.IsOnline() != tc.isOnline {
			t.Errorf("Status %v expecting to be online=%v", tc.status, tc.isOnline)
		}
	}
}
