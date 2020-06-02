# Config Service

Responsible for storing application configuration.

Features:

 - stores config in C* to give high availability
 - can store config under arbitrary IDs and then merge these ("compile") in any
   combinations to create a single set of configuration
 - configuration is all represented as JSON
 - authenticates updates via the login service and maintains an audit trail

## Install

If you're using Boxen, and have H2 installed:

    go get github.com/HailoOSS/config-service
    cat dao/cassandra.dev | cassandra-cli -p 19160

If you're starting from scratch, you have a chicken and egg problem where the config
service needs some config to get going, so that it can serve config to all the other
services that make up the H2 kernel. This is why we have the HTTP interface for
`compile`, however it doesn't help you get config in to start with.

For this, you can use the bootstrap script:

    cd bootstrap
    go build
    ./bootstrap -config=`cat ../schema/base.boxen.json | jq -c .`

If you need to update config, you can do so via the [call API](github.com/HailoOSS/call-api):

	curl -d service=com.HailoOSS.service.config \
		 -d endpoint=update \
		 -d request="{\"id\":\"H2:BASE\",\"message\":\"Install config\",\"config\":`cat schema/base.boxen.json | php -r 'echo json_encode(stream_get_contents(STDIN));'`}" \
		 http://localhost:8080/v2/h2/call?session_id=R%2FP4AmzEi2xUIS%2Bt6WpLv%2Bq3HNfgSj18FWfjDVVACh%2F4kg3wAd2%2BbQh%2B51MqWrOJ

On startup, the config service currently gets a bootstrap list of C* nodes from
an environment variable `H2_CONFIG_SERVICE_CASSANDRA`. Then it can actually extract the
_real_ config. This environment variable should be configured automatically within
all Hailo environments including boxen, so you shouldn't need to do anything with this.

Another thing to watch out for is to make sure you have the environment variable `H2_CONFIG_SERVICE_ADDR`
defined:

    export H2_CONFIG_SERVICE_ADDR=http://localhost:8097

This should be automatic if you're using Boxen.

## Use

The config service in H2 stores both **city config** and **service config**. We keep
these two use cases distinct and seperate.

#### City config

The config is stored under ID `CITY:<code>`, for example `CITY:PHL`. You can
inspect this directly, if you wish. When applications make use of city config,
they should do so via the [localisation library](https://github.com/HailoOSS/go-hailo-lib/blob/master/localisation/city.go#L29):

	localisation.City("PHL").Config("at", "path").AsString("foo")

This will load the config via the [city service](https://github.com/HailoOSS/city-service/tree/master/proto/config).

For cities there is **no hierarchy** - they simply fetch this one config file. Cities load
their config via the H2 (RMQ) interface, which relies on the platform being up.

#### Service config

Service config is loaded by both Java and Go services on launch. For Go, you
should be accessing config via the [config service layer library](https://github.com/HailoOSS/service/tree/master/config):

	config.AtPath("foo","bar").AsString("Foo")

Services load config based on a **hierarchy**. The layers are as follows:

  - `H2:BASE`
  - `H2:BASE:<service-name>` - for example `H2:BASE:com.HailoOSS.service.job`
  - `H2:REGION:<aws region>` - for example `H2:REGION:us-east-1`
  - `H2:REGION:<aws region>:<service-name>` - for example `H2:REGION:us-east-1:com.HailoOSS.service.job`

Services load their config via an HTTP interface, so we do not rely on the RMQ
platform being up and available.

## An Example

hshell can be used to set up canfig, for example, for the 'allocation' service as follows:

    execute update {"id": "H2:BASE:com.HailoOSS.service.allocation", "path": "hailo/service/allocation", "message": "Hope to hell this works", "config": "{\"cycleTime\":\"10s\",\"expiryTime\":\"75s\"}"}
    
Points to note are:

* id uses the fully-qualified service name - com.HailoOSS.service.allocation
* path is '/' separated and does *not* include the 'config' prefix that will be used to read the config
* escape the '"'s in the config JSON (but not '{' etc.)

You can use the update method to create a config, if the service can't find the id it will create it:

    execute update {"id": "H2:BASE:com.HailoOSS.service.allocation", "path": "", "message": "Hope to hell this works", "config": "{}" }

Assuming this is sent to the test environment it can be read back as follows:

    curl -sS https://h2-config-test.elasticride.com/compile?ids=H2:BASE,H2:BASE:com.HailoOSS.service.allocation,H2:REGION:eu-west-1,H2:REGION:eu-west-1:com.HailoOSS.service.allocation \
    | jq -r '.config.hailo.service.allocation'

which will return:

    {
      "expiryTime": "75s",
      "cycleTime": "10s"
    }

## HTTP interface

The config service establishes an HTTP server running on port **8097**.

### /

The index resource acts as a sanity check.

    curl localhost:8097
    {"about":"com.HailoOSS.service.config","docs":"github.com/HailoOSS/config-service","version":20130624113616}

### /compile?ids=a,b,c&path=foo.bar.baz

Constructs compiled config.

    curl localhost:8097/compile?ids=H2:BASE | jq . -M
    {
      "hash": "85df333770e5e952a851541ddc82af8b",
      "config": {
        "hailo": {
          "service": {
            "zookeeper": {
              "recvTimeout": "200ms",
              "hosts": [
                "localhost:12181"
              ]
            },
        ... snipped ...

## Next steps

  - add schema validation against JSON schema definitions, where appropriate
