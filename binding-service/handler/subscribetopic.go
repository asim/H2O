package handler

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/binding-service/binding"
	subscribetopic "github.com/hailo-platform/H2O/binding-service/proto/subscribetopic"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/raven"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
	"net/http"
)

// Subscribe a queue to a topic
// Just need to create a binding with routing key set as the topic
func SubscribeTopicHandler(req *server.Request) (proto.Message, errors.Error) {
	request := &subscribetopic.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.hailo-platform/H2O.kernel.binding.subscribetopic", err.Error())
	}
	queue := request.GetQueue()
	topic := request.GetTopic()

	log.Debug("Subscribing queue to topic ", request)

	httpClient := &http.Client{}

	err := binding.CreateTopicBindingE2Q(httpClient, binding.LocalHost+":"+binding.DefaultRabbitPort, raven.TOPIC_EXCHANGE, queue, topic)
	if err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.kernel.binding.setupservice", fmt.Sprintf("Error while creating E2Q binding h2o.topic -> %v. %v", queue, err))
	}

	return &subscribetopic.Response{Ok: proto.Bool(true)}, nil

}
