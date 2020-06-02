// +build integration

// (relies on having running CASSANDRA...)

package dao

import (
	"testing"
	"time"

	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/service/cassandra"
	"github.com/HailoOSS/service/config"
	"github.com/HailoOSS/gossie/src/gossie"
)

func TestReadSession(t *testing.T) {
	config.LoadFromService("testservice")

	// put column first
	rowKey := `9Bf3155ejPvwuzhU33YUmYOAz7tVI1dv+h+vBCpWivvV68Sz8PVMH3OaeM/+h3RM`
	storedSess := `{"key":"9Bf3155ejPvwuzhU33YUmYOAz7tVI1dv+h+vBCpWivvV68Sz8PVMH3OaeM\/+h3RM","createdTimestamp":1378377733,"token":{"createdTimestamp":1378377733,"authMechanism":"admin","deviceType":"cli","id":"dave","expiryTimestamp":1378406533,"autoRenewTimestamp":null,"roleCollection":{"ADMIN":"ADMIN"},"signature":"IFvyIAtFhMu9Gkh\/cf\/2JtwZe8+Kk2jf4xSkAHuaJwtmlnvrIBAVmAPytzKv2hFumd7gyP\/lPrGXutKAeBmUrWYd\/NeewKRdT\/vaAmNgtT9qi3QqeiloOfyieZjcruFnzMRc6XY4CJYHbXth2qUyF+gYFT0MIV3Dv6OUDNDOMymOuQ8z67rDMsyoKKKko5UY8oY+kURnBH0tkGy5IFLY0sH9LWldSJYXTzKWSQl3siwkCoCeCpOn9j4q2fx4vAitJ1+Vz078V1BgHDWkyx3AlM9QefRcypBOFZZ2x+fwSzNEqm37FU5DJyDA3vhCQn2Wn2DodJVr9NWRBSXPZ1\/0zQ=="}}`
	row := &gossie.Row{
		Key: []byte(rowKey),
		Columns: []*gossie.Column{
			{
				Name:  []byte("session"),
				Value: []byte(storedSess),
			},
		},
	}
	pool, _ := cassandra.ConnectionPool(keyspace)
	pool.Writer().Insert(cfSessions, row).Run()

	sess, err := ReadSession(rowKey)
	if err != nil {
		t.Fatalf("Error reading from C*: %v", err)
	}
	if sess == nil {
		t.Fatal("Expecting row (session) to exist in C*")

	}
	if sess.Id != `9Bf3155ejPvwuzhU33YUmYOAz7tVI1dv+h+vBCpWivvV68Sz8PVMH3OaeM/+h3RM` {
		t.Fatal("Session ID does not match expected")
	}
	if !sess.Created.Equal(time.Unix(1378377733, 0)) {
		t.Fatal("Session Created does not match expected")
	}
	if !sess.Token.Created.Equal(time.Unix(1378377733, 0)) {
		t.Fatal("Token Created does not match expected")
	}
	if sess.Token.AuthMechanism != "admin" {
		t.Fatal("Token AuthMechanism does not match expected")
	}
	if sess.Token.DeviceType != "cli" {
		t.Fatal("Token DeviceType does not match expected")
	}
	if sess.Token.Id != "dave" {
		t.Fatal("Token Id does not match expected")
	}
	if !sess.Token.Expires.Equal(time.Unix(1378406533, 0)) {
		t.Fatal("Token Expires does not match expected")
	}
	if !sess.Token.AutoRenew.IsZero() {
		t.Fatal("Token AutoRenew does not match expected")
	}
	if len(sess.Token.Roles) != 1 || sess.Token.Roles[0] != "ADMIN" {
		t.Fatal("Token Roles does not match expected")
	}
}

func TestWriteSession(t *testing.T) {
	config.LoadFromService("testservice")
	session := &domain.Session{
		Id:      "testsession",
		Created: time.Now(),
		Token: domain.Token{
			Created:       time.Unix(1378377733, 0),
			AuthMechanism: "admin",
			DeviceType:    "cli",
			Id:            "dave",
			Expires:       time.Unix(1378406533, 0),
			AutoRenew:     time.Time{},
			Roles:         []string{"ADMIN"},
			Signature:     `IFvyIAtFhMu9Gkh/cf/2JtwZe8+Kk2jf4xSkAHuaJwtmlnvrIBAVmAPytzKv2hFumd7gyP/lPrGXutKAeBmUrWYd/NeewKRdT/vaAmNgtT9qi3QqeiloOfyieZjcruFnzMRc6XY4CJYHbXth2qUyF+gYFT0MIV3Dv6OUDNDOMymOuQ8z67rDMsyoKKKko5UY8oY+kURnBH0tkGy5IFLY0sH9LWldSJYXTzKWSQl3siwkCoCeCpOn9j4q2fx4vAitJ1+Vz078V1BgHDWkyx3AlM9QefRcypBOFZZ2x+fwSzNEqm37FU5DJyDA3vhCQn2Wn2DodJVr9NWRBSXPZ1/0zQ==`,
		},
	}
	err := WriteSession(session)
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	// let's read it back as well
	read, err := ReadSession("testsession")
	if err != nil {
		t.Fatalf("Failed to read it back: %v", err)
	}
	if read.Token.Id != "dave" {
		t.Fatal("Token ID not as expected after write/read")
	}
}

func TestCreateUser(t *testing.T) {
	config.LoadFromService("testservice")

	user := &domain.User{
		App:             domain.Application("test"),
		Uid:             "dave",
		Ids:             []domain.Id{"dg@HailoOSS.com", "+447000000000"},
		Created:         time.Unix(1378740807, 0),
		Roles:           []string{"ADMIN"},
		PasswordHistory: make([][]byte, 0),
		Password:        []byte{},
		PasswordChange:  time.Time{},
	}
	user.SetPassword("MyPassword1")
	if err := CreateUser(user, "MyPassword1"); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	user.Uid = "someone"
	if err := CreateUser(user, "MyPassword1"); err == nil {
		t.Fatalf("Created duplicate user which should not happen: %v", err)
	}

	user.Uid = "dave"
	DeleteUser(user)
}

func TestCreateUserIdempotent(t *testing.T) {
	config.LoadFromService("testservice")

	user := &domain.User{
		App:             domain.Application("test"),
		Uid:             "daveyg",
		Ids:             []domain.Id{"dg+1@HailoOSS.com", "+447000000001"},
		Created:         time.Unix(1378740807, 0),
		Roles:           []string{"ADMIN"},
		PasswordHistory: make([][]byte, 0),
		Password:        []byte{},
		PasswordChange:  time.Time{},
	}
	user.SetPassword("MyPassword5454")
	if err := CreateUser(user, "MyPassword5454"); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// ok, let's try again -- minting different details this time
	// NOTE: salted pass via bcrypt, so end up with diff things
	user = &domain.User{
		App:             domain.Application("test"),
		Uid:             "daveyg",
		Ids:             []domain.Id{"dg+1@HailoOSS.com", "+447000000001"},
		Created:         time.Unix(1378740807, 0),
		Roles:           []string{"ADMIN"},
		PasswordHistory: make([][]byte, 0),
		Password:        []byte{},
		PasswordChange:  time.Time{},
	}
	user.SetPassword("MyPassword5454")
	if err := CreateUser(user, "MyPassword5454"); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	DeleteUser(user)
}

func TestCreateUserIndexConstraints(t *testing.T) {
	config.LoadFromService("testservice")

	user := &domain.User{
		App:             domain.Application("test"),
		Uid:             "constraintstest",
		Ids:             []domain.Id{"dg+11@HailoOSS.com", "+447000000002"},
		Created:         time.Unix(1378740807, 0),
		Roles:           []string{"ADMIN"},
		PasswordHistory: make([][]byte, 0),
		Password:        []byte{},
		PasswordChange:  time.Time{},
	}
	user.SetPassword("MyPassword5454")
	if err := CreateUser(user, "MyPassword5454"); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// copy and tweak
	cUser := user
	cUser.Uid = "constraintstest2"

	// should fail -- same IDs
	if err := CreateUser(user, "MyPassword5454"); err == nil {
		t.Fatal("Should not be able to create duplicate user with same indexes")
	} else {
		// err should be known
		if err.Code() != "com.HailoOSS.service.login.createuser.indexinuse" {
			t.Fatalf("Index constraint err code should be com.HailoOSS.service.login.createuser.indexinuse, got %v", err.Code())
		}
	}

	DeleteUser(user)
}

func TestWriteWithNoRoles(t *testing.T) {
	config.LoadFromService("testservice")
	session := &domain.Session{
		Id:      "testsessionnoroles",
		Created: time.Now(),
		Token: domain.Token{
			Created:       time.Unix(1378377733, 0),
			AuthMechanism: "admin",
			DeviceType:    "cli",
			Id:            "daveg",
			Expires:       time.Unix(1378406533, 0),
			AutoRenew:     time.Time{},
			Roles:         []string{},
			Signature:     `IFvyIAtFhMu9Gkh/cf/2JtwZe8+Kk2jf4xSkAHuaJwtmlnvrIBAVmAPytzKv2hFumd7gyP/lPrGXutKAeBmUrWYd/NeewKRdT/vaAmNgtT9qi3QqeiloOfyieZjcruFnzMRc6XY4CJYHbXth2qUyF+gYFT0MIV3Dv6OUDNDOMymOuQ8z67rDMsyoKKKko5UY8oY+kURnBH0tkGy5IFLY0sH9LWldSJYXTzKWSQl3siwkCoCeCpOn9j4q2fx4vAitJ1+Vz078V1BgHDWkyx3AlM9QefRcypBOFZZ2x+fwSzNEqm37FU5DJyDA3vhCQn2Wn2DodJVr9NWRBSXPZ1/0zQ==`,
		},
	}
	err := WriteSession(session)
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	// let's read it back as well
	read, err := ReadSession("testsessionnoroles")
	if err != nil {
		t.Fatalf("Failed to read it back: %v", err)
	}
	if read.Token.Id != "daveg" {
		t.Fatal("Token ID not as expected after write/read")
	}
}
