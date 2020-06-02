package cassandra

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAuthSuccess(t *testing.T) {
	successTestCases := map[string]map[string]*authenticationOptions{
		"": make(map[string]*authenticationOptions),
		`{"job":{"password":"secrets","username":"bob"}}`: {
			"job": {
				username: "bob",
				password: "secrets",
			},
		},
		`{"job":{"password":"secrets","username":"bob"},"zoning":{"password":"lesssecret","username":"zoner"}}`: {
			"job": {
				username: "bob",
				password: "secrets",
			},
			"zoning": {
				username: "zoner",
				password: "lesssecret",
			},
		},
	}

	for tc, expected := range successTestCases {
		a, err := parseAuth([]byte(tc))
		assert.Equal(t, a, expected, "they should be equal")
		assert.Nil(t, err, "err should be nil")
	}
}

func TestParseAuthFailure(t *testing.T) {
	failureTestCases := []string{
		"i love long walks on the beach",          // Not JSON
		`{"password":"secrets","username":"bob"}`, // This fails as its not a map[keyspace]map[string]string
	}

	// blank options
	expected := make(map[string]*authenticationOptions)

	for _, tc := range failureTestCases {
		a, err := parseAuth([]byte(tc))
		assert.Equal(t, a, expected, "they should be equal")
		assert.NotNil(t, err, "err should not be nil")
	}
}
