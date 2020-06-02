# RabbitMQ H2O Exchange

Custom exchange for Hailo 2.0 platform, based on the official headers
exchange distributed with RabbitMQ offering `<` operator for pattern
matching.

We'll also need a shovel/federation type extra process to help with
connecting multiple brokers together.

## Installation

First make sure you have Erlang and RabbitMQ installed.

Install and setup the RabbitMQ Public Umbrella as explained here:
[Plugin Development Guide](http://www.rabbitmq.com/plugin-development.html)

Then type:

	$ git clone https://github.com/rabbitmq/rabbitmq-public-umbrella.git
	$ cd rabbitmq-public-umbrella/
	$ make co
	$ make up_c BRANCH=rabbitmq_v3_1_1
	$ git clone https://github.com/HailoOSS/rabbitmq-hailo-magic-exchange.git
	$ cd rabbitmq-h2o-exchange/
	$ make dist

Finally copy **only** the exchange `*.ez` file inside the `dist/` folder
to `$RABBITMQ_HOME/plugins`.

The version (`3_1_1`) needs to correspond to the version of RabbitMQ used
in the environment you want to deploy to. At the time of writing 3.1.1 is
used in boxen and 3.2.4 is used in stg and lve.

## Usage

Simply declare an exchange of type `x-hailo2`.

## ETS

If you want to understand more detail what's going on in `find_routes`, there's a good chapter in [Learn You Some Erlang on ETS](http://learnyousomeerlang.com/ets).

## Deployment

1. Copy the new **.ez to the box running rabbit-server
2. Move the .ez file to rabbit's plugin folder
3. Stop rabbit-server
4. Delete the old .ez of the same plugin
5. Start rabbit-server
6. Verify if the plugin is enabled and loaded with rabbit-plugins list

Example:
```
scp /usr/local/Cellar/rabbitmq/3.5.1/plugins/rabbitmq_hailo_magic_exchange-0.3.0-rmq0.0.0.ez john.dobronszki@ip-10-21-0-38:~/
# on the box:
john.dobronszki@ip-10-21-0-38:/usr/lib/rabbitmq/lib/rabbitmq_server-3.2.4/plugins$ sudo cp ~/rabbitmq_hailo_magic_exchange-0.3.0-rmq0.0.0.ez ./
john.dobronszki@ip-10-21-0-38:/usr/lib/rabbitmq/lib/rabbitmq_server-3.2.4/plugins$ sudo service rabbitmq-server stop
 * Stopping message broker rabbitmq-server
john.dobronszki@ip-10-21-0-38:/usr/lib/rabbitmq/lib/rabbitmq_server-3.2.4/plugins$ sudo rm rabbitmq_hailo_magic_exchange-0.2.0-rmq0.0.0.ez
john.dobronszki@ip-10-21-0-38:/usr/lib/rabbitmq/lib/rabbitmq_server-3.2.4/plugins$ sudo service rabbitmq-server start
 * Starting message broker rabbitmq-server
john.dobronszki@ip-10-21-0-38:/usr/lib/rabbitmq/lib/rabbitmq_server-3.2.4/plugins$ rabbitmq-plugins list
[e] amqp_client                       3.2.4
[ ] cowboy                            0.5.0-rmq3.2.4-git4b93c2d
[ ] eldap                             3.2.4-gite309de4
[e] mochiweb                          2.7.0-rmq3.2.4-git680dba8
[ ] rabbitmq_amqp1_0                  3.2.4
[ ] rabbitmq_auth_backend_ldap        3.2.4
[ ] rabbitmq_auth_mechanism_ssl       3.2.4
[ ] rabbitmq_consistent_hash_exchange 3.2.4
[ ] rabbitmq_federation               3.2.4
[ ] rabbitmq_federation_management    3.2.4
[E] rabbitmq_hailo_magic_exchange     0.3.0-rmq0.0.0
[ ] rabbitmq_jsonrpc                  3.2.4
[ ] rabbitmq_jsonrpc_channel          3.2.4
[ ] rabbitmq_jsonrpc_channel_examples 3.2.4
[E] rabbitmq_management               3.2.4
[e] rabbitmq_management_agent         3.2.4
[ ] rabbitmq_management_visualiser    3.2.4
[ ] rabbitmq_mqtt                     3.2.4
[ ] rabbitmq_shovel                   3.2.4
[ ] rabbitmq_shovel_management        3.2.4
[ ] rabbitmq_stomp                    3.2.4
[ ] rabbitmq_tracing                  3.2.4
[e] rabbitmq_web_dispatch             3.2.4
[ ] rabbitmq_web_stomp                3.2.4
[ ] rabbitmq_web_stomp_examples       3.2.4
[ ] rfc4627_jsonrpc                   3.2.4-git5e67120
[ ] sockjs                            0.3.4-rmq3.2.4-git3132eb9
[e] webmachine                        1.10.3-rmq3.2.4-gite9359c7
```

If in doubt, here are a couple of useful resources:

https://www.rabbitmq.com/relocate.html
https://www.rabbitmq.com/plugins.html 
