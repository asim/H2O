package dao

import (
	"fmt"
	"github.com/HailoOSS/service/cassandra"
	"github.com/HailoOSS/gossie/src/gossie"
)

const (
	regionPrefix       = "REGION:"
	servicesRowKey     = "SERVICES"
	runLevelCf         = "run_levels"
	serviceRunLevelsCf = "service_run_levels"
)

var (
	regions = []string{"us-east-1", "eu-west-1", "ap-northeast-1"}
)

func ReadRunLevels() ([]*RunLevel, error) {
	var keys [][]byte

	for _, region := range regions {
		keys = append(keys, []byte(regionPrefix+region))
	}

	pool, err := cassandra.ConnectionPool(keyspace)
	if err != nil {
		return nil, fmt.Errorf("Failed to get connection pool: %v", err)
	}

	rows, err := pool.Reader().Cf(runLevelCf).MultiGet(keys)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch row: %v", err)
	}

	if rows == nil {
		return nil, fmt.Errorf("Rows not found: %v", err)
	}

	result := &rowProvider{
		columnLimit: 100,
		buffer:      rows,
		mapping:     rlmapping,
	}

	var response []*RunLevel

	for {
		st := &storedRunLevel{}
		err := result.Next(st)
		if err != nil {
			break
		}
		response = append(response, st.runLevel())
	}

	return response, nil
}

func ReadRunLevel(region string) (*RunLevel, error) {
	pool, err := cassandra.ConnectionPool(keyspace)
	if err != nil {
		return nil, fmt.Errorf("Failed to get connection pool: %v", err)
	}

	query := pool.Query(rlmapping)
	result, err := query.Get(regionPrefix + region)
	if err != nil {
		return nil, fmt.Errorf("C* query failed: %v", err)
	}

	res := &storedRunLevel{}
	if err := result.Next(res); err != nil {
		return nil, fmt.Errorf("Failed to read result %v", err)
	}

	return res.runLevel(), nil
}

func SetRunLevel(region string, level int64) error {
	runLevel := &RunLevel{
		Region: region,
		Level:  level,
	}

	rl := runLevel.stored()

	row, err := rlmapping.Map(rl)
	if err != nil {
		return fmt.Errorf("Error mapping run levels: %v", err)
	}

	pool, err := cassandra.ConnectionPool(keyspace)
	if err != nil {
		return fmt.Errorf("Failed to get connection pool: %v", err)
	}

	if err := pool.Writer().Insert(runLevelCf, row).Run(); err != nil {
		return fmt.Errorf("Create error writing to C*: %v", err)
	}

	return nil
}

func ReadServiceRunLevels() ([]*ServiceRunLevels, error) {
	pool, err := cassandra.ConnectionPool(keyspace)
	if err != nil {
		return nil, fmt.Errorf("Failed to get connection pool: %v", err)
	}

	rowKey, err := gossie.Marshal(servicesRowKey, gossie.UTF8Type)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal with gossie: %v", err)
	}

	indexRow, err := pool.Reader().Cf(serviceRunLevelsCf).Slice(&gossie.Slice{Count: 1000}).Get(rowKey)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch row: %v", err)
	}
	if indexRow == nil {
		return nil, fmt.Errorf("Service run level index row not found")
	}

	var keys [][]byte

	for _, column := range indexRow.Columns {
		keys = append(keys, column.Name)
	}

	var response []*ServiceRunLevels

	rows, err := pool.Reader().Cf(serviceRunLevelsCf).MultiGet(keys)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch row: %v", err)
	}

	result := &rowProvider{
		columnLimit: 100,
		buffer:      rows,
		mapping:     rlsmapping,
	}

	for {
		st := &storedServiceRunLevels{}
		err := result.Next(st)
		if err != nil { // Done is also returned in err
			break
		}
		response = append(response, st.runLevels())
	}

	return response, nil
}

func ReadServiceRunLevel(serviceName string) (*ServiceRunLevels, error) {
	pool, err := cassandra.ConnectionPool(keyspace)
	if err != nil {
		return nil, fmt.Errorf("Failed to get connection pool: %v", err)
	}

	query := pool.Query(rlsmapping)
	result, err := query.Get(serviceName)
	if err != nil {
		return nil, fmt.Errorf("C* query failed: %v", err)
	}

	res := &storedServiceRunLevels{}
	if err := result.Next(res); err != nil {
		return nil, fmt.Errorf("Failed to read result %v", err)
	}

	return res.runLevels(), nil
}

func SetServiceRunLevels(serviceName string, levels [6]bool) error {
	pool, err := cassandra.ConnectionPool(keyspace)
	if err != nil {
		return fmt.Errorf("Failed to get connection pool: %v", err)
	}

	columnName, err := gossie.Marshal(serviceName, gossie.UTF8Type)
	if err != nil {
		return fmt.Errorf("Failed to marshal with gossie: %v", err)
	}

	rowKey, err := gossie.Marshal(servicesRowKey, gossie.UTF8Type)
	if err != nil {
		return fmt.Errorf("Failed to marshal with gossie: %v", err)
	}

	indexRow := &gossie.Row{
		Key: rowKey,
		Columns: []*gossie.Column{
			&gossie.Column{
				Name: columnName,
			},
		},
	}

	// insert index
	if err := pool.Writer().Insert(serviceRunLevelsCf, indexRow).Run(); err != nil {
		return fmt.Errorf("Create error writing to C*: %v", err)
	}

	runLevels := &ServiceRunLevels{
		ServiceName: serviceName,
		Levels:      levels,
	}

	rls := runLevels.stored()

	row, err := rlsmapping.Map(rls)
	if err != nil {
		return fmt.Errorf("Error mapping service run levels: %v", err)
	}

	// insert service run level
	if err := pool.Writer().Insert(serviceRunLevelsCf, row).Run(); err != nil {
		return fmt.Errorf("Create error writing to C*: %v", err)
	}

	return nil

}
