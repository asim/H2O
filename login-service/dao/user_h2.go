package dao

import (
	"fmt"
	"strings"
	"time"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/go-hailo-lib/util"
	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/service/cassandra"
	inst "github.com/HailoOSS/service/instrumentation"
	"github.com/HailoOSS/gossie/src/gossie"
)

var (
	userColumns [][]byte
)

func init() {
	userColumns = util.FieldNamesAsByteSlice(storedUser{})
}

func getCorrectId(rowKey string, ids []domain.Id) string {
	for _, v := range ids {
		if strings.Contains(rowKey, string(v)) {
			return string(v)
		}
	}
	log.Warnf("rowKey %v does not contain any of the ids %v", rowKey, ids)
	return ""
}

// testIndexes makes sure a secondary index (user.Ids) is not in use by someone else already
func testIndexes(pool gossie.ConnectionPool, user *domain.User) errors.Error {
	// look for _other_ users with this secondary ID
	if len(user.Ids) > 0 {
		tempIds := make([][]byte, len(user.Ids))
		for i, id := range user.Ids {
			tempIds[i] = []byte(userIdToRowKey(user.App, string(id)))
		}
		t := time.Now()
		rows, err := pool.Reader().Cf(cfUsers).Slice(&gossie.Slice{
			Start:    []byte("uid"),
			End:      []byte("uid"),
			Count:    1,
			Reversed: false,
		}).MultiGet(tempIds)
		inst.Timing(1.0, "cassandra.read.testindexes", time.Since(t))
		if err != nil {
			return errors.InternalServerError("com.HailoOSS.service.login.createuser.cassandra", fmt.Sprintf("Failed to test for existing: %v", err))
		}
		for _, row := range rows {
			if len(row.Columns) < 1 || string(row.Columns[0].Name) != "uid" {
				log.Warnf("Unexpected result from C* - expecting column with name 'uid'")
				continue
			}
			if string(row.Columns[0].Value) != user.Uid {
				id := getCorrectId(string(row.Key), user.Ids)
				return errors.BadRequest("com.HailoOSS.service.login.createuser.indexinuse", fmt.Sprintf("Index '%s' is already linked to another user '%s'", id, string(row.Columns[0].Value)))
			}
		}
	}
	return nil
}

func readH2User(app domain.Application, uid string) (*domain.User, error) {
	pool, err := cassandra.ConnectionPool(Keyspace)
	if err != nil {
		return nil, fmt.Errorf("Failed to get connection pool: %v", err)
	}
	t := time.Now()
	row, err := pool.Reader().Cf(cfUsers).Columns(userColumns).Get(userIdToRowKey(app, uid))
	inst.Timing(1.0, "cassandra.read.readuser", time.Since(t))
	if err != nil {
		return nil, fmt.Errorf("Failed to read from C*: %v", err)
	}
	if row == nil {
		// no results
		return nil, nil
	}
	user, err := unmarshalUser(row)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshall row to user (invalid row): %v", err)
	}

	return user, nil
}

// readH2Users reads the users from cassandra, if a user isn't found then
// the returned array length won't match ids length
func readH2Users(app domain.Application, ids []string) ([]*domain.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	pool, err := cassandra.ConnectionPool(Keyspace)
	if err != nil {
		return nil, fmt.Errorf("Failed to get connection pool: %v", err)
	}

	start := time.Now()
	rows, err := pool.Reader().Cf(cfUsers).MultiGet(userIdsToRowKeys(app, ids))
	if err != nil {
		return nil, fmt.Errorf("Failed to read from C*: %v", err)
	}
	inst.Timing(0.1, "dao.multireaduser.timing", time.Since(start))
	inst.Counter(0.1, "dao.multireaduser.rows", len(ids))

	ret := make([]*domain.User, 0, len(rows))
	for _, row := range rows {
		user, err := unmarshalUser(row)
		if err != nil {
			return nil, fmt.Errorf("Failed to unmarshall row to customer: %v", err)
		}
		ret = append(ret, user)
	}

	return ret, nil
}
