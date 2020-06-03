// for Cassandra 1.2 use:
// CREATE KEYSPACE provisioning WITH replication = {'class':'SimpleStrategy', 'replication_factor':1};

// for Cassandra 1.1 use:
CREATE KEYSPACE provisioning WITH strategy_class='SimpleStrategy' AND strategy_options:replication_factor=1;

USE provisioning;

CREATE TABLE provisioned_service (
	id varchar,
	servicename varchar,
	serviceversion bigint,
	machineclass varchar,
	nofilesoftlimit bigint, 
	nofilehardlimit bigint,
	servicetype bigint, 
	cpu bigint, 
	memory bigint,
	PRIMARY KEY (id)
);

CREATE INDEX ON provisioned_service(servicename);

CREATE INDEX ON provisioned_service(machineclass);

// populate db with initial values
INSERT INTO provisioned_service (id, servicename, serviceversion, machineclass,nofilesoftlimit,nofilehardlimit, servicetype)
	VALUES ('2e48e4541c5666adc859a9eafcd20458881872de', 'com.hailocab.service.log', 20140821140014, 'dev',1024, 4096,1);

CREATE TABLE run_levels (
	key text PRIMARY KEY,
	level bigint,
	region text
);

CREATE TABLE service_run_levels (
	key text PRIMARY KEY
);
