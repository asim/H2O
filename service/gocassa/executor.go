package gocassa

import (
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff"

	"errors"

	log "github.com/cihub/seelog"
	"github.com/gocql/gocql"
	"github.com/HailoOSS/gocassa"

	"github.com/HailoOSS/service/config"
)

var (
	ksConnections    = map[string]gocassa.Connection{}
	ksConnectionsMtx sync.RWMutex
)

type gocqlExecutor struct {
	sync.RWMutex
	ks          string
	initialised bool
	initMtx     sync.RWMutex
	lastHash    uint32
	cfg         ksConfig
	session     *gocql.Session
}

func (e *gocqlExecutor) init() error {
	e.initMtx.RLock()
	if e.initialised {
		e.initMtx.RUnlock()
		return nil
	}

	e.initMtx.RUnlock()
	e.initMtx.Lock()
	defer e.initMtx.Unlock()
	if !e.initialised { // Guard against race
		cfg, err := getKsConfig(e.ks)
		if err != nil {
			return err
		}
		err = e.switchConfig(cfg)
		if err != nil {
			return err
		}
		go e.watchConfig()
		e.initialised = true
	}
	return nil
}

func (e *gocqlExecutor) switchConfig(newConfig ksConfig) error {
	e.Lock()
	defer e.Unlock()

	e.cfg = newConfig
	session, err := e.cfg.cc.CreateSession()
	if err != nil {
		return err
	}

	// Close existing session before switching to new session
	if e.session != nil {
		e.session.Close()
	}

	e.session = session
	e.lastHash = newConfig.hash()

	return nil
}

func (e *gocqlExecutor) watchConfig() {
	for _ = range config.SubscribeChanges() {
		// Keep trying to reload session until it successes
		b := backoff.NewExponentialBackOff()
		b.MaxElapsedTime = 0

		err := backoff.RetryNotify(e.reloadSession, b, func(err error, d time.Duration) {
			log.Errorf("%s, retrying in %s", err, d)
		})
		if err != nil {
			log.Errorf("[Cassandra:%s] Failed to reload config and backoff failed: %s", e.ks, err)
		}
	}
}

func (e *gocqlExecutor) reloadSession() error {
	e.RLock()
	ks := e.ks
	lastHash := e.lastHash
	e.RUnlock()

	if cfg, err := getKsConfig(ks); err != nil {
		return fmt.Errorf("[Cassandra:%s] Error getting new config: %s", ks, err)
	} else if cfg.hash() != lastHash {
		log.Infof("[Cassandra:%s] Config changed; invalidating connection pool", ks)

		if err := e.switchConfig(cfg); err != nil {
			return fmt.Errorf("[Cassandra:%s] Error creating new session: %s", ks, err)
		}

		log.Infof("[Cassandra:%s] Switched config to: %s", e.ks, cfg.String())
	} else {
		log.Debugf("[Cassandra:%s] Config changed but not invalidating connection pool (hash %d unchanged)", e.ks, e.lastHash)
	}

	return nil
}

func (e *gocqlExecutor) Query(stmt string, params ...interface{}) ([]map[string]interface{}, error) {
	return e.QueryWithOptions(gocassa.Options{}, stmt, params...)
}

func (e *gocqlExecutor) QueryWithOptions(opts gocassa.Options, stmt string, params ...interface{}) ([]map[string]interface{}, error) {
	if err := e.init(); err != nil {
		return nil, err
	}

	start := time.Now()
	e.RLock()
	session, ks := e.session, e.ks
	e.RUnlock()

	if session == nil {
		return nil, fmt.Errorf("No open session")
	}

	q := session.Query(stmt, params...)
	if opts.Consistency != nil {
		q = q.Consistency(*opts.Consistency)
	}

	iter := q.Iter()
	results := []map[string]interface{}{}
	result := map[string]interface{}{}
	for iter.MapScan(result) {
		results = append(results, result)
		result = map[string]interface{}{}
	}
	err := iter.Close()
	log.Tracef("[Cassandra:%s] Query took %s: %s", ks, time.Since(start).String(), stmt)
	return results, err
}

func (e *gocqlExecutor) Execute(stmt string, params ...interface{}) error {
	return e.ExecuteWithOptions(gocassa.Options{}, stmt, params...)
}

func (e *gocqlExecutor) ExecuteWithOptions(opts gocassa.Options, stmt string, params ...interface{}) error {
	if err := e.init(); err != nil {
		return err
	}

	start := time.Now()
	e.RLock()
	session, ks := e.session, e.ks
	e.RUnlock()

	if session == nil {
		return fmt.Errorf("No open session")
	}

	q := session.Query(stmt, params...)
	if opts.Consistency != nil {
		q = q.Consistency(*opts.Consistency)
	}

	err := q.Exec()
	log.Tracef("[Cassandra:%s] Execute took %s: %s", ks, time.Since(start).String(), stmt)
	return err
}

func (e *gocqlExecutor) ExecuteAtomically(stmt []string, params [][]interface{}) error {
	return errors.New("Execute atomically is not implemented yet")
}

func (e *gocqlExecutor) Close() {
	if e.session != nil {
		e.session.Close()
	}
}

func gocqlConnector(ks string) gocassa.Connection {
	ksConnectionsMtx.RLock()
	conn, ok := ksConnections[ks]
	ksConnectionsMtx.RUnlock()
	if ok {
		return conn
	}

	ksConnectionsMtx.Lock()
	defer ksConnectionsMtx.Unlock()
	if conn, ok = ksConnections[ks]; !ok { // Guard against race
		conn = gocassa.NewConnection(&gocqlExecutor{
			ks: ks,
		})
		ksConnections[ks] = conn
	}
	return conn
}
