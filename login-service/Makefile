# The name of the image to create. By default will use the name of the repo. e.g
# config-service
image?=$(notdir ${PWD})

# The docker network to connect to when running the service locally
network?=h2_default

container_name?=$(image)

# The environment flags to pass to the docker run command: This allows you
# to set any environment variables needed for a container
run_env_flags?=H2_CONFIG_SERVICE_ADDR=http://config-service:8097 H2_CONFIG_SERVICE_CASSANDRA=cassandra:9160 BOXEN_RABBITMQ_URL=amqp://hailo:hailo@rabbitmq:5672
run_flags = $(patsubst %,-e %,$(run_env_flags))

# go env flags
go_build_flags=GOOS=linux GOOARCH=amd64 CGO_ENABLED=0

all : clean build run

.PHONY : all

clean :
	-docker rm $(container_name)

test :
	go test -v -race $$(glide novendor)

build :
	$(go_build_flags) go build -v
	docker build -t $(image) .
	docker run --rm $(image) /$(image) -name 2>/dev/null | grep 'com.HailoOSS' || exit 1

run :
	docker run -it --rm --name $(container_name) \
		$(run_flags) \
		--net=$(network) \
		--volumes-from data \
		$(image)
