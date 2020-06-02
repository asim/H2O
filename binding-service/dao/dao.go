package dao

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	log "github.com/cihub/seelog"

	"github.com/HailoOSS/binding-service/domain"
	"github.com/HailoOSS/service/cassandra"
	"github.com/HailoOSS/service/sync"
	"github.com/HailoOSS/gossie/src/gossie"
)

// Defines crud actions for binding rules

const (
	BINDING_KEYSPACE = "binding"
	RULES_CF         = "binding_rules"
)

func CreateRule(rule *domain.Rule) error {
	log.Debugf("Creating rule %+v", rule)
	pool, err := cassandra.ConnectionPool(BINDING_KEYSPACE)
	if err != nil {
		return fmt.Errorf("Error while getting cassandra connection %s", err)
	}
	row, err := marshalRuleToRow(rule)
	if err != nil {
		return fmt.Errorf("Error while running cassandra insert for rule %s", err)
	} else {
		lock, err := getLock(rule.Service)
		if err != nil {
			return fmt.Errorf("Error while attempting to create lock %s", err)
		}
		defer lock.Unlock()

		existing, err := GetRules(rule.Service)
		if err != nil {
			return fmt.Errorf("Error while trying to get binding rules %s", err)
		}
		for _, r := range existing {
			if rule.Version == r.Version {
				// delete
				err = unsafeDelete(r)
				if err != nil {
					return fmt.Errorf("Error deleting existing rule before creation of new one %s", err)
				}
			}
		}
		err = pool.Writer().Insert(RULES_CF, row).Run()
		if err != nil {
			return fmt.Errorf("Error while running cassandra insert for rule %s", err)
		}
	}
	return err
}

func getHash(bytes []byte) string {
	h := md5.New()
	h.Write(bytes)
	return hex.EncodeToString(h.Sum(nil))
}

func marshalRuleToRow(rule *domain.Rule) (*gossie.Row, error) {
	var row gossie.Row
	row.Key, _ = gossie.Marshal(rule.Service, gossie.AsciiType)

	bytes, err := json.Marshal(rule)
	if err != nil {
		return nil, fmt.Errorf("Error while marshalling json %s", err)
	}
	colName, _ := gossie.Marshal(getHash(bytes), gossie.AsciiType)
	colVal, _ := gossie.Marshal(string(bytes), gossie.AsciiType)

	row.Columns = append(row.Columns, &gossie.Column{Name: colName, Value: colVal})
	return &row, nil
}

func DeleteRule(rule *domain.Rule) error {
	lock, err := getLock(rule.Service)
	if err != nil {
		return fmt.Errorf("Error while attempting to lock %s", err)
	}
	defer lock.Unlock()

	return unsafeDelete(rule)
}

func unsafeDelete(rule *domain.Rule) error {
	log.Debugf("Deleting rule %+v", rule)
	pool, err := cassandra.ConnectionPool(BINDING_KEYSPACE)
	if err != nil {
		return fmt.Errorf("Error while getting cassandra connection %s", err)
	}

	bytes, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("Error while marshalling json %s", err)
	}
	colName, _ := gossie.Marshal(getHash(bytes), gossie.AsciiType)
	rowKey, _ := gossie.Marshal(rule.Service, gossie.AsciiType)
	if err != nil {
		return fmt.Errorf("Error while running cassandra delete for rule %s", err)
	} else {
		err = pool.Writer().DeleteColumns(RULES_CF, rowKey, [][]byte{colName}).Run()
		if err != nil {
			return fmt.Errorf("Error while running cassandra delete for rule %s", err)
		}
	}
	return err
}

func GetRules(service string) ([]*domain.Rule, error) {
	log.Debugf("Getting rules for service %s", service)
	pool, err := cassandra.ConnectionPool(BINDING_KEYSPACE)
	if err != nil {
		return nil, fmt.Errorf("Error while getting cassandra connection %s", err)

	}

	rowKey, err := gossie.Marshal(service, gossie.AsciiType)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling rowKey %s", err)

	}

	ret := make([]*domain.Rule, 0)

	row, err := pool.Reader().Cf(RULES_CF).Get(rowKey)
	if err != nil {
		return nil, fmt.Errorf("Error while running cassandra query for service %s %+v", RULES_CF, err)
	}
	if row != nil {
		for _, col := range row.Columns {
			if len(col.Value) == 0 {
				// nil column, don't bother unmarshalling
				continue
			}

			v := col.Value
			r := &domain.Rule{}
			err = json.Unmarshal(v, r)
			if err != nil {
				return nil, fmt.Errorf("Error unmarshalling rule %s", err)
			} else {
				ret = append(ret, r)
			}

		}
	} else {
		log.Debugf("No rules found for service %s", service)
	}

	return ret, err
}

func getLock(service string) (sync.Lock, error) {
	return sync.RegionLock([]byte(service))
}
