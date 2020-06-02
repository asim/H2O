package dao

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/gossie/src/gossie"
)

// Test unmarshalling a session using the legacy RoleCollection format
func TestUnmarshalSession_LegacyRoleCollection(t *testing.T) {
	storedSess := `{"key":"9Bf3155ejPvwuzhU33YUmYOAz7tVI1dv+h+vBCpWivvV68Sz8PVMH3OaeM\/+h3RM","createdTimestamp":1378377732,"token":{"createdTimestamp":1378377733,"authMechanism":"admin","deviceType":"cli","id":"dave","expiryTimestamp":1378406533,"autoRenewTimestamp":null,"roleCollection":{"BADMIN":"BADMIN","ADMIN":"ADMIN"},"signature":"IFvyIAtFhMu9Gkh\/cf\/2JtwZe8+Kk2jf4xSkAHuaJwtmlnvrIBAVmAPytzKv2hFumd7gyP\/lPrGXutKAeBmUrWYd\/NeewKRdT\/vaAmNgtT9qi3QqeiloOfyieZjcruFnzMRc6XY4CJYHbXth2qUyF+gYFT0MIV3Dv6OUDNDOMymOuQ8z67rDMsyoKKKko5UY8oY+kURnBH0tkGy5IFLY0sH9LWldSJYXTzKWSQl3siwkCoCeCpOn9j4q2fx4vAitJ1+Vz078V1BgHDWkyx3AlM9QefRcypBOFZZ2x+fwSzNEqm37FU5DJyDA3vhCQn2Wn2DodJVr9NWRBSXPZ1\/0zQ=="}}`
	// make a pretend row
	row := &gossie.Row{
		Key: []byte("doesntmatterthisreally"),
		Columns: []*gossie.Column{
			{
				Name:  []byte("session"),
				Value: []byte(storedSess),
			},
		},
	}
	sess, err := unmarshalSession(row)
	assert.NoError(t, err, "Unexpected unmarshal error")

	assert.Equal(t, `9Bf3155ejPvwuzhU33YUmYOAz7tVI1dv+h+vBCpWivvV68Sz8PVMH3OaeM/+h3RM`, sess.Id)
	assert.True(t, sess.Created.Equal(time.Unix(1378377732, 0)))
	assert.True(t, sess.Token.Created.Equal(time.Unix(1378377733, 0)))
	assert.Equal(t, "admin", sess.Token.AuthMechanism)
	assert.Equal(t, "cli", sess.Token.DeviceType)
	assert.Equal(t, "dave", sess.Token.Id)
	assert.True(t, sess.Token.Expires.Equal(time.Unix(1378406533, 0)))
	assert.True(t, sess.Token.AutoRenew.IsZero())
	assert.Equal(t, 2, len(sess.Token.Roles))
	// Must be sorted lexicographically
	assert.Equal(t, []string{"ADMIN", "BADMIN"}, sess.Token.Roles)
}

// Test unmarshalling a session (using the standard RolePatterns format)
func TestUnmarshalSession(t *testing.T) {
	storedSess := `{"key":"9Bf3155ejPvwuzhU33YUmYOAz7tVI1dv+h+vBCpWivvV68Sz8PVMH3OaeM\/+h3RM","createdTimestamp":1378377732,"token":{"createdTimestamp":1378377733,"authMechanism":"admin","deviceType":"cli","id":"dave","expiryTimestamp":1378406533,"autoRenewTimestamp":null,"rolePatterns":["ADMIN.**","FOO.BAR","BAZ"],"signature":"IFvyIAtFhMu9Gkh\/cf\/2JtwZe8+Kk2jf4xSkAHuaJwtmlnvrIBAVmAPytzKv2hFumd7gyP\/lPrGXutKAeBmUrWYd\/NeewKRdT\/vaAmNgtT9qi3QqeiloOfyieZjcruFnzMRc6XY4CJYHbXth2qUyF+gYFT0MIV3Dv6OUDNDOMymOuQ8z67rDMsyoKKKko5UY8oY+kURnBH0tkGy5IFLY0sH9LWldSJYXTzKWSQl3siwkCoCeCpOn9j4q2fx4vAitJ1+Vz078V1BgHDWkyx3AlM9QefRcypBOFZZ2x+fwSzNEqm37FU5DJyDA3vhCQn2Wn2DodJVr9NWRBSXPZ1\/0zQ=="}}`
	row := &gossie.Row{
		Key: []byte("doesntmatterthisreally"),
		Columns: []*gossie.Column{
			{
				Name:  []byte("session"),
				Value: []byte(storedSess),
			},
		},
	}
	sess, err := unmarshalSession(row)
	assert.NoError(t, err, "Unexpected unmarshal error")

	assert.Equal(t, `9Bf3155ejPvwuzhU33YUmYOAz7tVI1dv+h+vBCpWivvV68Sz8PVMH3OaeM/+h3RM`, sess.Id)
	assert.True(t, sess.Created.Equal(time.Unix(1378377732, 0)))
	assert.True(t, sess.Token.Created.Equal(time.Unix(1378377733, 0)))
	assert.Equal(t, "admin", sess.Token.AuthMechanism)
	assert.Equal(t, "cli", sess.Token.DeviceType)
	assert.Equal(t, "dave", sess.Token.Id)
	assert.True(t, sess.Token.Expires.Equal(time.Unix(1378406533, 0)))
	assert.True(t, sess.Token.AutoRenew.IsZero())
	assert.Equal(t, 3, len(sess.Token.Roles))
	// These roles must be in order
	assert.Equal(t, []string{"ADMIN.**", "FOO.BAR", "BAZ"}, sess.Token.Roles)
}

// Test that RolePatterns has precendence of RoleCollection, if both are specified
func TestUnmarshalSession_RolePatternsPrecedence(t *testing.T) {
	storedSess := `{"key":"9Bf3155ejPvwuzhU33YUmYOAz7tVI1dv+h+vBCpWivvV68Sz8PVMH3OaeM\/+h3RM","createdTimestamp":1378377732,"token":{"createdTimestamp":1378377733,"authMechanism":"admin","deviceType":"cli","id":"dave","expiryTimestamp":1378406533,"autoRenewTimestamp":null,"roleCollection":{"ADMIN":"ADMIN"},"rolePatterns":["ADMIN.**","FOO.BAR","BAZ"],"signature":"IFvyIAtFhMu9Gkh\/cf\/2JtwZe8+Kk2jf4xSkAHuaJwtmlnvrIBAVmAPytzKv2hFumd7gyP\/lPrGXutKAeBmUrWYd\/NeewKRdT\/vaAmNgtT9qi3QqeiloOfyieZjcruFnzMRc6XY4CJYHbXth2qUyF+gYFT0MIV3Dv6OUDNDOMymOuQ8z67rDMsyoKKKko5UY8oY+kURnBH0tkGy5IFLY0sH9LWldSJYXTzKWSQl3siwkCoCeCpOn9j4q2fx4vAitJ1+Vz078V1BgHDWkyx3AlM9QefRcypBOFZZ2x+fwSzNEqm37FU5DJyDA3vhCQn2Wn2DodJVr9NWRBSXPZ1\/0zQ=="}}`
	row := &gossie.Row{
		Key: []byte("doesntmatterthisreally"),
		Columns: []*gossie.Column{
			{
				Name:  []byte("session"),
				Value: []byte(storedSess),
			},
		},
	}
	sess, err := unmarshalSession(row)
	assert.NoError(t, err, "Unexpected unmarshal error")

	assert.Equal(t, `9Bf3155ejPvwuzhU33YUmYOAz7tVI1dv+h+vBCpWivvV68Sz8PVMH3OaeM/+h3RM`, sess.Id)
	assert.True(t, sess.Created.Equal(time.Unix(1378377732, 0)))
	assert.True(t, sess.Token.Created.Equal(time.Unix(1378377733, 0)))
	assert.Equal(t, "admin", sess.Token.AuthMechanism)
	assert.Equal(t, "cli", sess.Token.DeviceType)
	assert.Equal(t, "dave", sess.Token.Id)
	assert.True(t, sess.Token.Expires.Equal(time.Unix(1378406533, 0)))
	assert.True(t, sess.Token.AutoRenew.IsZero())
	assert.Equal(t, 3, len(sess.Token.Roles))
	// These roles must be in order
	assert.Equal(t, []string{"ADMIN.**", "FOO.BAR", "BAZ"}, sess.Token.Roles)
}

// @fixes HTWO-319
func TestUnmarshalH1NoRolesSession(t *testing.T) {
	storedSess := `{"key":"jVGoMlJ5Z7\/nSPyFSMP7fQuLsog3Uu12Dd8i9WRUPXFji8u\/jOHznkmPJTGQKDS0","createdTimestamp":1378911447,"token":{"createdTimestamp":1378911447,"authMechanism":"admin","deviceType":"hshell","id":"norole","expiryTimestamp":1378940247,"autoRenewTimestamp":1378939647,"roleCollection":[],"signature":"nYi5TmXnbtTk1ahI8stqZS0OqmHsB7Vz3YR5CnhD8+qVvjaWd9FRr5DtG2IAE17YHNpae2\/VjgDt4nWEvPExrtwhDwPfVtwZ0x1ForHahwCOeT+029\/Wrb55dY8OWFJyT10nc9\/Nh\/09AbhQ2PSERhdlR0aXPvEhUW9D8yXKG+Y="}}`
	// make a pretend row
	row := &gossie.Row{
		Key: []byte("doesntmatterthisreally"),
		Columns: []*gossie.Column{
			{
				Name:  []byte("session"),
				Value: []byte(storedSess),
			},
		},
	}
	_, err := unmarshalSession(row)
	assert.NoError(t, err, "Unexpected unmarshal error")
}

func TestDeterministicUnmarshaling(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	storedSess := `{"key":"E8dZ8eoSSJXDgvzIQ0qTbyjM7W/ZM3auJEjTND5fQMPjKHBYcjxr0DETmpHXHOsYWwmxzgULeAD0PbuSDELrEO/bdlbpNb+P/GDSibH3PsAuDIsIvuxWZ/iLvLgjbhK047n7dp+5bQlzp6AMP/CGdx3alrB2v7dN5IEc82AgQXctJYZY7ft/efrj4N53NMKId9agyuUe/uAXOuDI4OrCow==","createdTimestamp":1394919846,"token":{"createdTimestamp":1394919846,"authMechanism":"h2.PASSENGER","deviceType":"cli","id":"290840068858810368","expiryTimestamp":1394948646,"autoRenewTimestamp":1394946846,"roleCollection":{"CUSTOMER":"CUSTOMER","H4BADMIN.0641085f-ee14-49de-b8e2-e330e4d76150":"H4BADMIN.0641085f-ee14-49de-b8e2-e330e4d76150","H4BADMIN.40ebd8c6-698e-4347-95af-1430ab2fb331.PENDING":"H4BADMIN.40ebd8c6-698e-4347-95af-1430ab2fb331.PENDING","H4BADMIN.5a706c69-a33b-4e8f-ba40-09ee760385e3.PENDING":"H4BADMIN.5a706c69-a33b-4e8f-ba40-09ee760385e3.PENDING","H4BADMIN.7d793409-19b6-43d8-a165-5cb94397ab01.PENDING":"H4BADMIN.7d793409-19b6-43d8-a165-5cb94397ab01.PENDING"},"signature":"gTQDvyhyVbBjiUDJKGaNwEAhx3gaDOw88OYKoWZnj+c5hx/CCfclWC1YxSvRT2WJE4AQbUvN+Afsn8MvKLfHbvJv8Rpuuq8XGkbFShNUc3Jkp/BWhU16o1H/kq4ensDSu/40k7zSzwzJ1u1E/y5WmNeWten+i4yT8fgUasAfmx+XwLwwjKlWZ9Hu8YUxxJ+vgoeYpTt44fXrPG2mK8h3HD0i5BJjvxW398eFFTmoxkxKLNrJC6iICa/wTeaT4T6rJfF5Gi7g9ML/+Jk9kePw51AOs6rF+fjrZ5P4xPstq7U/jOPYlV81K8PtlRA+jHlm4oqVQh5s1r1qi0AstGZK8g=="}}`
	expectedTok := `am=h2.PASSENGER:d=cli:id=290840068858810368:ct=1394919846:et=1394948646:rt=1394946846:r=CUSTOMER,H4BADMIN.0641085f-ee14-49de-b8e2-e330e4d76150,H4BADMIN.40ebd8c6-698e-4347-95af-1430ab2fb331.PENDING,H4BADMIN.5a706c69-a33b-4e8f-ba40-09ee760385e3.PENDING,H4BADMIN.7d793409-19b6-43d8-a165-5cb94397ab01.PENDING:sig=gTQDvyhyVbBjiUDJKGaNwEAhx3gaDOw88OYKoWZnj+c5hx/CCfclWC1YxSvRT2WJE4AQbUvN+Afsn8MvKLfHbvJv8Rpuuq8XGkbFShNUc3Jkp/BWhU16o1H/kq4ensDSu/40k7zSzwzJ1u1E/y5WmNeWten+i4yT8fgUasAfmx+XwLwwjKlWZ9Hu8YUxxJ+vgoeYpTt44fXrPG2mK8h3HD0i5BJjvxW398eFFTmoxkxKLNrJC6iICa/wTeaT4T6rJfF5Gi7g9ML/+Jk9kePw51AOs6rF+fjrZ5P4xPstq7U/jOPYlV81K8PtlRA+jHlm4oqVQh5s1r1qi0AstGZK8g==`

	// unmarshal this N times to ensure consistent
	matching := 0
	n := 10
	for i := 0; i < n; i++ {
		row := &gossie.Row{
			Key: []byte("doesntmatterthisreally"),
			Columns: []*gossie.Column{
				{
					Name:  []byte("session"),
					Value: []byte(storedSess),
				},
			},
		}
		sess, err := unmarshalSession(row)
		assert.NoError(t, err, "Unexpected unmarshal error")

		actual := sess.Token.String()
		if !assert.Equal(t, expectedTok, actual, "Unmarshaling session non-deterministic") {
			continue
		}
		matching++

		// marshal again
		newRow, err := marshalSession(sess)
		assert.NoError(t, err, "Cannot re-marshal session")

		// unmarshal again
		newSess, err := unmarshalSession(newRow)
		assert.NoError(t, err, "Unexpected unmarshal error")

		actual = newSess.Token.String()
		if !assert.Equal(t, expectedTok, actual, "Unmarshaling session non-deterministic") {
			continue
		}
		matching++
	}

	assert.Equal(t, 20, matching, "Did wrong number of test cases")
}

func TestDeterministicMarshaling(t *testing.T) {
	// start with a user as if just authed
	u := &domain.User{
		App:     domain.Application("PASSENGER"),
		Uid:     "290840068858810368",
		Ids:     []domain.Id{},
		Created: time.Unix(1394919846, 0),
		Roles: []string{
			"CUSTOMER",
			"H4BADMIN.0641085f-ee14-49de-b8e2-e330e4d76150",
			"H4BADMIN.40ebd8c6-698e-4347-95af-1430ab2fb331.PENDING",
			"H4BADMIN.7d793409-19b6-43d8-a165-5cb94397ab01.PENDING",
		},
	}

	// make a token
	token := &domain.Token{
		Created:       time.Now(),
		AuthMechanism: u.App.ToAuthMechanism(),
		DeviceType:    "cli",
		Id:            u.Uid,
		Expires:       time.Now().Add(time.Hour),
		Roles:         u.Roles,
	}
	sess := &domain.Session{
		Id:      "foobarbaz",
		Created: token.Created,
		Token:   *token,
	}

	for i := 0; i < 50; i++ {
		// marshal and unmarshal
		row, err := marshalSession(sess)
		assert.NoError(t, err, "Cannot marshal session")

		// unmarshal again
		newSess, err := unmarshalSession(row)
		assert.NoError(t, err, "Unexpected unmarshal error")
		actual := newSess.Token.String()
		assert.Equal(t, token.String(), actual, "Marshaling/unmarshaling session non-deterministic")
	}
}

// Go <1.3 had a stable iteration order for maps with less than 8 items, and we depended on this behaviour to generate
// token strings. Upgrading to Go 1.3 exposed this behaviour (it randomises iteration order), so we must have stable
// string ordering. See #PLAT-313.
func TestRegression_MarshalledTokenStability(t *testing.T) {
	u := &domain.User{
		App:     domain.Application("PASSENGER"),
		Uid:     "290840068858810368",
		Ids:     []domain.Id{},
		Created: time.Unix(1394919846, 0),
		Roles: []string{
			"CUSTOMER",
			"H4BADMIN.0641085f-ee14-49de-b8e2-e330e4d76150",
			"H4BADMIN.40ebd8c6-698e-4347-95af-1430ab2fb331.PENDING",
			"H4BADMIN.7d793409-19b6-43d8-a165-5cb94397ab01.PENDING",
			"AAATHISSHOULDNTBEFIRST",
		},
	}

	// Make a token
	token := &domain.Token{
		Created:       time.Date(2014, 8, 1, 11, 35, 00, 00, time.UTC),
		AuthMechanism: u.App.ToAuthMechanism(),
		DeviceType:    "cli",
		Id:            u.Uid,
		Expires:       time.Date(2014, 8, 1, 12, 35, 00, 00, time.UTC),
		Roles:         u.Roles,
	}
	expectedString := "am=h2.PASSENGER:" +
		"d=cli:" +
		"id=290840068858810368:" +
		"ct=1406892900:" +
		"et=1406896500:" +
		"rt=:" +
		"r=CUSTOMER,H4BADMIN.0641085f-ee14-49de-b8e2-e330e4d76150,H4BADMIN.40ebd8c6-698e-4347-95af-1430ab2fb331.PENDING,H4BADMIN.7d793409-19b6-43d8-a165-5cb94397ab01.PENDING,AAATHISSHOULDNTBEFIRST:" +
		"sig=Zm9vYmFyYmF6"
	token.Sign([]byte("foobarbaz"))
	assert.Equal(t, expectedString, token.String())
}
