# Binding service

This service is part of the magic glue for our broker. It is responsible for:

* Creating bindings for brokers
* Creating bindings for services
* Providing a control plane which controls the routing of messages

## What it does

### Periodic rebinding
Every 90 seconds a binding service will check to make sure that all service bindings are correct. It achieves this by:
1. Query discovery service "instances" endpoint
2. For every service instance in this AZ we set up the bindings on the local rabbit to point to this queue AND we set up bindings on all the other rabbit clusters to point to this AZ
3. Get all the bindings in this cluster that point to remote clusters (list is from rabbit itself). Cross check this with the list from discovery service and delete any that aren't in discovery service. 

### Failover
In failover scenario the binding service ensures that all bindings pointing to the failed AZ are torn down. This means that until the binding service is failed back over, nothing will be bound in the failed AZ e.g. if the AZ is restored and services start reconnecting to the recovered RabbitMQ they will not be bound until the binding service connects. 

## Setting up your machine
You need to do the following to set up your local machine to use binding service

You'll need a file at /etc/h2o/azname the contents of which should be "eu-west-1a" or something. Doesn't actually matter what this is as long as the file is there and it's a single word (no whitespace or funny characters).

You need to install version R15B03 of erlang which can be found at https://www.erlang-solutions.com/downloads/download-erlang-otp . This is required for the magic hailo exchange

Download the hailo magic exchange package from 

    https://github.com/HailoOSS/scratch/blob/master/dom/rabbitmq-hailo-magic-exchange/dist/rabbitmq_hailo_magic_exchange-0.1.0-rmq0.0.0.ez 

Copy this to $RABBIT_HOME/plugins. Then you need to enable the various rabbit plugins we need

    sudo rabbitmq-plugins enable rabbitmq_federation_management 
    sudo rabbitmq-plugins enable rabbitmq_federation  
    sudo rabbitmq-plugins enable rabbitmq_hailo_magic_exchange 
    <restart rabbit>

Rabbitmq exchanges. You'll need 3 - name, type per below

    1. h2o, headers
    2. h2o.topic, topic
    3. h2o.direct, direct

You can set these exchanges up via the web admin tool http://localhost:55672
 
Add a user hailo with password hailo (under admin tab) to RabbitMQ admin page. Then click on this user and set permissions for virtual host "/"
