package server

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/hailo-platform/H2O/protobuf/proto"
	"github.com/nu7hatch/gouuid"

	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/healthcheck"
	pllogs "github.com/hailo-platform/H2O/platform/logs"
	"github.com/hailo-platform/H2O/platform/raven"
	"github.com/hailo-platform/H2O/platform/stats"
	plutil "github.com/hailo-platform/H2O/platform/util"
	"github.com/hailo-platform/H2O/service/config"
	"github.com/hailo-platform/H2O/service/config/service_loader"
	slhc "github.com/hailo-platform/H2O/service/healthcheck"
	inst "github.com/hailo-platform/H2O/service/instrumentation"
	ssync "github.com/hailo-platform/H2O/service/sync"

	healthproto "github.com/hailo-platform/H2O/platform/proto/healthcheck"
	jsonschemaproto "github.com/hailo-platform/H2O/platform/proto/jsonschema"
	loadedconfigproto "github.com/hailo-platform/H2O/platform/proto/loadedconfig"
	profilestartproto "github.com/hailo-platform/H2O/platform/proto/profilestart"
	profilestopproto "github.com/hailo-platform/H2O/platform/proto/profilestop"
	statsproto "github.com/hailo-platform/H2O/platform/proto/stats"
)

type Options struct {
	SelfBind bool // bind self in rabbit
	Die      bool // don't die when discovery heartbeating fails
}

var (
	// Name is the name of the service such as com.hailocab.example
	Name string
	// Description is the human readable version for the Name
	Description string
	// Version is the timestamp of the release
	Version uint64
	// Source is the URL of the source code, eg: github.com/hailo-platform/H2O/foo
	Source string
	// OwnerEmail is the email address of the person who is responsible for this service
	OwnerEmail string
	// OwnerMobile is the international mobile number of the person responsible, eg: +44791111111
	OwnerMobile string
	// OwnerTeam is the name of the team who is responsible for this service, e.g. platform
	OwnerTeam string
	// InstanceID is the unique id of this running instance
	InstanceID string
	// ConcurrentRequests is how many concurrent requests we want to serve per calling service
	ConcurrentRequests int = 1000
)

// Unexported variables
var (
	reg                  *registry
	dsc                  *discovery
	postConnHdlrs        []PostConnectHandler
	cleanupHdlrs         []CleanupHandler
	hostname, az         string
	initialised          bool
	configDir            string
	serviceStarted       time.Time
	commonLogger         io.WriteCloser
	serviceToServiceAuth = true
	tokens               map[string]chan bool // Per calling service
	tokensMtx            sync.RWMutex
	requestsWg           sync.WaitGroup
	inFlightRequests     uint64 = 0
)

const (
	// The amount of time we wait for requests to finish when the service has
	// been interrupted.
	requestsWaitTimeout = time.Second * 60
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	postConnHdlrs = make([]PostConnectHandler, 0)
	cleanupHdlrs = make([]CleanupHandler, 0)

	var err error
	hostname, err = os.Hostname()
	if err != nil {
		log.Criticalf("[Server] Unable to determine hostname: %v", err)
		panic(err)
	}
	az, _ = plutil.GetAwsAZName()
}

// tokensChan returns the appropriate token channel for the passed calling service
func tokensChan(callingService string) chan bool {
	tokensMtx.RLock()
	result, ok := tokens[callingService]
	tokensMtx.RUnlock()

	if !ok {
		newChan := make(chan bool, ConcurrentRequests)
		for i := 0; i < ConcurrentRequests; i++ {
			newChan <- true
		}

		tokensMtx.Lock()
		result, ok = tokens[callingService] // Prevent overwriting if someone else got the lock before us
		if !ok {
			tokens[callingService] = newChan
			result = newChan
		}
		tokensMtx.Unlock()
	}

	return result
}

// monitorRaven monitors our raven connection status and just logs it atm
func monitorRaven(ch chan bool) {
	for {
		select {
		case status := <-ch:
			log.Warnf("[Server] Raven connection status: %v", status)
		}
	}
}

// Init is a local init call that handles setup
func Init() {
	// Parse flags and handle them. No other code should be calling flag.Parse()
	handleFlags()

	if len(Name) == 0 {
		log.Critical("[Server] No service name found")
		cleanupLogs()
		os.Exit(1)
	}

	// GO!
	log.Infof("[Server] Starting up %v (%v)", Name, Version)

	rand.Seed(time.Now().UnixNano())
	tokens = make(map[string]chan bool, 100)

	log.Infof("[Server] Running with %d cores", runtime.GOMAXPROCS(0))
	uuid, _ := uuid.NewV4()
	InstanceID = fmt.Sprintf("server-%s-%s", Name, uuid)

	// Setup service logs
	pllogs.LoadServiceConfig(Name)
	commonLogger = pllogs.CreateAccessLogger(Name)

	// Configure key service layer components, loading config from the config service automatically
	service_loader.Init(Name)
	inst.SetNamespace(Name)
	ssync.SetRegionLockNamespace(Name)

	// Register region leader cleanup function
	RegisterCleanupHandler(ssync.CleanupRegionLeaders)

	// Ping graphite that we have started
	inst.Counter(1.0, "runtime.started", 1)

	// Connect the raven and keep checking its status
	ch := raven.Connect()
	if online := <-ch; !online {
		log.Warn("[Server] Failed to connect the raven on first attempt")
	}
	go monitorRaven(ch)

	// Create a new registry for the endpoints
	reg = newRegistry()

	// Add default middleware
	registerMiddleware(authMiddleware)
	registerMiddleware(tracingMiddleware)
	registerMiddleware(instrumentedMiddleware)
	registerMiddleware(tokenConstrainedMiddleware)
	registerMiddleware(waitGroupMiddleware)
	registerMiddleware(commonLoggerMiddleware(commonLogger))

	// Add default endpoints
	registerEndpoint(&Endpoint{
		Name:             "health",
		Mean:             100,
		Upper95:          200,
		Handler:          healthHandler,
		RequestProtocol:  new(healthproto.Request),
		ResponseProtocol: new(healthproto.Response),
	})
	registerEndpoint(&Endpoint{
		Name:             "stats",
		Mean:             100,
		Upper95:          200,
		Handler:          statsHandler,
		RequestProtocol:  new(statsproto.Request),
		ResponseProtocol: new(statsproto.PlatformStats),
	})
	registerEndpoint(&Endpoint{
		Name:             "loadedconfig",
		Mean:             100,
		Upper95:          200,
		Handler:          loadedConfigHandler,
		RequestProtocol:  new(loadedconfigproto.Request),
		ResponseProtocol: new(loadedconfigproto.Response),
	})
	registerEndpoint(&Endpoint{
		Name:             "jsonschema",
		Mean:             100,
		Upper95:          200,
		Handler:          jsonschemaHandler,
		RequestProtocol:  new(jsonschemaproto.Request),
		ResponseProtocol: new(jsonschemaproto.Response),
		Authoriser:       OpenToTheWorldAuthoriser(),
	})
	reg.add(&Endpoint{
		Name:             "profilestart",
		Mean:             100,
		Upper95:          200,
		Handler:          profileStartHandler,
		RequestProtocol:  new(profilestartproto.Request),
		ResponseProtocol: new(profilestartproto.Response),
	})
	reg.add(&Endpoint{
		Name:             "profilestop",
		Mean:             100,
		Upper95:          200,
		Handler:          profileStopHandler,
		RequestProtocol:  new(profilestopproto.Request),
		ResponseProtocol: new(profilestopproto.Response),
	})

	// Initialise platform healthchecks
	initHealthChecks()
	initialised = true
}

// Register an endpoint with the registry
func Register(eps ...*Endpoint) (err error) {
	if !initialised {
		log.Critical("Server must be initialised before you can register an endpoint")
		cleanupLogs()
		os.Exit(2)
	}

	for _, ep := range eps {
		if err = registerEndpoint(ep); err != nil {
			log.Critical("Error registering endpoint, %v: %v", ep.Name, err)
			log.Flush()
			os.Exit(2)
		}

		log.Infof("[Server] Registered endpoint: %s", ep.Name)
	}

	return nil
}

func registerEndpoint(ep *Endpoint) error {
	return reg.add(ep)
}

func RegisterMiddleware(mws ...Middleware) (err error) {
	for _, mw := range mws {
		if err = registerMiddleware(mw); err != nil {
			return err
		}
	}

	return nil
}

func registerMiddleware(mw Middleware) error {
	return reg.addMiddleware(mw)
}

// HealthCheck registers a standard health check for this server
func HealthCheck(id string, checker slhc.Checker) {
	PriorityHealthCheck(id, checker, healthcheck.StandardPriority)
}

// PriorityHealthCheck is a healthcheck with a configurable priority for this server
func PriorityHealthCheck(id string, checker slhc.Checker, priority slhc.Priority) {
	hc := &healthcheck.HealthCheck{
		Id:             id,
		ServiceName:    Name,
		ServiceVersion: Version,
		Hostname:       hostname,
		InstanceId:     InstanceID,
		Interval:       healthcheck.StandardInterval,
		Checker:        checker,
		Priority:       priority,
	}

	// Allow healthcheck parameters to be overridden in config
	config.AtPath("hailo", "platform", "healthcheck", id).AsStruct(hc)

	healthcheck.Register(hc)
}

// HandleRequest and send back response
func HandleRequest(req *Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Criticalf("[Server] Panic \"%v\" when handling request: (id: %s, endpoint: %s, content-type: %s,"+
				" content-length: %d)", r, req.MessageID(), req.Destination(), req.delivery.ContentType,
				len(req.delivery.Body))
			inst.Counter(1.0, "runtime.panic", 1)
			publishFailure(r)
			debug.PrintStack()
		}
	}()

	if len(req.Service()) > 0 && req.Service() != Name {
		log.Criticalf(`[Server] Message meant for "%s" not "%s"`, req.Service(), Name)
		return
	}

reqProcessor:
	switch {
	case req.isHeartbeat():
		if dsc.IsConnected() {
			log.Tracef("[Server] Inbound heartbeat from: %s", req.ReplyTo())
			dsc.hb.beat()
			raven.SendResponse(PongResponse(req), InstanceID)
		} else {
			log.Warnf("[Server] Not connected but heartbeat from: %s", req.ReplyTo())
		}

	case req.IsPublication():
		log.Tracef("[Server] Inbound publication on topic: %s", req.Topic())

		if endpoint, ok := reg.find(req.Topic()); ok { // Match + call handler
			if data, err := endpoint.unmarshalRequest(req); err != nil {
				log.Warnf("[Server] Failed to unmarshal published message: %s", err.Error())
				break reqProcessor
			} else {
				req.unmarshaledData = data
			}

			if _, err := endpoint.Handler(req); err != nil {
				// don't do anything on error apart from log - it's a pub sub call so no response required
				log.Warnf("[Server] Failed to process published message: %v", err)
			}
		}

	default:
		log.Tracef("[Server] Inbound message %s from %s", req.MessageID(), req.ReplyTo())

		// Match a handler
		endpoint, ok := reg.find(req.Endpoint())
		if !ok {
			if rsp, err := ErrorResponse(req, errors.InternalServerError("com.hailocab.kernel.handler.missing", fmt.Sprintf("No handler registered for %s", req.Destination()))); err != nil {
				log.Criticalf("[Server] Unable to build response: %v", err)
			} else {
				raven.SendResponse(rsp, InstanceID)
			}
			return
		}

		// Unmarshal the request data
		var (
			reqData, rspData proto.Message
			err              errors.Error
		)
		if reqData, err = endpoint.unmarshalRequest(req); err == nil {
			req.unmarshaledData = reqData
		}

		// Call handler if no errors so far
		if err == nil {
			rspData, err = endpoint.Handler(req)
		}

		// Check response type matches what's registered
		if err == nil && rspData != nil {
			_, rspProtoT := endpoint.ProtoTypes()
			rspDataT := reflect.TypeOf(rspData)
			if rspProtoT != nil && rspProtoT != rspDataT {
				err = errors.InternalServerError("com.hailocab.kernel.server.mismatchedprotocol",
					fmt.Sprintf("Mismatched response protocol. %s != %s", rspDataT.String(), rspProtoT.String()))
			}
		}

		if err != nil {
			switch err.Type() {
			case errors.ErrorBadRequest, errors.ErrorForbidden, errors.ErrorNotFound:
				log.Debugf("[Server] Handler error %s calling %v.%v from %v: %v", err.Type(), req.Service(),
					req.Endpoint(), req.From(), err)
			case errors.ErrorInternalServer:
				go publishError(req, err)
				fallthrough
			default:
				log.Errorf("[Server] Handler error %s calling %v.%v from %v: %v", err.Type(), req.Service(),
					req.Endpoint(), req.From(), err)
			}

			if rsp, err := ErrorResponse(req, err); err != nil {
				log.Criticalf("[Server] Unable to build response: %v", err)
			} else {
				raven.SendResponse(rsp, InstanceID)
			}

			return
		}

		if rsp, err := ReplyResponse(req, rspData); err != nil {
			if rsp, err2 := ErrorResponse(req, errors.InternalServerError("com.hailocab.kernel.marshal.error", fmt.Sprintf("Could not marshal response %v", err))); err2 != nil {
				log.Criticalf("[Server] Unable to build error response: %v", err2)
			} else { // Send the error response
				raven.SendResponse(rsp, InstanceID)
			}
		} else { // Send the succesful response
			raven.SendResponse(rsp, InstanceID)
		}
	}
}

// BindAndRun enables self-binding for this service
func BindAndRun() {
	doRun(&Options{
		SelfBind: true,
		Die:      true,
	})
}

// Run the server by listening for messages
func Run() {
	doRun(&Options{
		SelfBind: false,
		Die:      true,
	})
}

// RunWithOptions starts the server with user defined options
func RunWithOptions(opts *Options) {
	doRun(opts)
}

// cleanup is called when exiting
func cleanup() {
	// Shutdown notification to monitoring service
	stats.Stop()

	// disconnecting from discovery service
	dsc.disconnect()

	waitRequests()

	// run some cleanup handlers
	for _, f := range cleanupHdlrs {
		f()
	}

	cleanupLogs()
}

func cleanupLogs() {
	log.Flush()
	if commonLogger != nil {
		commonLogger.Close()
	}
}

func waitRequests() {
	raven.Disconnect()

	waitdone := make(chan struct{})
	go func() {
		defer close(waitdone)
		requestsWg.Wait()
	}()

	select {
	case <-waitdone:
		log.Debugf("All requests finished")
	case <-time.After(requestsWaitTimeout):
		log.Warnf("Giving up waiting for outstanding requests")
	}
}

func doRun(opts *Options) {
	defer cleanupLogs()

	// check we have some endpoints
	if reg.size() == 0 {
		log.Critical("There are no endpoints for this service")
		os.Exit(3)
	}

	// start listening for incoming messages
	deliveries, err := raven.Consume(InstanceID)
	if err != nil {
		log.Critical("[Server] Failed to consume: %v", err)
		os.Exit(5)
	}

	if opts.SelfBind {
		// binding should come after you've started consuming
		if err := raven.BindService(Name, InstanceID); err != nil {
			log.Criticalf("[Server] Failed to bind itself: %v", err)
			os.Exit(7)
		}
	}

	// announce ourselves to the discovery service
	dsc = newDiscovery(opts)
	go dsc.connect()

	// run some post connect handlers
	for _, f := range postConnHdlrs {
		go f()
	}

	// register stats collector
	go registerStats()

	// listen for SIGQUIT
	go signalCatcher()

	// consume messages
	for d := range deliveries {
		req := NewRequestFromDelivery(d)
		go HandleRequest(req)
	}

	log.Critical("[Server] Stopping due to channel closing")
	os.Exit(6)
}

// ScopedRequest returns a client request, prepared with scoping information
// of the service _making_ the call - such that service-to-service auth can work
func ScopedRequest(service, endpoint string, payload proto.Message) (*client.Request, error) {
	r, err := client.NewRequest(service, endpoint, payload)
	if err != nil {
		return nil, err
	}
	// scope -- who WE are
	r.SetFrom(Name)

	// set request as already authorised
	r.SetAuthorised(true)

	return r, nil
}

func signalCatcher() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1)
	defer signal.Stop(c)

	for sig := range c {
		if sig == syscall.SIGUSR1 {
			go pllogs.EnableTrace()
		} else {
			log.Infof("[Service] Received signal: %s", sig.String())
			cleanup()
			if sig == syscall.SIGQUIT { // Print stack dump
				buf := make([]byte, 1<<16)
				os.Stderr.Write([]byte("SIGQUIT: core dump\n\n"))
				os.Stderr.Write(buf[:runtime.Stack(buf, true)])
			}
			break
		}
	}

	os.Exit(2)
}

// DisableServiceToServiceAuth stops the service attempting to load auth rules
// for service-to-service calls where it is not needed.
func DisableServiceToServiceAuth() {
	serviceToServiceAuth = false
}

// Exit provides a wrapper to os.Exit so that we can disconnect, cleanup and
// shutdown in a graceful manner.
func Exit(code int) {
	log.Infof("[Service] Exiting with code %d", code)
	cleanup()
	os.Exit(code)
}
