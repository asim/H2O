package dao

import (
	"fmt"
	"time"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/service/cassandra"
	"github.com/HailoOSS/gossie/src/gossie"
)

const (
	reindexBatch  = 100
	sleepInterval = 500 * time.Millisecond
)

// ReindexUsers is **temporary** so we can upgrade-in-place our data model and put users into a TS index
func ReindexUsers() {
	if err := doReindex(); err != nil {
		log.Criticalf("[Reindex] Failed to index users: %v", err)
	}
}

func doReindex() error {
	log.Info("[Reindex] Kicking off reindexing")

	pool, err := cassandra.ConnectionPool(Keyspace)
	if err != nil {
		return fmt.Errorf("[Reindex] Failed to get connection pool: %v", err)
	}

	start := []byte{}

	for {
		// prevent this getting carried away
		time.Sleep(sleepInterval)

		rows, err := pool.Reader().Cf(cfUsers).RangeGet(&gossie.Range{
			Start: start,
			End:   []byte{},
			Count: reindexBatch,
		})
		if err != nil {
			return fmt.Errorf("[Reindex] Failed to read from C*: %v", err)
		}
		// if start is present, then we're not on first row, so we have to skip that one
		if rows == nil || len(rows) == 0 || (len(start) > 0 && len(rows) == 1) {
			// no results
			break
		}

		for _, r := range rows {
			if len(start) > 0 && string(start) == string(r.Key) {
				// on loop 2+, skip start key as we have already processed
				continue
			}

			user, err := unmarshalUser(r)
			if err != nil {
				log.Warnf("[Reindex] Failed to unmarshal: %v", string(r.Key))
				continue
			}
			writer := pool.Writer()
			if err := userTs.Map(writer, user, nil); err != nil {
				log.Errorf("[Reindex] Failed to map user to time series %s '%s': %v", string(user.App), user.Uid, err)
				continue
			}
			if err := writer.Run(); err != nil {
				return fmt.Errorf("[Reindex] Failed to write to C*: %v", err)
			}
			log.Infof("[Reindex] Indexing %s '%s'", string(user.App), user.Uid)
		}

		start = rows[len(rows)-1].Key
	}

	return nil
}
