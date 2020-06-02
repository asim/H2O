package handler

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/HailoOSS/binding-service/binding"
	subscribetopic "github.com/HailoOSS/binding-service/proto/subscribetopic"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/raven"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
	"net/http"
)

// Subscribe a queue to a topic
// Just need to create a binding with routing key set as the topic
func SubscribeTopicHandler(req *server.Request) (proto.Message, errors.Error) {
	request := &subscribetopic.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.kernel.binding.subscribetopic", err.Error())
	}
	queue := request.GetQueue()
	topic := request.GetTopic()

	log.Debug("Subscribing queue to topic ", request)

	httpClient := &http.Client{}

	err := binding.CreateTopicBindingE2Q(httpClient, binding.LocalHost+":"+binding.DefaultRabbitPort, raven.TOPIC_EXCHANGE, queue, topic)
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.kernel.binding.setupservice", fmt.Sprintf("Error while creating E2Q binding h2o.topic -> %v. %v", queue, err))
	}

	return &subscribetopic.Response{Ok: proto.Bool(true)}, nil

}
