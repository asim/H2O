package localisation

import (
	"testing"
)

func TestExtractHobFromID(t *testing.T) {
	testCases := []struct {
		id  string
		hob string
	}{
		{"LON123", "LON"},
		{"LONDON123", "LON"},
		{"LO", ""},
	}

	for _, tc := range testCases {
		if tc.hob != ExtractHobFromID(tc.id) {
			t.Errorf(`Incorrect HOB %s extracted from id %s`, ExtractHobFromID(tc.id), tc.hob)
		}
	}
}
