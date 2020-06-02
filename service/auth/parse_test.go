package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test that the order of roles appearing in a token are preserved on load
func TestTokenRoleOrderPreservation(t *testing.T) {
	mockValidator()
	defer unmockValidator()

	sessId := "testSessionId"
	token := "am=h2.PASSENGER:" +
		"d=cli:" +
		"id=123:" +
		"ct=1406892900:" +
		"et=1406896500:" +
		"rt=:" +
		"r=CUSTOMER,H4BADMIN,AAAATHISSHOULDNOTBEFIRST,ADMIN.DRIVER.*,!ADMIN.DRIVER.LON,ADMIN.DRIVER.LON.ABC:" +
		"sig=MTIz"

	u, err := FromSessionToken(sessId, token)
	assert.NoError(t, err, "Unexpected error unmarshalling token")

	assert.Equal(t, []string{
		"CUSTOMER",
		"H4BADMIN",
		"AAAATHISSHOULDNOTBEFIRST",
		"ADMIN.DRIVER.*",
		"!ADMIN.DRIVER.LON",
		"ADMIN.DRIVER.LON.ABC",
	}, u.Roles)
}
