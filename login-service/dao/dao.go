package dao

import (
	"fmt"
	"time"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/service/cassandra/timeseries"
	"github.com/HailoOSS/gossie/src/gossie"
)

const (
	Keyspace         = "login"
	cfSessions       = "sessions"
	cfUsers          = "users"
	cfEndpointAuths  = "endpointAuths"
	cfUserIndex      = "usersIndex"
	cfUserIndexIndex = "usersIndexIndex"
	cfUserSessions   = "userSessions"

	defaultType = gossie.UTF8Type
	separator   = "§"
)

var (
	sessionMapping gossie.Mapping
	userMapping    gossie.Mapping
	userTs         *timeseries.TimeSeries

	Cfs []string = []string{cfSessions, cfUsers, cfEndpointAuths, cfUserIndex, cfUserIndexIndex}
)

func init() {
	var err error
	sessionMapping, err = gossie.NewMapping(&storedSession{})
	if err != nil {
		log.Flush()
		panic(fmt.Sprintf("Invalid mapping - unexpected error: %v", err))
	}
	userMapping, err = gossie.NewMapping(&storedUser{})
	if err != nil {
		log.Flush()
		panic(fmt.Sprintf("Invalid mapping - unexpected error: %v", err))
	}
	userTs = &timeseries.TimeSeries{
		Ks:             Keyspace,
		Cf:             cfUserIndex,
		RowGranularity: time.Hour,
		Marshaler: func(i interface{}) (uid string, t time.Time) {
			return i.(*domain.User).Uid, i.(*domain.User).Created
		},
		SecondaryIndexer: func(i interface{}) (index string) {
			return string(i.(*domain.User).App)
		},
		IndexCf: cfUserIndexIndex,
	}
}
