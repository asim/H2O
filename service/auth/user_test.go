package auth

import (
	"os"
	"testing"
	"time"

	log "github.com/cihub/seelog"
	glob "github.com/obeattie/ohmyglob"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Enable glob logging during tests
	glob.Logger, _ = log.LoggerFromWriterWithMinLevel(os.Stderr, log.TraceLvl)
}

func TestHasRole(t *testing.T) {
	testcases := []struct {
		expected bool
		role     string
		list     []string
	}{
		{true, "ADMIN", []string{"ADMIN"}},
		{true, "ADMIN", []string{"ADMIN", "DRIVER"}},
		{false, "admin", []string{"ADMIN", "DRIVER"}},
		{true, "DRIVER", []string{"ADMIN", "DRIVER"}},
		{true, "DRIVER", []string{"DRIVER", "ADMIN.DRIVER"}},
		{false, "ADMIN", []string{"ADMIN.DRIVER"}},
		{true, "ADMIN.DRIVER", []string{"ADMIN.DRIVER"}},
		{false, "ADMIN", []string{"DRIVER", "ADMIN.DRIVER"}},
		{false, "USER", []string{"DRIVER", "ADMIN.DRIVER"}},
		{false, "ADMIN", []string{"DRIVER.ADMIN"}},
		{false, "DRIVER", []string{"DRIVERADMIN"}},
		{false, "DRIVER", []string{"ADMINDRIVER"}},
		{true, "ADMIN.DRIVER", []string{"ADMIN.DRIVER"}},
		{false, "ADMIN.DRIVER", []string{"DRIVER"}},
		// Legacy wildcards (this is a horrifying edge-case we're allowing for backwards-compatibility)
		{true, "H4BADMIN.*", []string{"CUSTOMER", "H4BADMIN.123456"}},
		{false, "H4BADMIN.**", []string{"CUSTOMER", "H4BADMIN.123456"}}, // ** is bad
		{true, "H4BADMIN.*", []string{"CUSTOMER", "H4BADMIN"}},
		{false, "*", []string{"CUSTOMER", "H4BADMIN"}},         // * not allowed
		{false, "*.DRIVER", []string{"ADMIN.DRIVER"}},          // *.FOO not allowed
		{false, "*", []string{"CUSTOMER", "H4BADMIN"}},         // * not allowed
		{false, "*.DRIVER", []string{"ADMIN.DRIVER"}},          // *.FOO not allowed
		{false, "H4BADMIN.*.*", []string{"H4BADMIN.FOO.BAR"}},  // only one wildcard component was allowed
		{true, "H4BADMIN.*.*.*", []string{"H4BADMIN.*.*.BAZ"}}, // but match and actual * intervening component
		// New-style user globs
		{true, "ADMIN.DRIVER", []string{"ADMIN.*"}},
		{true, "ADMIN.*", []string{"ADMIN.*"}},
		{true, "*", []string{"*"}},
		// Fancier globsets
		{false, "ADMIN.DRIVER", []string{"ADMIN.**", "ADMIN.DRIVER", "!ADMIN.DRIVER"}},
		{true, "ADMIN.DRIVER.LON", []string{"CUSTOMER", "ADMIN.**.LON", "!ADMIN.DRIVER.*.LON"}},
		{true, "ADMIN.DRIVER.LON", []string{"ADMIN"}},
		{false, "ADMIN.DRIVER.MNC", []string{"ADMIN.DRIVER.MNC.FOO"}},
		{true, "ADMIN.DRIVER.MNC.FOO", []string{"ADMIN.DRIVER.???.**", "ADMIN.DRIVER.MNC.BAR"}},
		{false, "ADMIN.DRIVER.MNC.FOO", []string{"ADMIN.DRIVER.???.**", "ADMIN.DRIVER.MNC.BAR", "!ADMIN"}},
		// Check backwards compatibility (some of these essentially duplicate rules above, but it doesn't hurt to be
		// even more OCD about the behaviour here)
		{true, "ADMIN", []string{"ADMIN"}},
		{true, "ADMIN.DRIVER", []string{"ADMIN"}},
		{true, "ADMIN", []string{"ADMIN.**"}},
		{true, "ADMIN.DRIVER.LON", []string{"ADMIN.DRIVER"}},
		{true, "ADMIN.DRIVER.LON.FLEETY", []string{"ADMIN.DRIVER"}},
		{true, "ADMIN.DRIVER.LON.FLEETY", []string{"ADMIN.DRIVER.**"}},
		{true, "ADMIN.DRIVER.LON.FLEETY", []string{"ADMIN.DRIVER.LON"}},
		{true, "ADMIN.DRIVER.LON.FLEETY", []string{"ADMIN.DRIVER.LON.**"}},
		{true, "H4BADMIN.*", []string{"H4BADMIN.ORG"}},
		{true, "H4BADMIN.*", []string{"H4BADMIN"}},
		{false, "H4BADMIN.*", []string{"H4BADMIN.ORG.ABC"}},
		{true, "ADMIN.CRM.LON", []string{"ADMIN.CRM"}},
		{false, "ADMIN.CRM", []string{"ADMIN.CRM.LON"}},
		{true, "ADMIN.DRIVER.LON", []string{"ADMIN.DRIVER", "ADMIN.CRM", "ADMIN.CUSTOMER", "ADMIN.DEDICATED_DRIVER",
			"ADMIN.DRIVER.**", "ADMIN.CUSTOMER.**", "ADMIN.DEDICATED-DRIVER.**", "ADMIN.CRM.**"}},
		{true, "ADMIN.DRIVER", []string{"ADMIN.DRIVER", "ADMIN.CRM", "ADMIN.CUSTOMER", "ADMIN.DEDICATED_DRIVER",
			"ADMIN.DRIVER.**", "ADMIN.CUSTOMER.**", "ADMIN.DEDICATED-DRIVER.**", "ADMIN.CRM.**"}},
		{false, "ADMIN", []string{"ADMIN.DRIVER", "ADMIN.CRM", "ADMIN.CUSTOMER", "ADMIN.DEDICATED_DRIVER",
			"ADMIN.DRIVER.**", "ADMIN.CUSTOMER.**", "ADMIN.DEDICATED-DRIVER.**", "ADMIN.CRM.**"}},
	}

	user := User{}
	for _, testcase := range testcases {
		// Deliberately mutate Roles on an existing user so it will test the caching behaviour, too. If we just created
		// a new user each time, the caching wouldn't necessairly be tested
		user.Roles = testcase.list
		assert.Equal(t, testcase.expected, user.HasRole(testcase.role),
			"Unexpected result for role set %v to role %v", user.Roles, testcase.role)
	}
}

func TestCanAutoRenew(t *testing.T) {
	nonAutoRenewingUser := User{}
	if nonAutoRenewingUser.CanAutoRenew() {
		t.Errorf("Expected user to not auto renew")
	}

	autoRenewingUser := User{RenewTs: time.Now()}
	if !autoRenewingUser.CanAutoRenew() {
		t.Errorf("Expected user to auto renew")
	}
}
