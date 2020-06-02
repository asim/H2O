# API

Global API for "Hailo 2.0" - aka the "thin" API.

Responsible for intercepting _all_ Hailo traffic globally and providing a routing
layer. The key purpose of this layer is to:

 - provide an HTTP interface for H2 (since internally all messages are passed
   over RabbitMQ)
 - provide a layer that can proxy traffic to H1 to allow for a structured
   and controlled transition from H1 to H2
 - provide traffic management facilities for H1 or H2 - like being able to
   shed a % of traffic

Request routing is rule based:

 1. figure out which city the destination request is for - either from an HTTP
    GET or POST parameter (for H2 or customer API) or from the hostname (so
    we offer compatability with H1 driver)
 2. look at the rules for the city in question and see if any match
    (since we have a known bounded set of H1 cities, we will simply setup rules
    for these cities for a proxy outcome)
 3. if there is no match at all, simply send via H2

The way we match requests to an H2 service is based on the path:

	/v1/driver/foo

Calls:

 - service: com.hailocab.api.v1.driver
 - endpoint: foo

### Other features

 - provides RPC direct to H2 via `/rpc`
 - provides functionality for pinning apps to regions, via the `/endpoints` endpoint
   and the `X-H-ENDPOINT-TIMESTAMP` and `X-H-ENDPOINT-FOO` headers and JSON body
   augmentation
 - provides current thin API version number via `/version`
 - serves `favicon.ico` traffic directly
 - provides a healthcheck that can be used to automatically "fail over" an AZ
   via `/status` and `/v2/az/status` endpoints


## Install

Add the service auth rule to api-throttling
    execute grantservice {"endpoint":  {"service": "com.hailocab.service.api-throttling", "endpoint": "checkin", "granted": [ {"name": "com.hailocab.hailo-2-api.throttlesync", "role": "ADMIN" }]}}

You can install a sample config assuming you run the API and have the rest of
the platform kernel running (discovery, binding, config, login):

    curl -d service=com.hailocab.service.config -d endpoint=update -d request="{\"id\":\"H2:BASE:com.hailocab.hailo-2-api\",\"message\":\"Install API config\",\"config\":`cat schema/example.boxen.json | php -r 'echo json_encode(stream_get_contents(STDIN));'`}"     http://localhost:8080/v2/h2/call?session_id=8lJ0Tsds9lIhth3bzzkVYB4zHucviXFnWdaNVbgsNwOIDmHcbnJydieG%2B%2F%2FSiZXheAaXTTTLWZyp9%2Fkk30RQqJNqImN7R4spLxQqu2%2BosJzeVqDMbljb3iHMWli%2BoEgPEFw7QNSJnI59Q35hJxqEsFmHSW17MhYmSm03dsctQTQXUDM0bbH4BKYtNGCG5a%2BllCar7ZgYoDBBp9AjFmHcHA%3D%3D

The default behaviour is simply to send via H2, so unless you need proxying or
throttling, you won't need to install any config.

## Rules

Rules have a `match` and an `action`. Here's a sample:

	{"match":{"path":"/v1/system/ping","proportion":0.5},"action":3}

The `match` object comprises:

 - `path` is the path within the request, eg: `/v1/system/ping` within `http://foo.bar/v1/system/ping`
 - `regulatoryArea` tells us to match one of Hailo's cities - we can match city based on
   the hostname being called (for H1 compatability) or from a parameter (how the customer app does it)
 - `source` is matched as "hostname contains" - and is useful for splitting by customer/driver traffic
   since the hostname contains "customer" or "driver"
 - `proportion` defines a sampling proportion, allowing us to pick a percentage of requests
 - `sampler` defines how we carry out sampling (see below)

The `action` is an enumerated value from:

 1. Proxy request to H1 (1)
 2. Send via H2 (2)
 3. Throttle request (3)
 4. Deprecate request (4)

When handling requests, we process rules in order of "specificity", where we score rules
based on how specific they are with regard to matches.

  - `regulatoryArea` specified = +5 points
  - `source` specified = +5 points
  - `path` specified = +10 points

Thus a rule that specifies a `RegulatoryArea` **and** a `path` will be evaluated first.

	{"match":{"path":"/v1/system/ping","proportion":0.5},"action":3}
	{"match":{"path":"/v1/customer/index","proportion":1.0},"action":1}

### Samplers

There are four available sampling mechanisms (use the numbers in brackets when
defining rules):

  - Random (0)
  - Customer (1)
  - Driver (2)
  - Device (3)

These break down into two main types:

  1. Random - we evaluate sampling purely randomly and the same request would be
  expected to produce different results each time encountered
  2. Hashing - we evaluate sampling by hashing some value from the request and
  the same request would be expected to produce the same results every time it
  is encountered

The hashing sampling is useful for shedding traffic - meaning that some customers,
drivers or devices should "just work" whereas some will find that nothing works.

Example of specifying device sampler in a rule:

	{"match":{"path":"/v1/customer/index","proportion":1.0,"sampler":3},"action":1}


### Reverse proxy to H1

If the `action` is to proxy to H1, then we do the following:

  - prefix the hostname with `v1-`
  - invoke Go's reverse proxy in order to pass the request onto H1

This mechanism relies on us having "destination" hostnames setup for each Hailo
domain, for example `v1-api-driver-london.elasticride.com` to accept the traffic
originally pointed at `api-driver-london.elasticride.com`.

If you try this on your local machine, curling `localhost` then you'll see that it
tries to forward to `v1-localhost` which will almost certainly fail.

Is is possible to override the proxy host, supplying an exact mapping of which
hostname to forward to. This can be achieved by inserting a map of string:string
into the config service under `api.proxyMappings`.

### Sending to H2

If the `action` is to send on to H2, then we do the following:

  - construct a [proto request](https://github.com/hailocab/api-hailo-2/blob/master/proto/api/api.proto)
  that looks very much like an HTTP request
  - automatically map the `path` within the request to an H2 service, by replacing
  `/` with `.` and prefixing with `com.hailocab.api.` -- thus `/v1/system/ping` will go
  to an H2 service named `com.hailocab.api.v1.system` and will call the endpoint within
  this service named `ping`.
  - dispatch the request via Rabbit and then send back the eventual response

You really only need to write **API** services if you wish to handle HTTP parameters
such as GET, POST, PUT etc. If not, you can call H2 services directly via the RPC
endpoint (see below).

### Throttling

If the `action` is to throttle then we don't make any request to either H1 or H2;
instead we just respond with a standard response.

### Deprecating

If the `action` is to deprecate then we treat just like throttling but we track calls
to this endpoint.

## RPC

The thin API has a specific endpoint for executing an RPC call to H2.

	 /rpc

There are two modes for RPC, firstly using **JSON**, which will respond with
a JSON response.

```
curl -d service=com.hailocab.service.login \
  -d endpoint=health \
  -d request="{}" \
  http://localhost:8080/rpc?session_id=8lJ0Tsds9lIhth3bzzkVYB4zHucviXFnWdaNVbgsNwOIDmHcbnJydieG%2B%2F%2FSiZXheAaXTTTLWZyp9%2Fkk30RQqJNqImN7R4spLxQqu2%2BosJzeVqDMbljb3iHMWli%2BoEgPEFw7QNSJnI59Q35hJxqEsFmHSW17MhYmSm03dsctQTQXUDM0bbH4BKYtNGCG5a%2BllCar7ZgYoDBBp9AjFmHcHA%3D%3D
```

Secondly, we can send raw protobuf-encoded bytes and get back raw bytes. When using
proto, we have to send `service` and `endpoint` as query string parameters,
reserving the entire post body for the raw bytes.

```
curl -XPOST \
  -d @/tmp/some-bytes \
  -H 'Content-Type: application/x-protobuf' \
  'http://localhost:8080/rpc?service=com.hailocab.service.idgen&endpoint=cruftflake&session_id=8lJ0Tsds9lIhth3bzzkVYB4zHucviXFnWdaNVbgsNwOIDmHcbnJydieG%2B%2F%2FSiZXheAaXTTTLWZyp9%2Fkk30RQqJNqImN7R4spLxQqu2%2BosJzeVqDMbljb3iHMWli%2BoEgPEFw7QNSJnI59Q35hJxqEsFmHSW17MhYmSm03dsctQTQXUDM0bbH4BKYtNGCG5a%2BllCar7ZgYoDBBp9AjFmHcHA%3D%3D'
```

For backwards compatability we also support `/v2/h2/call` as a path -- this now aliases
`/rpc` and is all contained directly within the thin API (so the "call API" is now deprecated).


## Region pinning

Region pinning is configuration that tells apps a hostname dynamically. The purpose
is that we can pin HOBs to a single region, avoiding large-scale data replication
(eg: federation of all points everywhere) and also allowing for responsiveness in the
event of fail over (plus complete server control).

Apps will ship with a single hostname per-environment, which will continue to use
latency-based DNS to locate the nearest AWS region. This is virtually identical to
how we can currently use api2.elasticride.com. The can use this hostname to find out
which hostnames they should connect to for API requests.

### /endpoints

A new endpoint is provided by the thin API to allow apps to "phone home" and find
out which hostname they should be using:

```
/endpoints?hob=LON&app=customer&version=v3.2.1
```

Where:

 - **hob**: exactly the same as what the apps send in ?city=FOO now; `city` works
   just fine too
 - **app**: what app it is; customer/driver
 - **version**: the version number of the app - not used right now, but we might
   do in the future

This will respond with JSON:

```
{
  "status": true,
  "payload": "OK",
  "endpoints": {
    "timestamp": 1399889758,
    "api": "api-customer-eu.elasticride.com",
    "hms": "h2o-hms-eu.elasticride.com"
  }
}
```

The response tells us what the home region is for a given HOB - so in this instance,
we're getting EU for LON.

### Corrective measures

Once the app is in full flight, happily connected, sending API requests, the
home region might change. As this happens, the next API request made will be
against the wrong region. In order to allow for fast correction and switching,
every API response will have corrective measures baked in.

For JSON content type, we will add in a key named `endpoints`, which will
contain the same information that would be served by `/endpoints`. For example:

```
{
  "status":false,
  "payload":"CustomerInternationalNoService",
  "code":1053,
  "dotted_code":"com.hailocab.service.nearestdriver.search.internationalnoservice",
  "context":[],
  "endpoints": {
    "timestamp": 1399889758,
    "api": "api-customer-eu.elasticride.com",
    "hms": "h2o-hms-eu.elasticride.com"
  }
}
```

In addition, there will be HTTP headers added to the response. These are useful
in two situations: firstly for protobuf content type, and secondly H2 RPC calls.
For both of these, we do not augment the response, and thus the headers are the
only place you can pickup the new hostnames.

```
X-H-ENDPOINT-TIMESTAMP: 12312415
X-H-ENDPOINT-API: api-customer-eu.elasticride.com
X-H-ENDPOINT-HMS: h2o-hms-eu.elasticride.com
```

Whenever these headers of these keys are present, it is the API telling the app
"you are pointing at the wrong place".

### Timestamp

When we update the config, to change the home region for a HOB, we have so far
assumed the updates are globally instantaneous. In reality, we might have some
lag between the updating happening in all regions. We can imagine a failure
scenario where our requests to EU have just updated to tell us that we should
be looking at the US, but when we connect to the US, it still has the old
settings and thus tries to shunt us back to the EU.

Example API request sequence would be:

#### 1. Request made to https://api-customer-eu.elasticride.com

```
{
  "status":false,
  "payload":"CustomerInternationalNoService",
  "code":1053,
  "dotted_code":"com.hailocab.service.nearestdriver.search.internationalnoservice",
  "context":[],
  "endpoints":{
    "timestamp":1398347253,
    "api": "https://api-customer-us.elasticride.com",
    "hms": "https://h2o-hms-us.elasticride.com"
  }
}
```

#### 2. Request made to https://api-customer-us.elasticride.com

```
{
  "status":false,
  "payload":"CustomerInternationalNoService",
  "code":1053,
  "dotted_code":"com.hailocab.service.nearestdriver.search.internationalnoservice",
  "context":[],
  "endpoints":{
    "timestamp":1398260839,
    "api": "https://api-customer-eu.elasticride.com",
    "hms": "https://h2o-hms-eu.elasticride.com"
  }
}
```

In this instance, the timestamp comes into play. The app must compare the
timestamp it is receiving from the config with the most recently read. If
the timestamp is lower, then the config is invalid and must be ignored. In
this case, the second request `endpoint` has a lower timestamp than we already
have, and thus we must ignore it, and keep sending requests to the US.

