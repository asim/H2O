package dao

import (
	"fmt"
	"time"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/service/cassandra"
	"github.com/HailoOSS/service/cassandra/timeseries"
	inst "github.com/HailoOSS/service/instrumentation"
)

// userLoginTs is our pure time-series of all login requests for a user
var userLoginTs *timeseries.TimeSeries = &timeseries.TimeSeries{
	Ks:             Keyspace,
	Cf:             "userLoginIndex",
	RowGranularity: time.Hour * 24 * 30,
	Marshaler: func(i interface{}) (uid string, t time.Time) {
		return i.(*domain.Login).Uid, i.(*domain.Login).LoggedIn
	},
	SecondaryIndexer: func(i interface{}) (index string) {
		login := i.(*domain.Login)
		return toIndex(login.App, login.Uid)
	},
}

func toIndex(app domain.Application, uid string) string {
	return string(app) + separator + uid
}

// WriteLogin will record details of a user login
func WriteLogin(login *domain.Login) error {
	pool, err := cassandra.ConnectionPool(Keyspace)
	if err != nil {
		return fmt.Errorf("Failed to get connection pool: %v", err)
	}

	writer := pool.Writer()
	userLoginTs.Map(writer, login, nil)
	t := time.Now()
	if err := writer.Run(); err != nil {
		return fmt.Errorf("Create error writing to C*: %v", err)
	}
	inst.Timing(1.0, "cassandra.write.writelogin", time.Since(t))

	return nil
}

// ReadUserLogins will return a list of user logins for a single user, within a time range
func ReadUserLogins(app domain.Application, uid string, start, end time.Time, count int, lastId string) ([]*domain.Login, string, error) {
	iter := userLoginTs.ReversedIterator(start, end, lastId, toIndex(app, uid))
	logins := make([]*domain.Login, 0)

	for iter.Next() {
		login := &domain.Login{}
		if err := iter.Item().Unmarshal(login); err != nil {
			return nil, "", fmt.Errorf("Failed to unmarshal login: %v", err)
		}
		logins = append(logins, login)
		if len(logins) >= count {
			break
		}
	}

	if err := iter.Err(); err != nil {
		return nil, "", fmt.Errorf("DAO read error: %v", err)
	}
	log.Debugf("Read %v logins as time series", len(logins))
	return logins, iter.Last(), nil
}
