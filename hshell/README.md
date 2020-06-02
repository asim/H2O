platform-cli

Build:
go get github.com/HailoOSS/platform-cli/hshell

Usage:
hshell flag interface

Example:
    
    hshell -protobuf=github.com/HailoOSS/go-platform-layer/examples/sayhello -endpoint="com.HailoOSS.service.helloworld.sayhello" -request="Name: proto.String(\"Moddie\")"  -update=false -hint=false -default=false

Explain:

-protobuf
    The protobuf to import (as you would put in your go imports)

-endpoint
    The endpoint you want to hit

-request
    The request object to send (try -hint=true to see what this should look like before sending)

-hint
    this will not send anything but will give you an example protobuf object

-update
    This will update the client, protobuf lib and the protobuf file from the network

-default
    If true, sends a default request as parsed from the protos

-h
    Usage info
    
hshell console

Example:

    hshell -cli=true

use the help command for more information about hshell console

Example usage:

>ls 
//lists services
>use {service} 
//sets a service as the service of interest
>ls
//lists endpoints of service of interest
>execute {endpoint} {json content}
//executes the endpoint
>profit
//invalid command
