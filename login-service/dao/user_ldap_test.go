package dao

import (
	"bytes"
	"github.com/HailoOSS/service/config"
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestMain(t *testing.T) {
	buf := bytes.NewBufferString(`{
		"hailo": {
			"ldap": {
				"domains": {
					"HailoOSS.com": true
				}
			}
		}
	}`)
	config.Load(buf)
}

func TestIsLDAPUserValid(t *testing.T) {
	uid := "john.smith@HailoOSS.com"

	username, domain, ok := IsLDAPUser("ADMIN", uid)

	assert.True(t, ok)
	assert.Equal(t, "john.smith", username)
	assert.Equal(t, "HailoOSS.com", domain)
}

func TestIsLDAPUserInvalidUID(t *testing.T) {
	uid := "john.smith@example.com"

	_, _, ok := IsLDAPUser("ADMIN", uid)

	assert.False(t, ok)

	uid = "john.smith"

	_, _, ok = IsLDAPUser("ADMIN", uid)

	assert.False(t, ok)
}

func TestIsLDAPUserInvalidApp(t *testing.T) {
	uid := "john.smith@example.com"

	_, _, ok := IsLDAPUser("CUSTOMER", uid)

	assert.False(t, ok)
}
