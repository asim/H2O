package main

import (
	"flag"
	"os"
	"time"

	log "github.com/cihub/seelog"

	"github.com/hailo-platform/H2O/api-proxy/controlplane"
	"github.com/hailo-platform/H2O/api-proxy/handler"
	hc "github.com/hailo-platform/H2O/api-proxy/healthcheck"
	"github.com/hailo-platform/H2O/api-proxy/stats"
	service "github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/service/config"
	"github.com/hailo-platform/H2O/service/zookeeper"
)

var (
	accessLogName string
)

func init() {
	service.Name = "com.hailocab.hailo-2-api"
	service.Description = "Routing layer that handles all inbound client requests, routing them to H1, H2, or " +
		"throttling them."
	service.Version = ServiceVersion
	service.Source = "github.com/hailo-platform/H2O/api-proxy"
	service.OwnerEmail = "oliver.beattie@hailo-platform/H2O.com"
	service.OwnerMobile = "+447584048620"
	service.OwnerTeam = "h2o"

	// Before we proceed, load the "last known good" config
	controlplane.LoadLastGoodConfig()

	flag.StringVar(&accessLogName, "accesslog", "access_log", "The location where Apache-style logs should be written")
	service.Init()
}

func main() {
	config.WaitUntilLoaded(2 * time.Second)

	accessLog, err := os.OpenFile(accessLogName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Criticalf("Can't create access log:	", err)
		os.Exit(6)
	}
	defer accessLog.Close()

	// Register stats collection
	stats.Init(service.Name, ServiceVersion, service.InstanceID)
	stats.Register("/rpc")
	stats.Register("/v2/h2/call")
	stats.Register("/v1/point/batch")
	stats.Register("/")
	stats.Start()

	server := handler.NewServer(accessLog)

	// Initalise ZK and wait for us to get up and running (for status monitor)
	zookeeper.WaitForConnect(time.Second)

	// Initalise healthchecks
	hc.Init(server.Monitor)
	hc.RegisterErrorRateCheck(service.Name+".http.root", "/", 0.05)
	hc.RegisterErrorRateCheck(service.Name+".http.v2-h2-call", "/v2/h2/call", 0.25)
	hc.RegisterErrorRateCheck(service.Name+".http.rpc", "/rpc", 0.25)
	hc.RegisterErrorRateCheck(service.Name+".http.v1-point-batch", "/v1/point/batch", 0.10)
	hc.RegisterDropOffCheck(service.Name+".http.root.request-rate", "/", 0.5)

	log.Infof("Listening at %v", server.HTTPServer.Addr)
	log.Critical(server.HTTPServer.ListenAndServe())
}
