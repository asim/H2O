# Build service

HTTP service responsible for providing information about available builds.

## Interface
All responses will be in JSON

    - GET    /builds                  - A list of all builds
    - GET    /build/{name}            - A list of all versions of a service
    - GET    /builds/{name}/{version} - The details of a specific build
    - DELETE /builds/{name}/{version} - Delete the build
    - POST   /builds                  - Create a new build
    
The expected JSON format is

```json
	{
	  "Hostname": "localhost",
	  "Architecture": "amd64",
	  "GoVersion": "1.1.1",
	  "SourceUrl": "https://github.com/hailo-platform/H2O/build-service/commit/53d6db9a88494e948b64415f53e1bf9da7efcc4b",
	  "BinaryUrl": "http://s3.amazon.com/abcdefg",
	  "Version": "20130627091746",
	  "Language": "Go",
	  "Name": "com.hailo-platform/H2O.kernel.build-service",
	  "TimeStamp": 1372346773
	}
```

## Inner workings

Broadly speaking this service will:

  - be able to answer queries about which binaries it has ever built

#### DB

We will store a record of every service built in RDS.

Each record will be an immutable record of a build that has completed, and
will include:

  - timestamp (UNIX timestamp when built, UTC seconds since 1970)
  - builtOn (hostname that did the build)
  - environment (debug information about the build environment, including
    Go version, architecture etc.)
  - sourceurl (the Github URL that prompted the build, down to the level of the
    exact commit hash)
  - service (fully qualified service name, eg: com.hailo-platform/H2O.kernel.discovery)
  - version (actually a date, eg: 20130601114431)
  - type (enum of Java/Go)
  - binaryurl (the location of the go binary or jar)

### Installation

    go get github.com/hailo-platform/H2O/build-service
    
build-service should now be in your go/bin directory

The service requires a MYSQL compatible DB (Amazon RDS works)

### Running

Set the following environmet variables

    BUILD_SERVICE_SQL_SERVER=db_hostname
    BUILD_SERVICE_SQL_PORT=db_port
    BUILD_SERVICE_SQL_USERNAME=db_username
    BUILD_SERVICE_SQL_PASSWORD=db_password
    BUILD_SERVICE_SQL_DATABASE=db_database_name
    
Run the following command to set up the tables

    build-service -createtables
    
Run the service

	build-service -port 1234 (-port is optional, the default is 3000)



