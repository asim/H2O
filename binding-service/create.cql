// for Cassandra 1.2 use:
// CREATE KEYSPACE binding WITH replication = {'class':'SimpleStrategy', 'replication_factor':3};

// for Cassandra 1.1 use:
CREATE KEYSPACE binding WITH strategy_class='NetworkTopologyStrategy'
  AND strategy_options:GL2L = '3'
  AND strategy_options:GL1L = '3'
  AND strategy_options:GL3L = '3';

USE binding;

CREATE columnfamily binding_rules (
	key text primary key
) with
  comparator = text and
  default_validation = text
;
