package sync

import (
	"testing"
)

func TestConstructLockPath(t *testing.T) {
	// Cases where we don't expect an error
	validCases := []struct {
		namespace string
		id        string
		expected  string
	}{
		{namespace: "ns", id: "id", expected: "/ns/id"},
		{namespace: "ns", id: "some/id", expected: "/ns/some#id"},
	}

	errorCases := []struct {
		namespace string
		id        string
	}{
		{namespace: "/ns", id: "/id"},
		{namespace: "", id: "/id"},
		{namespace: "ns", id: ""},
	}

	for i, tc := range validCases {
		p, err := constructLockPath(tc.namespace, tc.id)
		if err != nil {
			t.Errorf("Error: %v (Case %d)", err, i)
			continue
		}
		if p != tc.expected {
			t.Errorf("Want: %s, Got %s", tc.expected, p)
		}
	}

	for i, tc := range errorCases {
		_, err := constructLockPath(tc.namespace, tc.id)
		if err == nil {
			t.Errorf("Expected an error, didn't get one (Case %d)", i)
		}
	}
}
