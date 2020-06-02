# Login service

H2 login service. Has two responsibilities:

 - session store for active user sessions
 - general purpose credential store

Any Hailo application can leverage the login service to store user account details
with password management features.

## Install

If you're using Boxen, and have H2 installed:

    go get github.com/HailoOSS/login-service
    cat config/cassandra.boxen | cassandra-cli -p 19160

You can setup a default user thus:

	cd bootstrap
	go build
	./bootstrap

The Go login service has private key location hard-coded, so you should set
this up:

	mkdir /opt/hailo/login-service
	cp ~/src/login-service/config/boxen/*-key /tmp/login-service/

Create a user that you can login with:

	curl -d 'service=com.HailoOSS.service.login' -d endpoint='auth' -d 'request={"mech":"h2","deviceType":"cli","username":"admin","password":"Password1","application":"ADMIN"}' http://localhost:8080/v2/h2/call

Take the sessId parameter returned by this call and urlencode it for the next step.

A default config so that we know how to call the H1 login service can be 
installed for boxen via the [call API](github.com/HailoOSS/call-api):

	curl -d service=com.HailoOSS.service.config \
		 -d endpoint=update \
		 -d request="{\"id\":\"H2:BASE:com.HailoOSS.service.login\",\"message\":\"Install login config\",\"config\":`cat config/configservice.boxen.json | php -r 'echo json_encode(stream_get_contents(STDIN));'`}" \
		 http://localhost:8080/v2/h2/call?session_id=<url encoded sessId from previous curl>


## Features

### Session store

A **session** consists of a random ID of 160 bytes base64 encoded to a string.
This is what clients should use to identify themselves. The session ID itself
contains no user identifiable information on its own.

Session are a lookup key for a **token**.

Tokens store information about an authenticated user and are signed by a private
key that only the login service has access to (this should be managed by isolated
deployment to secured nodes). Tokens **always expire** after 8 hours.

It is possible for certain tokens to be **automatically renewed** to give the
impression that a user is signed in for longer than 8 hours. This preserves the
same session ID and is transparent to people using sessions/tokens. It is not
possible for a session/token to be extended if it carries any ADMIN roles. In
this situation, users must re-authenticate themselves every 8 hours.

There is a constraint that users can only maintain one active session
per-application per-device type. The **device type** is any arbitrary string
that means an application can maintain two sessions for different use cases.
For example you may have one session on a Hailo web client and another on a
phone. However if you were to try to establish a new session on _another_
phone, the first phone would have its session invalidated.

Sessions are cached locally by clients and thus the login service broadcasts
session expiry globally (via federated NSQ) such that clients can clear down
their cache and hence invalidate tokens faster than the maximum bounded
8 hours. This is an optimisation. There are no cryptographic guarantees of
this -- a session once issued could theoretically be exploited by a 
discrete part of the system for a maximum of 8 hours.

### Credentials store

The [original login service](https://github.com/HailoOSS/login-service) had
a specific system for storing "admin" users (think internal Hailo users who
had access to portals etc.) It _did not_ store customer credentials nor
driver credentials. Instead it had a system whereby different "authentication
mechanisms" could provide different ways to identify users.

With the H2 version we want to take responsibility for the storage of all
credentials.

To maintain backwards compatability we introduce the `h2` authentication
mechanism, which takes the following parameters:

	required string mech = 1;
	required string deviceType = 2;
	required string password = 3;
	optional string username = 4;
	optional string application = 9;

We will use the `application` value to namespace credentials into distinct
stores, where we can reuse a unique identifier (eg: email address) between
different "applications".


