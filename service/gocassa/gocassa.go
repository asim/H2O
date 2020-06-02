package gocassa

import (
	"regexp"
	"strings"

	s "github.com/HailoOSS/platform/server"

	"github.com/HailoOSS/gocassa"
)

var (
	DefaultConnector ConnectorFunc = gocqlConnector
	Connector        ConnectorFunc = DefaultConnector            // Connector is the active ConnectorFunc
	ksInvalid                      = regexp.MustCompile(`[^\w]`) // Matches invalid characters in keyspace names
)

type ConnectorFunc func(ks string) gocassa.Connection

// Returns a configured keyspace for this service. The name is composed from the service name, but this is not
// relevant to the caller. If the name of the keyspace is important, use KeySpaceWithName.
func KeySpace() gocassa.KeySpace {
	if s.Name == "" {
		panic("service.Name is not populated. Call KeySpace() after service.Name is set.")
	}

	// Convert the "." separators to "_"'s, and remove all non-alphanumeric characters
	name := strings.Replace(s.Name, ".", "_", -1)
	name = ksInvalid.ReplaceAllLiteralString(name, "")
	return KeySpaceWithName(name)
}

// Do not use this unless you have already existing data in an already existing keyspace.
func KeySpaceWithName(ks string) gocassa.KeySpace {
	conn := Connector(ks)
	return conn.KeySpace(ks)
}
