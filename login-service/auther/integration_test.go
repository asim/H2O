// +build integration

// (relies on having running CASSANDRA...)

package auther

import (
	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/service/config"
	"testing"
	"time"
)

func TestAuthUser(t *testing.T) {
	config.LoadFromService("testservice")

	// clean up/reset
	if existing, _ := dao.ReadUser(domain.Application("test"), "auther1"); existing != nil {
		dao.DeleteUser(existing)
	}

	// make a user to test against
	user := &domain.User{
		App:             domain.Application("test"),
		Uid:             "auther1",
		Ids:             []domain.Id{"auther@HailoOSS.com", "+447100000000"},
		Created:         time.Unix(1378740807, 0),
		Roles:           []string{"ADMIN"},
		PasswordHistory: make([][]byte, 0),
		Password:        []byte{},
		PasswordChange:  time.Time{},
	}

	// set password to something we know
	if err := user.SetPassword("foobarbaz"); err != nil {
		t.Fatalf("Failed to set password for user: %v", err)
	}

	if err := dao.CreateUser(user, "foobarbaz"); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	sess, err := Auth("test", "test", "auther1", []byte("foobarbaz"), []byte{}, map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatalf("Failed to auth: %v", err)
	}
	if sess == nil {
		t.Fatalf("Expecting to be authenticated")
	}

	// ok now let's read it back via session
	fetched, err := dao.ReadSession(sess.Id)
	if err != nil {
		t.Fatalf("Failed to read session back: %v", err)
	}
	if fetched == nil {
		t.Fatalf("Failed to read back session from DAO")
	}

	// now let's log in again; this should invalidate the first session
	newSess, err := Auth("test", "test", "auther1", []byte("foobarbaz"), []byte{}, map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatalf("Failed to auth: %v", err)
	}
	if newSess == nil {
		t.Fatalf("Expecting to be authenticated")
	}
	original, err := dao.ReadSession(sess.Id)
	if err != nil {
		t.Fatalf("Failed to read session back: %v", err)
	}
	if original != nil {
		t.Fatalf("Second auth did not invalidate the first session")
	}

	if err := dao.DeleteUser(user); err != nil {
		t.Fatalf("Fafiled to delete user: %v", err)
	}
}
