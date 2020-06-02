package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockScope(t *testing.T) {
	// make sure it implements Scope
	var s Scope = &MockScope{}

	assert.False(t, s.IsAuth())
	assert.False(t, s.HasAccess("ADMIN"))
}
