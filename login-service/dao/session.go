package dao

import (
	"fmt"
	"time"

	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/service/cassandra"
	inst "github.com/HailoOSS/service/instrumentation"
	"github.com/HailoOSS/gossie/src/gossie"
)

// ReadSession fetches a single session - usually by base64-encoded sessionId, but also called
// by ReadActiveSessionFor for secondary indexed sessions
func ReadSession(rowKey string) (*domain.Session, error) {
	pool, err := cassandra.ConnectionPool(Keyspace)
	if err != nil {
		return nil, fmt.Errorf("Failed to get connection pool: %v", err)
	}
	t := time.Now()
	row, err := pool.Reader().Cf(cfSessions).Columns([][]byte{[]byte("session")}).Get([]byte(rowKey))
	inst.Timing(1.0, "cassandra.read.readsession", time.Since(t))
	if err != nil {
		return nil, fmt.Errorf("Failed to read from C*: %v", err)
	}
	if row == nil {
		// no results
		return nil, nil
	}
	session, err := unmarshalSession(row)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshall row to session: %v", err)
	}

	return session, nil
}

// ReadSession fetches a single session by secondary auth mechanism + device type + user ID index
func ReadActiveSessionFor(authMechanism, deviceType, userId string) (*domain.Session, error) {
	sess, err := ReadSession(string(authMechDeviceUserIdToRowKey(authMechanism, deviceType, userId)))
	return sess, err
}

// WriteSession is create/update combined (we don't care) for sessions
func WriteSession(sess *domain.Session) error {
	pool, err := cassandra.ConnectionPool(Keyspace)
	if err != nil {
		return fmt.Errorf("Failed to get connection pool: %v", err)
	}

	writer := pool.Writer()
	if err := writeSession(sess, writer); err != nil {
		return fmt.Errorf("Failed to marshal into mutation: %v", err)
	}

	// Add a column to the user's row in userSessions representing this session, keyed on the device type
	if sess.Token.Id != "" {
		writer.Insert(cfUserSessions, &gossie.Row{
			Key: []byte(sess.Token.Id),
			Columns: []*gossie.Column{{
				Name:  []byte(sess.Token.DeviceType),
				Value: []byte(sess.Id),
			}},
		})
	}

	t := time.Now()
	if err := writer.Run(); err != nil {
		return fmt.Errorf("Write error writing to C*: %v", err)
	}
	inst.Timing(1.0, "cassandra.write.writesession", time.Since(t))

	return nil
}

// DeleteSession will remove all knowledge of a session. The session device type must be specified because it needs to
// be expunged from the userSessions column family
func DeleteSession(rowKey, deviceType string) error {
	sess, err := ReadSession(rowKey)
	if err != nil {
		return fmt.Errorf("Delete session failed - error reading existing session: %v", err)
	}

	// exists?
	if sess == nil {
		return nil
	}

	pool, err := cassandra.ConnectionPool(Keyspace)
	if err != nil {
		return fmt.Errorf("Failed to get connection pool: %v", err)
	}
	writer := pool.Writer()
	deleteSession(sess, writer)
	writer.DeleteColumns(cfUserSessions, []byte(sess.Token.Id), [][]byte{[]byte(deviceType)})
	t := time.Now()
	if err := writer.Run(); err != nil {
		return fmt.Errorf("Write error deleting from C*: %v", err)
	}
	inst.Timing(1.0, "cassandra.write.deletesession", time.Since(t))

	return nil
}

// ReadActiveSessionIdsFor retrieves all active session IDs (keyed in a map by their device type) for a given user
// ID
func ReadActiveSessionIdsFor(userId string) (sessionIds map[string]string, err error) {
	pool, err := cassandra.ConnectionPool(Keyspace)
	if err != nil {
		return nil, fmt.Errorf("Failed to get connection pool: %v", err)
	}

	t := time.Now()
	row, err := pool.Reader().Cf(cfUserSessions).Get([]byte(userId))
	inst.Timing(1.0, "cassandra.read.readactivesessionids", time.Since(t))
	if err != nil {
		return nil, err
	}
	if row == nil {
		return make(map[string]string, 0), nil
	}

	sessionIds = make(map[string]string, len(row.Columns))
	for _, col := range row.Columns {
		sessionIds[string(col.Name)] = string(col.Value)
	}
	return sessionIds, nil
}
