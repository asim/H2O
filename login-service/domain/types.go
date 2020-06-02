package domain

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"

	glob "github.com/obeattie/ohmyglob"
	"golang.org/x/crypto/bcrypt"

	"github.com/HailoOSS/go-hailo-lib/multierror"
	serviceauth "github.com/HailoOSS/service/auth"
	inst "github.com/HailoOSS/service/instrumentation"

	log "github.com/cihub/seelog"
)

const (
	passwordHistoryLength = 12
	h1PasswordSalt        = "2103ccff866295b95e057e9c3a75ceaf"
)

var (
	bcryptCost = bcrypt.DefaultCost
)

// TYPES

// Token identifies a user for a limited time, with private-key signature in order that it can
// be verified as legit by users of the token
type Token struct {
	Created       time.Time
	AuthMechanism string
	DeviceType    string
	Id            string
	Expires       time.Time
	AutoRenew     time.Time
	Roles         []string
	Signature     string
}

// Session represents a user's session and comprises a random unique ID and a token
type Session struct {
	Id      string
	Created time.Time
	Token   Token
}

// Application represents some top-level namespace within which users can register
type Application string

// User represents a single user within an application
type User struct {
	App                   Application
	Uid                   string
	Ids                   []Id
	Created               time.Time
	Roles                 []string
	PasswordHistory       [][]byte
	Password              []byte
	PasswordChange        time.Time
	Status                string
	AccountExpirationDate string
}

// Login represents a single successful login action by a user
type Login struct {
	App           Application
	Uid           string
	LoggedIn      time.Time
	AuthMechanism string
	DeviceType    string
	Meta          map[string]string
}

// EndpointAuth represents a single service being allowed `Role` access to a single endpoint
type EndpointAuth struct {
	// ServiceName and EndpointName identify the single endpoint that we are allowing access to
	ServiceName, EndpointName string
	// AllowedService is the calling service that is granted access
	AllowedService string
	// Role is the role we are granting
	Role string
}

// Id represents some secondary ID for a user, eg: email address or phone number
// we are expecting these to naturally avoid collisions within an application - this is
// left
type Id string

// PasswordAssertion asserts something about a new password a user is trying to set, returns nil (pass OK) or error (pass BAD)
type PasswordAssertion func(newPass string, user *User) error

// Policy contains all policy rules pertaining to a given application type (one set of rules per type)
type Policy struct {
	// NewPasswordChecks contains a slice of assertions to make on any new password being set
	NewPasswordChecks []PasswordAssertion
	// PasswordValidFor defines a number of DAYS users can use a password for before it times out
	PasswordValidFor int
}

// METHODS

// APPLICATION

func (a Application) ToAuthMechanism() string {
	return fmt.Sprintf("h2.%v", a)
}

// SESSION

// Copy makes a copy of a token and returns a new one
func (s *Session) Copy() *Session {
	return &Session{
		Id:      s.Id,
		Created: s.Created,
		Token:   s.Token,
	}
}

// TOKEN

// DecodedSig returns base64 decoded bytes of the signature component
func (t *Token) DecodedSig() []byte {
	data, _ := base64.StdEncoding.DecodeString(t.Signature)
	return data
}

// DataToSign returns bytes of the data component of the signature - this discludes sig
func (t *Token) DataToSign() []byte {
	return []byte(t.dataComponent())
}

// Sign will take a raw byte signature and base64 encode it as a string and store as Signature
func (t *Token) Sign(sig []byte) {
	t.Signature = base64.StdEncoding.EncodeToString(sig)
}

// CanAutoRenew tests to see if this token can be auto-renewed at the current time
func (t *Token) CanAutoRenew() bool {
	if t.AutoRenew.IsZero() || t.AutoRenew.After(time.Now()) {
		return false
	}
	return true
}

// String for stringer
func (t *Token) String() string {
	return t.dataComponent() + ":sig=" + t.Signature
}

// String for stringer
func (t *Token) Application() Application {
	return Application(strings.TrimLeft(t.AuthMechanism, "h2."))
}

// Copy makes a copy of a token and returns a new one
func (t *Token) Copy() *Token {
	copyRoles := make([]string, len(t.Roles))
	copy(copyRoles, t.Roles)
	return &Token{
		Created:       t.Created,
		AuthMechanism: t.AuthMechanism,
		DeviceType:    t.DeviceType,
		Id:            t.Id,
		Expires:       t.Expires,
		AutoRenew:     t.AutoRenew,
		Roles:         copyRoles,
		Signature:     t.Signature,
	}
}

// dataComponent returns a string representation of the token, minus signature
func (t *Token) dataComponent() string {
	// The key order MUST be stable here (so we always generate equivalent strings). The existing order is important to
	// preserve.
	buf := bytes.Buffer{}

	buf.WriteString("am=")
	buf.WriteString(t.AuthMechanism)

	buf.WriteString(":d=")
	buf.WriteString(t.DeviceType)

	buf.WriteString(":id=")
	buf.WriteString(t.Id)

	buf.WriteString(":ct=")
	buf.WriteString(timeToUnixString(t.Created))

	buf.WriteString(":et=")
	buf.WriteString(timeToUnixString(t.Expires))

	buf.WriteString(":rt=")
	buf.WriteString(timeToUnixString(t.AutoRenew))

	buf.WriteString(":r=")
	buf.WriteString(strings.Join(t.Roles, ","))

	return buf.String()
}

// USER

// SetPassword will set the user's password, hashing it using bcrypt and storing the hash
func (u *User) SetPassword(plain string) error {
	if err := TestPolicy(plain, u); err.AnyErrors() {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return err
	}
	u.Password = hash
	// also add this to the array of recent passwords and then strip the list
	u.PasswordHistory = append(u.PasswordHistory, hash)
	if len(u.PasswordHistory) > passwordHistoryLength {
		startFrom := len(u.PasswordHistory) - passwordHistoryLength
		u.PasswordHistory = u.PasswordHistory[startFrom:]
	}
	// record the fact we've changed it
	u.PasswordChange = time.Now()
	return nil
}

// PasswordMatches tests whether the un-hashed pass p matches our stored hashed version
func (u *User) PasswordMatches(p []byte) error {
	if strings.HasPrefix(string(u.Password), "$2a$") {
		return bcrypt.CompareHashAndPassword(u.Password, p)
	}

	// try h1 driver format
	if len(u.Password) == 32 {
		inst.Counter(1.0, "auth.h1", 1)

		h := md5.New()
		io.WriteString(h, string(p))
		io.WriteString(h, h1PasswordSalt)
		expected := fmt.Sprintf("%x", h.Sum(nil))
		if string(u.Password) == expected {
			return nil
		}
	}

	// fallback to bcrypt
	return bcrypt.CompareHashAndPassword(u.Password, p)
}

// OldHashFormat tests whether we have the old h1 driver hash format
func (u *User) OldHashFormat() bool {
	return len(u.Password) == 32 && !strings.HasPrefix(string(u.Password), "$2a$")
}

// InPasswordHistory tests whether the un-hashed pass p matches a stored value in our last N history items
func (u *User) InPasswordHistory(p []byte, n int) bool {
	length := len(u.PasswordHistory)
	for i, counter := length-1, 0; i >= 0; i-- {
		if err := bcrypt.CompareHashAndPassword(u.PasswordHistory[i], p); err == nil {
			return true
		}
		counter++
		if counter >= n {
			break
		}
	}
	return false
}

// MustChangePassword will determine if this user should be FORCED to change their
// password before they are granted access
func (u *User) MustChangePassword() bool {
	return MustChangePassword(u)
}

// AnyAdminRoles will return true if this user has any role within the ADMIN category
func (u *User) AnyAdminRoles() bool {
	for _, r := range u.Roles {
		r = strings.TrimSpace(r)
		if r == "ADMIN" || strings.HasPrefix(r, "ADMIN.") {
			return true
		}
	}
	return false
}

// GrantRoles will merge the user's current roles with those supplied
func (u *User) GrantRoles(roles []string) {
	for _, role := range roles {
		if !roleInList(role, u.Roles) {
			u.Roles = append(u.Roles, role)
		}
	}
}

// RevokeRoles will remove supplied roles from the user's roles, if present in the first place
func (u *User) RevokeRoles(roles []string) {
	newRoles := make([]string, 0)
	for _, role := range u.Roles {
		if !roleInList(role, roles) {
			newRoles = append(newRoles, role)
		}
	}
	u.Roles = newRoles
}

func (u *User) IsDisabled() bool {
	return u.Status == "disabled"
}

func (u *User) IsAccountExpired() bool {
	if u.AccountExpirationDate == "" {
		return false
	}

	accountExpirationDate, err := time.Parse("2006-01-02", u.AccountExpirationDate)

	if err != nil {
		log.Errorf("Date parsing failed in IsAccountExpired %+v", err)
		return true
	}

	return accountExpirationDate.Before(time.Now())
}

func (u *User) ShouldBePublished() bool {
	return u.App == "ADMIN"
}

func roleInList(role string, list []string) bool {
	for _, r := range list {
		if role == r {
			return true
		}
	}
	return false
}

// ENDPOINT AUTH

// FqEndpoint returns the "fully qualified" endpoint name, which is service . endpoint
func (e *EndpointAuth) FqEndpoint() string {
	return fmt.Sprintf("%s.%s", e.ServiceName, e.EndpointName)
}

func timeToUnixString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf("%v", t.Unix())
}

// POLICY

// Test will check if a new password is valid for a policy
func (p *Policy) Test(newPass string, user *User) *multierror.MultiError {
	errs := multierror.New()
	for _, assertion := range p.NewPasswordChecks {
		if err := assertion(newPass, user); err != nil {
			errs.Add(err)
		}
	}

	return errs
}

// MustChangePassword will see if a user should be forced to change their password,
// based on this policy
func (p *Policy) MustChangePassword(user *User) bool {
	// policy says never needs changing
	if p.PasswordValidFor <= 0 {
		return false
	}
	if user.PasswordChange.After(time.Now().AddDate(0, 0, -p.PasswordValidFor)) {
		return false
	}

	return true
}

// TestPolicy will test if a password is valid against the policy defined for this
// user's application, or against the default policy if none defined for this application
func TestPolicy(newPass string, user *User) *multierror.MultiError {
	policy, ok := policies[user.App]
	if !ok {
		policy = defaultPolicy
	}
	return policy.Test(newPass, user)
}

// MustChangePassword will test if a user needs to change their password using the policy defined for this
// user's application, or against the default policy if none defined for this application
func MustChangePassword(user *User) bool {
	policy, ok := policies[user.App]
	if !ok {
		policy = defaultPolicy
	}
	return policy.MustChangePassword(user)
}

// ValidateRoleSet will validate a user roleset is valid (what a shock)
func ValidateRoleSet(roles []string) error {
	// Check there are no duplicates
	rolesMap := make(map[string]bool, len(roles))
	for _, r := range roles {
		if hasRole := rolesMap[r]; hasRole {
			return fmt.Errorf("Duplicate role \"%s\"", r)
		}
		rolesMap[r] = true
	}

	_, err := glob.CompileGlobSet(roles, serviceauth.RoleGlobOptions)
	return err
}
