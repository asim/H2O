package dao

import (
	"fmt"

	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/service/cassandra"
	"github.com/HailoOSS/gossie/src/gossie"
)

/*
 CF structure:
  ROW KEY                                     COL														VALUE
 [target service (thing granting access to)] [endpoint (granted to) {separator} (service granted from)] ROLE

 Query pattern: add/remove one, read whole row to get everything granted to a service
*/

const (
	maxAuthedServices = 1000
)

// ReadEndpointAuth grabs a list of all authorised services that can make
// requests to the supplied service
func ReadEndpointAuth(service string) ([]*domain.EndpointAuth, error) {
	pool, err := cassandra.ConnectionPool(Keyspace)
	if err != nil {
		return nil, fmt.Errorf("Failed to get connection pool: %v", err)
	}
	row, err := pool.Reader().Cf(cfEndpointAuths).Slice(&gossie.Slice{
		Start:    []byte{},
		End:      []byte{},
		Count:    maxAuthedServices,
		Reversed: false,
	}).Get([]byte(service))
	if err != nil {
		return nil, fmt.Errorf("Failed to read from C*: %v", err)
	}
	if row == nil {
		return []*domain.EndpointAuth{}, nil
	}

	// deal with results and map to domain
	ret := unmarshalEndpointAuth(row)

	return ret, nil
}

// WriteEndpointAuths defines a new rule that allows some service to call some endpoint
func WriteEndpointAuths(epas []*domain.EndpointAuth) error {
	if len(epas) == 0 {
		return fmt.Errorf("No rules to write")
	}

	pool, err := cassandra.ConnectionPool(Keyspace)
	if err != nil {
		return fmt.Errorf("Failed to get connection pool: %v", err)
	}

	writer := pool.Writer()
	for _, epa := range epas {
		marshalEndpointAuth(epa, writer)
	}
	if err := writer.Run(); err != nil {
		return fmt.Errorf("Failed to write to C*: %v", err)
	}

	return nil
}

// DeleteEndpointAuths will revoke these rules for allowing things to talk to each other
func DeleteEndpointAuths(epas []*domain.EndpointAuth) error {
	if len(epas) == 0 {
		return fmt.Errorf("No rules to delete")
	}

	pool, err := cassandra.ConnectionPool(Keyspace)
	if err != nil {
		return fmt.Errorf("Failed to get connection pool: %v", err)
	}

	writer := pool.Writer()
	for _, epa := range epas {
		deleteEndpointAuth(epa, writer)
	}
	if err := writer.Run(); err != nil {
		return fmt.Errorf("Failed to delete from C*: %v", err)
	}

	return nil
}
