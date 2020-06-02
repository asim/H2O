package domain

import (
	"bytes"
	"runtime"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func init() {
	bcryptCost = bcrypt.MinCost
}

func mintToken() Token {
	return Token{
		Created:       time.Unix(1378377733, 0),
		AuthMechanism: "admin",
		DeviceType:    "cli",
		Id:            "dave",
		Expires:       time.Unix(1378406533, 0),
		AutoRenew:     time.Time{},
		Roles:         []string{"ADMIN"},
		Signature:     `IFvyIAtFhMu9Gkh/cf/2JtwZe8+Kk2jf4xSkAHuaJwtmlnvrIBAVmAPytzKv2hFumd7gyP/lPrGXutKAeBmUrWYd/NeewKRdT/vaAmNgtT9qi3QqeiloOfyieZjcruFnzMRc6XY4CJYHbXth2qUyF+gYFT0MIV3Dv6OUDNDOMymOuQ8z67rDMsyoKKKko5UY8oY+kURnBH0tkGy5IFLY0sH9LWldSJYXTzKWSQl3siwkCoCeCpOn9j4q2fx4vAitJ1+Vz078V1BgHDWkyx3AlM9QefRcypBOFZZ2x+fwSzNEqm37FU5DJyDA3vhCQn2Wn2DodJVr9NWRBSXPZ1/0zQ==`,
	}
}

func mintUser() *User {
	return &User{
		App:             Application("DRIVER"),
		Uid:             "LON1234",
		Ids:             []Id{},
		Created:         time.Now(),
		Roles:           []string{"DRIVER", "FOO", "BAR.BAZ"},
		PasswordHistory: make([][]byte, 0),
		PasswordChange:  time.Time{},
	}
}

func TestTokenString(t *testing.T) {
	token := mintToken()
	s := token.String()
	expected := `am=admin:d=cli:id=dave:ct=1378377733:et=1378406533:rt=:r=ADMIN:sig=IFvyIAtFhMu9Gkh/cf/2JtwZe8+Kk2jf4xSkAHuaJwtmlnvrIBAVmAPytzKv2hFumd7gyP/lPrGXutKAeBmUrWYd/NeewKRdT/vaAmNgtT9qi3QqeiloOfyieZjcruFnzMRc6XY4CJYHbXth2qUyF+gYFT0MIV3Dv6OUDNDOMymOuQ8z67rDMsyoKKKko5UY8oY+kURnBH0tkGy5IFLY0sH9LWldSJYXTzKWSQl3siwkCoCeCpOn9j4q2fx4vAitJ1+Vz078V1BgHDWkyx3AlM9QefRcypBOFZZ2x+fwSzNEqm37FU5DJyDA3vhCQn2Wn2DodJVr9NWRBSXPZ1/0zQ==`
	if s != expected {
		t.Fatalf("Unexpected token to string: %v", s)
	}
}

func TestTokenDataComponent(t *testing.T) {
	token := mintToken()
	s := token.dataComponent()
	expected := `am=admin:d=cli:id=dave:ct=1378377733:et=1378406533:rt=:r=ADMIN`
	if s != expected {
		t.Fatalf("Unexpected token dataComponent: %v", s)
	}
}

func TestTokenDataToSign(t *testing.T) {
	token := mintToken()
	s := token.DataToSign()
	expected := []byte(`am=admin:d=cli:id=dave:ct=1378377733:et=1378406533:rt=:r=ADMIN`)
	if !bytes.Equal(s, expected) {
		t.Fatalf("Unexpected token DataToSign: %v", s)
	}
}

func TestTokenCopy(t *testing.T) {
	t1 := mintToken()
	t2 := t1.Copy()

	// now let's change stuff
	t2.Created = time.Unix(1378377711, 0)
	t2.AuthMechanism = "notadmin"
	t2.DeviceType = "www"
	t2.Id = "bcb"
	t2.Expires = time.Unix(1378377799, 0)
	t2.AutoRenew = time.Unix(1378377788, 0)
	t2.Roles = append(t2.Roles, "FOO")
	t2.Signature = `IFvyIAtFhMu9Gkh/cf/2Jtw`

	// verify they're now different
	if t1.Created == t2.Created {
		t.Fatal("Expecting Created to be different")
	}
	if t1.AuthMechanism == t2.AuthMechanism {
		t.Fatal("Expecting AuthMechanism to be different")
	}
	if t1.DeviceType == t2.DeviceType {
		t.Fatal("Expecting DeviceType to be different")
	}
	if t1.Id == t2.Id {
		t.Fatal("Expecting Id to be different")
	}
	if t1.Expires == t2.Expires {
		t.Fatal("Expecting Expires to be different")
	}
	if t1.AutoRenew == t2.AutoRenew {
		t.Fatal("Expecting AutoRenew to be different")
	}
	if len(t1.Roles) == len(t2.Roles) {
		t.Fatal("Expecting Created to be different")
	}
	if t1.Signature == t2.Signature {
		t.Fatal("Expecting Created to be different")
	}
}

func TestTokenAutoRenew(t *testing.T) {
	tok := Token{
		Created:       time.Now().Add(-1 * time.Hour),
		AuthMechanism: "admin",
		DeviceType:    "cli",
		Id:            "dave",
		Expires:       time.Now().Add(-1 * time.Minute),
		AutoRenew:     time.Now().Add(-10 * time.Minute),
		Roles:         []string{},
		Signature:     `IFvyIAtFhMu9Gkh/cf/2JtwZe8+Kk2jf4xSkAHuaJwtmlnvrIBAVmAPytzKv2hFumd7gyP/lPrGXutKAeBmUrWYd/NeewKRdT/vaAmNgtT9qi3QqeiloOfyieZjcruFnzMRc6XY4CJYHbXth2qUyF+gYFT0MIV3Dv6OUDNDOMymOuQ8z67rDMsyoKKKko5UY8oY+kURnBH0tkGy5IFLY0sH9LWldSJYXTzKWSQl3siwkCoCeCpOn9j4q2fx4vAitJ1+Vz078V1BgHDWkyx3AlM9QefRcypBOFZZ2x+fwSzNEqm37FU5DJyDA3vhCQn2Wn2DodJVr9NWRBSXPZ1/0zQ==`,
	}
	if !tok.CanAutoRenew() {
		t.Fatal("Should be able to auto-renew this token")
	}
}

func TestGrantRoles(t *testing.T) {
	testCases := []struct {
		actOn   []string
		outcome []string
	}{
		{[]string{"DRIVER"}, []string{"DRIVER", "FOO", "BAR.BAZ"}},
		{[]string{"ADMIN"}, []string{"DRIVER", "FOO", "BAR.BAZ", "ADMIN"}},
		{[]string{"ADMIN", "ADMIN"}, []string{"DRIVER", "FOO", "BAR.BAZ", "ADMIN"}},
	}

	for _, tc := range testCases {
		u := mintUser()
		u.GrantRoles(tc.actOn)
		if len(tc.outcome) != len(u.Roles) {
			t.Errorf("Expected %v roles after GRANT", len(tc.outcome))
		}
	}
}

func TestRevokeRoles(t *testing.T) {
	testCases := []struct {
		actOn   []string
		outcome []string
	}{
		{[]string{"DRIVER"}, []string{"FOO", "BAR.BAZ"}},
		{[]string{"ADMIN"}, []string{"DRIVER", "FOO", "BAR.BAZ"}},
		{[]string{"FOO", "DRIVER"}, []string{"BAR.BAZ"}},
	}

	for _, tc := range testCases {
		u := mintUser()
		u.RevokeRoles(tc.actOn)
		if len(tc.outcome) != len(u.Roles) {
			t.Errorf("Expected %v roles after REVOKE", len(tc.outcome))
		}
	}
}

func TestDeterministicMarshaling(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	tok := &Token{
		Created:       time.Unix(1394914340, 0),
		AuthMechanism: "h2.PASSENGER",
		DeviceType:    "cli",
		Id:            "290840068858810368",
		Expires:       time.Unix(1394943140, 0),
		AutoRenew:     time.Unix(1394941340, 0),
		Roles: []string{
			"CUSTOMER",
			"H4BADMIN.0641085f-ee14-49de-b8e2-e330e4d76150",
			"H4BADMIN.40ebd8c6-698e-4347-95af-1430ab2fb331.PENDING",
			"H4BADMIN.5a706c69-a33b-4e8f-ba40-09ee760385e3.PENDING",
			"H4BADMIN.7d793409-19b6-43d8-a165-5cb94397ab01.PENDING",
		},
		Signature: `ORh06xWVqJJhKNCE0R5I0A+Dy1vnvlqaE0bbJWwUEvvsXyWs2wLZ8UpDHmTZkYLvyjcuh5aRkXAGh5Hm7HPQG1FrnNf741fYCSONeLW+zfbQnyYwuB54L81DkL2PqD7nu6xHkbBQqz4Ja4lSzpS7/E9oW8fWQxF26SyVBTy6wkv0inrSojA86vX6dYCNqT1pt5oBG+Re5jnBYiAesZ0h7elsv9yhLw1o1w9UzZ106qcIA2PRycD3ERDFcNGaPUkYvYAQS3sZXeQ9CqgyEOvpMArrvo+AjuO/l2gbrjEC5PzebZi2CKopUIOD64IfoeDryORotnj2uFE6lYlfdUY+3Q==`,
	}

	expected := `am=h2.PASSENGER:d=cli:id=290840068858810368:ct=1394914340:et=1394943140:rt=1394941340:r=CUSTOMER,H4BADMIN.0641085f-ee14-49de-b8e2-e330e4d76150,H4BADMIN.40ebd8c6-698e-4347-95af-1430ab2fb331.PENDING,H4BADMIN.5a706c69-a33b-4e8f-ba40-09ee760385e3.PENDING,H4BADMIN.7d793409-19b6-43d8-a165-5cb94397ab01.PENDING:sig=ORh06xWVqJJhKNCE0R5I0A+Dy1vnvlqaE0bbJWwUEvvsXyWs2wLZ8UpDHmTZkYLvyjcuh5aRkXAGh5Hm7HPQG1FrnNf741fYCSONeLW+zfbQnyYwuB54L81DkL2PqD7nu6xHkbBQqz4Ja4lSzpS7/E9oW8fWQxF26SyVBTy6wkv0inrSojA86vX6dYCNqT1pt5oBG+Re5jnBYiAesZ0h7elsv9yhLw1o1w9UzZ106qcIA2PRycD3ERDFcNGaPUkYvYAQS3sZXeQ9CqgyEOvpMArrvo+AjuO/l2gbrjEC5PzebZi2CKopUIOD64IfoeDryORotnj2uFE6lYlfdUY+3Q==`

	for i := 0; i < 50; i++ {
		if got := tok.String(); got != expected {
			t.Errorf("Token to String() does not match expected; '%v' vs expected '%v'", got, expected)
		}
	}
}
