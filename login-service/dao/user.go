package dao

import (
	"fmt"
	"time"

	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/login-service/sessinvalidator"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/service/cassandra"
	inst "github.com/HailoOSS/service/instrumentation"
)

// CreateUser will create a new user so long as none of the IDs already exist
// If another user is already using an index or ID, it will return an appropriate platform error
// The plainPass is used to give idempotence to the storage of a user
func CreateUser(user *domain.User, plainPass string) errors.Error {
	if _, _, ok := IsLDAPUser(user.App, user.Uid); ok {
		return errors.InternalServerError("com.HailoOSS.service.login.createuser.ldap", "Cant create LDAP user")
	}

	lock, err := lockUser(user)
	defer lock.Unlock()
	if err != nil {
		return errors.InternalServerError("com.HailoOSS.service.login.createuser.lockerr", fmt.Sprintf("Failed to achieve ZK lock for `create` operation: %v", err))
	}

	pool, err := cassandra.ConnectionPool(Keyspace)
	if err != nil {
		return errors.InternalServerError("com.HailoOSS.service.login.createuser.cassandra", fmt.Sprintf("Failed to get connection pool: %v", err))
	}

	// test to see if exists -- can happily "replay" a create request, but cannot overwrite something that exists with different data
	if err := testIndexes(pool, user); err != nil {
		return err
	}
	if err := testBlat(user, plainPass); err != nil {
		return err
	}

	writer := pool.Writer()
	if err := writeUser(user, writer, nil); err != nil {
		return errors.InternalServerError("com.HailoOSS.service.login.createuser.marshaling", fmt.Sprintf("Failed to marshal into mutation: %v", err))
	}
	t := time.Now()
	if err := writer.Run(); err != nil {
		return errors.InternalServerError("com.HailoOSS.service.login.createuser.cassandra", fmt.Sprintf("Create error writing to C*: %v", err))
	}
	inst.Timing(1.0, "cassandra.write.createuser", time.Since(t))

	return nil
}

// ReadUser returns a user, fetched by ID
func ReadUser(app domain.Application, uid string) (*domain.User, error) {
	if username, _, ok := IsLDAPUser(app, uid); ok {
		users, err := readLDAPUsers([]string{username})
		if err != nil {
			return nil, err
		}
		if len(users) != 1 {
			return nil, nil
		}
		return users[0], nil
	} else {
		return readH2User(app, uid)
	}
}

// If a user isn't found then returned array length won't match ids length
func MultiReadUser(app domain.Application, ids []string) ([]*domain.User, error) {
	// Fetch LDAP and H2 users separately if required
	h2IDs := []string{}
	ldapIDs := []string{}

	for _, uid := range ids {
		if _, _, ok := IsLDAPUser(app, uid); ok {
			ldapIDs = append(ldapIDs)
		} else {
			h2IDs = append(h2IDs, uid)
		}
	}

	users := []*domain.User{}

	// Fetch H2 Users
	h2Users, err := readH2Users(app, h2IDs)
	if err != nil {
		return nil, err
	}

	ldapUsers, err := readLDAPUsers(ldapIDs)
	if err != nil {
		return nil, err
	}

	users = append(users, h2Users...)
	users = append(users, ldapUsers...)

	return users, nil
}

// UpdateUser will update details of an existing user
func UpdateUser(user *domain.User) error {
	if _, _, ok := IsLDAPUser(user.App, user.Uid); ok {
		return fmt.Errorf("Cant update LDAP user")
	}

	lock, err := lockUser(user)
	defer lock.Unlock()
	if err != nil {
		return fmt.Errorf("Failed to achieve ZK lock for `update` operation: %v", err)
	}

	// test to see if exists -- must exist if we're updating
	existingUser, err := ReadUser(user.App, user.Uid)
	if err != nil {
		return fmt.Errorf("Failed to test for existing: %v", err)
	}
	if existingUser == nil {
		return fmt.Errorf("User %v:%s does not exist - cannot update", user.App, user.Uid)
	}

	pool, err := cassandra.ConnectionPool(Keyspace)
	if err != nil {
		return fmt.Errorf("Failed to get connection pool: %v", err)
	}
	if err := testIndexes(pool, user); err != nil {
		return err
	}
	writer := pool.Writer()
	if err := writeUser(user, writer, existingUser); err != nil {
		return fmt.Errorf("Failed to marshal into mutation: %v", err)
	}
	t := time.Now()
	if err := writer.Run(); err != nil {
		return fmt.Errorf("Create error writing to C*: %v", err)
	}
	inst.Timing(1.0, "cassandra.write.updateuser", time.Since(t))
	return nil
}

// DeleteUser deletes a user and expires all their active sessions
func DeleteUser(user *domain.User) error {
	if _, _, ok := IsLDAPUser(user.App, user.Uid); ok {
		return fmt.Errorf("Cant delete LDAP user")
	}

	// First, expire any active sessions
	sessionIds, err := ReadActiveSessionIdsFor(user.Uid)
	if err != nil {
		return fmt.Errorf("Failed to get active sessions: %s", err.Error())
	}
	for deviceType, sessionId := range sessionIds {
		if err = DeleteSession(sessionId, deviceType); err != nil {
			return fmt.Errorf("Failed to expire other sessions: %s", err.Error())
		}
		sessinvalidator.BroadcastSessionExpiry(sessionId)
	}

	if err := DeleteUserIndexes(user, user.Uid, user.Ids); err != nil {
		return err
	}
	// @todo expire any active sessions -- this relies on us storing sessions active by app+userId (eg: for all devices)
	// maybe we just store index a different way, eg: [app+userId][deviceType] = session JSON

	return nil
}

func DeleteUserIndexes(user *domain.User, uid string, ids []domain.Id) error {
	pool, err := cassandra.ConnectionPool(Keyspace)
	if err != nil {
		return fmt.Errorf("Failed to get connection pool: %s", err.Error())
	}
	writer := pool.Writer()
	deleteUser(user, writer, uid, ids)
	t := time.Now()
	if err := writer.Run(); err != nil {
		return fmt.Errorf("Write error deleting from C*: %s", err.Error())
	}
	inst.Timing(1.0, "cassandra.write.deleteuser", time.Since(t))

	return nil
}

// ReadUserList returns a timeseries list of all users, ordered by created
// timestamp. This function will only return users stored in cassandra (H2 users).
func ReadUserList(app domain.Application, start, end time.Time, count int, lastId string) ([]*domain.User, string, error) {
	iter := userTs.ReversedIterator(start, end, lastId, string(app))
	users := make([]*domain.User, 0)

	for iter.Next() {
		user := &domain.User{}
		if err := iter.Item().Unmarshal(user); err != nil {
			return nil, "", fmt.Errorf("Failed to unmarshal user: %v", err)
		}
		users = append(users, user)
		if len(users) >= count {
			break
		}
	}

	if err := iter.Err(); err != nil {
		return nil, "", fmt.Errorf("DAO read error: %v", err)
	}

	return users, iter.Last(), nil
}
