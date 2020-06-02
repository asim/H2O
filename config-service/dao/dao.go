package dao

import (
	"fmt"
	"time"

	"github.com/HailoOSS/config-service/domain"
	"github.com/HailoOSS/service/cassandra"
	"github.com/HailoOSS/service/cassandra/timeseries"
	"github.com/HailoOSS/gossie/src/gossie"
)

const (
	// Keyspace for C* storage
	Keyspace = "configservice"
	// CfConfig is CF where we store all config
	CfConfig = "configservice"
	// CfAudit is CF where we store a timeseries of all changes
	CfAudit = "audit"
	// CfAuditIndex is where we keep an index of which rows exist in our time series
	CfAuditIndex = "auditIndex"
	// CfAuditService is CF where we store a timeseries of all changes for a service
	CfAuditService = "auditService"
	// CfAuditServiceIndex is where we keep an index of which rows exist in our time series
	CfAuditServiceIndex = "auditServiceIndex"
)

var (
	// Cfs is a list of all active CFs, which we should monitor
	Cfs = []string{CfConfig, CfAudit, CfAuditIndex, CfAuditService, CfAuditServiceIndex}

	mapping         gossie.Mapping
	changeTs        *timeseries.TimeSeries
	serviceChangeTs *timeseries.TimeSeries
)

func init() {
	var err error
	mapping, err = gossie.NewMapping(&domain.ChangeSet{})
	if err != nil {
		panic("Failed to create mapping: " + err.Error())
	}
	changeTs = &timeseries.TimeSeries{
		Ks:             Keyspace,
		Cf:             CfAudit,
		RowGranularity: time.Hour * 24,
		Marshaler: func(i interface{}) (uid string, t time.Time) {
			return i.(*domain.ChangeSet).ChangeId, i.(*domain.ChangeSet).Timestamp
		},
		IndexCf: CfAuditIndex,
	}
	serviceChangeTs = &timeseries.TimeSeries{
		Ks:             Keyspace,
		Cf:             CfAuditService,
		RowGranularity: time.Hour * 24,
		Marshaler: func(i interface{}) (uid string, t time.Time) {
			return i.(*domain.ChangeSet).ChangeId, i.(*domain.ChangeSet).Timestamp
		},
		SecondaryIndexer: func(i interface{}) (index string) {
			return i.(*domain.ChangeSet).Id
		},
		IndexCf: CfAuditServiceIndex,
	}
}

type CassandraRepository struct{}

// ReadConfig fetches N config definitions
func (r CassandraRepository) ReadConfig(ids []string) ([]*domain.ChangeSet, error) {
	pool, err := cassandra.ConnectionPool(Keyspace)
	if err != nil {
		return nil, fmt.Errorf("Failed to get connection pool: %v", err)
	}

	query := pool.Query(mapping)

	// Need to convert to []interface{}
	// Since []string cannot be passed as an argument
	// expecting []inerface{}
	tempIds := make([]interface{}, len(ids))
	for i, id := range ids {
		tempIds[i] = id
	}
	result, err := query.MultiGet(tempIds)
	if err != nil {
		return nil, fmt.Errorf("Failed to get changesets (%v): %v", ids, err)
	}

	results := make(map[string]*domain.ChangeSet)
	for {
		cs := &domain.ChangeSet{}
		err = result.Next(cs)
		if err == gossie.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to get changesets (%v): %v", ids, err)
		}
		if cs.Id != "" {
			results[cs.Id] = cs
		}
	}
	// Make sure the change sets are sorted by id order
	sortedResults := make([]*domain.ChangeSet, 0)
	for _, id := range ids {
		if cs, ok := results[id]; ok {
			sortedResults = append(sortedResults, cs)
			// We want it only once
			delete(results, id)
		}
	}

	return sortedResults, nil
}

// UpdateConfig writes out a changeset
func (r *CassandraRepository) UpdateConfig(cs *domain.ChangeSet) error {
	pool, err := cassandra.ConnectionPool(Keyspace)
	if err != nil {
		return fmt.Errorf("Failed to get connection pool: %v", err)
	}

	row, err := mapping.Map(cs)
	if err != nil {
		return fmt.Errorf("Failed to map changeset: %v", err)
	}
	writer := pool.Writer()
	writer.Insert(CfConfig, row)
	changeTs.Map(writer, cs, nil)
	serviceChangeTs.Map(writer, cs, nil)

	if err := writer.Run(); err != nil {
		return fmt.Errorf("Error writing to C*: %v", err)
	}

	return nil
}

// ChangeLog returns a list of changesets within a certain time range
func (r *CassandraRepository) ChangeLog(start, end time.Time, count int, lastId string) ([]*domain.ChangeSet, string, error) {
	iter := changeTs.ReversedIterator(start, end, lastId, "")
	css := make([]*domain.ChangeSet, 0)

	for iter.Next() {
		cs := &domain.ChangeSet{}
		if err := iter.Item().Unmarshal(cs); err != nil {
			return nil, "", fmt.Errorf("Failed to unmarshal change set: %v", err)
		}
		css = append(css, cs)
		if len(css) >= count {
			break
		}
	}

	if err := iter.Err(); err != nil {
		return nil, "", fmt.Errorf("DAO read error: %v", err)
	}

	return css, iter.Last(), nil
}

// ServiceChangeLog returns a list of changesets within a certain time range for
// the given ID
func (r *CassandraRepository) ServiceChangeLog(id string, start, end time.Time, count int, lastId string) ([]*domain.ChangeSet, string, error) {
	iter := serviceChangeTs.ReversedIterator(start, end, lastId, id)
	css := make([]*domain.ChangeSet, 0)

	for iter.Next() {
		cs := &domain.ChangeSet{}
		if err := iter.Item().Unmarshal(cs); err != nil {
			return nil, "", fmt.Errorf("Failed to unmarshal change set: %v", err)
		}
		css = append(css, cs)
		if len(css) >= count {
			break
		}
	}

	if err := iter.Err(); err != nil {
		return nil, "", fmt.Errorf("DAO read error: %v", err)
	}

	return css, iter.Last(), nil
}
