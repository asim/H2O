# Provisioning service

Kernel service which provides a classic SCRUD interface (minus Update) for all running services in this environment (and functionality to ensure this matches what is actually running - starting/stopping services as required)

## Interface

  - com.HailoOSS.kernel.provision.search (search for provisions which match a given name and/or running on this class of machine)
  - com.HailoOSS.kernel.provision.create (create a new provision)
  - com.HailoOSS.kernel.provision.read (read an existing provision)
  - com.HailoOSS.kernel.provision.delete (delete an exisitng provision)

There's no update endpoint because users just bring services up and down, they don't modify any of the fields.


## Inner workings

This service will:

  - start services
  - stop services
  - ensure services running match what are supposed to be running


#### DB

We will store a provisioned_service record for every service which is running in Cassandra.

Each record will be an immutable record of a provision that is running, and will include:

  - service_name (fully qualified service name, eg: com.HailoOSS.kernel.discovery)
  - service_version (actually a date, eg: 20130618183200)
  - machine_class (the class of machine the service should be running on)


## Setup

  - go get github.com/HailoOSS/goprotobuf/{proto,protoc-gen-go}
  - go get "github.com/pomack/thrift4go/lib/go/src/thrift"
  - cat create.cql | cqlsh -3

