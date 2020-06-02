package dao

import (
	"fmt"
	"strings"
	"time"

	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/platform/errors"

	ldapproto "github.com/HailoOSS/ldap-service/proto"
)

// blatComparison is used to allow idempotence on create
type blatComparison struct {
	createdSeconds int64
	uid            string
	ids            string
	roles          string
}

func comparison(user *domain.User) *blatComparison {
	ids := make([]string, len(user.Ids))
	for i, id := range user.Ids {
		ids[i] = string(id)
	}
	return &blatComparison{
		createdSeconds: user.Created.Unix(),
		uid:            user.Uid,
		ids:            strings.Join(ids, "|"),
		roles:          strings.Join(user.Roles, ","),
	}
}

func (c *blatComparison) equals(other *blatComparison) bool {
	return c.createdSeconds == other.createdSeconds && c.uid == other.uid && c.ids == other.ids && c.roles == other.roles
}

func (c *blatComparison) diff(other *blatComparison) string {
	ret := make([]string, 0)
	if c.createdSeconds != other.createdSeconds {
		ret = append(ret, fmt.Sprintf("createdSeconds: %v != %v", c.createdSeconds, other.createdSeconds))
	}
	if c.uid != other.uid {
		ret = append(ret, fmt.Sprintf("UID: %v != %v", c.uid, other.uid))
	}
	if c.ids != other.ids {
		ret = append(ret, fmt.Sprintf("IDs: %v != %v", c.ids, other.ids))
	}
	if c.roles != other.roles {
		ret = append(ret, fmt.Sprintf("Roles: %v != %v", c.roles, other.roles))
	}
	if len(ret) > 0 {
		return strings.Join(ret, ", ")
	}
	return ""
}

// testBlat makes sure we're not blitzing over the top of some pre-existing user, with some
// ability to allow idempotence
func testBlat(user *domain.User, plainPass string) errors.Error {
	// the main problem with idempotence is the password hash, which will be different
	// on subsequent attempts

	existing, err := ReadUser(user.App, user.Uid)
	if err != nil {
		return errors.InternalServerError("com.HailoOSS.service.login.createuser.testblat", fmt.Sprintf("Failed to test for existing: %v", err))
	}
	if existing == nil {
		return nil
	}

	// assert:
	//  - same creation timestamp
	//  - same UID
	//  - same IDs
	//  - same Roles

	newFc := comparison(user)
	existingFc := comparison(existing)

	if !newFc.equals(existingFc) {
		return errors.BadRequest("com.HailoOSS.service.login.createuser.exists", fmt.Sprintf("User with ID '%v' already exists: %v", user.Uid, newFc.diff(existingFc)))
	}

	// now test password
	if err := existing.PasswordMatches([]byte(plainPass)); err != nil {
		return errors.BadRequest("com.HailoOSS.service.login.createuser.exists", fmt.Sprintf("User with ID '%v' already exists: Password does not match", user.Uid))
	}

	return nil
}

func ConvertLDAPUser(response *ldapproto.User) *domain.User {
	return &domain.User{
		App:            "ADMIN", // Only ADMIN LDAP users are currently supported
		Uid:            response.GetUsername(),
		Ids:            []domain.Id{domain.Id(response.GetUserID()), domain.Id(response.GetUserID())},
		Created:        time.Now(), // TODO: Store created in LDAP?
		Roles:          response.GetRoles(),
		PasswordChange: time.Now(),
		Status:         "enabled",
	}
}
