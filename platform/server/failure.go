package server

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	log "github.com/cihub/seelog"

	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/service/nsq"

	fproto "github.com/hailo-platform/H2O/platform/proto/failure"
	"github.com/hailo-platform/H2O/protobuf/proto"
)

var (
	PublishErrors = true // Services default to publishing errors to NSQ
	errorTopic    = "errors"
)

// publishFailure publishes a failure/panic event to be monitored.
func publishFailure(r interface{}) {
	var p string

	switch r.(type) {
	case string:
		p = r.(string)
	case error:
		p = fmt.Sprintf("%v", r.(error))
	default:
		p = "Unknown panic"
	}

	b := make([]byte, 1024)
	runtime.Stack(b, true)

	if err := client.Pub("com.hailo-platform/H2O.monitor.failure", &fproto.Failure{
		ServiceName:    proto.String(Name),
		ServiceVersion: proto.Uint64(Version),
		AzName:         proto.String(az),
		Hostname:       proto.String(hostname),
		InstanceId:     proto.String(InstanceID),
		Timestamp:      proto.Int64(time.Now().Unix()),
		Uptime:         proto.Int64(int64(time.Since(serviceStarted).Seconds())),
		Type:           proto.String("PANIC"),
		Reason:         proto.String(p),
		Stack:          proto.String(string(b)),
	}); err != nil {
		log.Errorf("[Server] Failed to publish failure event: %v", err)
	}
}

// publishError publishes an event when a handler returns an error
func publishError(req *Request, e errors.Error) {
	if !PublishErrors {
		return
	}

	stacktrace := ""
	if e.MultiStack() != nil {
		stacktrace = e.MultiStack().String()
	}

	application := ""
	if req.Auth().IsAuth() && req.Auth().AuthUser() != nil {
		application = req.Auth().AuthUser().Application()
	}
	userId := ""
	if req.Auth().IsAuth() && req.Auth().AuthUser() != nil {
		userId = req.Auth().AuthUser().Id
	}

	msg := map[string]interface{}{
		"created":     time.Now(),
		"service":     Name,
		"version":     Version,
		"azName":      az,
		"hostname":    hostname,
		"instanceId":  InstanceID,
		"error":       e.Error(),
		"type":        e.Type(),
		"code":        e.Code(),
		"description": e.Description(),
		"httpCode":    e.HttpCode(),
		"context":     e.Context(),
		"userId":      userId,
		"application": application,
		"traceId":     req.TraceID(),
		"remoteAddr":  req.RemoteAddr(),
		"stacktrace":  stacktrace,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("[Server] Failed to JSON encode error event: %v", err)
	}
	if err = nsq.Publish(errorTopic, payload); err != nil {
		log.Errorf("[Server] Failed to publish error event: %v", err)
	}
}
