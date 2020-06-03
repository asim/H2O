package client

import (
	"fmt"
	"os"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/protobuf/proto"
	"github.com/nu7hatch/gouuid"
	"github.com/streadway/amqp"

	"github.com/hailo-platform/H2O/platform/circuitbreaker"
	"github.com/hailo-platform/H2O/platform/errors"
	pe "github.com/hailo-platform/H2O/platform/proto/error"
	traceproto "github.com/hailo-platform/H2O/platform/proto/trace"
	"github.com/hailo-platform/H2O/platform/raven"
	plutil "github.com/hailo-platform/H2O/platform/util"
	inst "github.com/hailo-platform/H2O/service/instrumentation"
	trace "github.com/hailo-platform/H2O/service/trace"

	_ "github.com/hailo-platform/H2O/platform/logs"
)

var (
	loadTimeoutsInterval time.Duration = time.Minute * 30
	// NewDefaultClient is the regular Client constructor, used to create real clients.
	NewDefaultClient func() Client = newClient
	// NewClient is the default client constructor used. Override this during testing to ensure all new clients created
	// will use a mock (ensuring to set it back to DefaultClientConstructor at the conclusion of the tests).
	NewClient func() Client = NewDefaultClient
	// DefaultClient is a client with default options. This can be overridden for testing.
	DefaultClient Client = NewClient()
)

// A client stores the details of a service client
type Client interface {
	// Req sends a request, and marhsals a successful response or returns an error
	Req(req *Request, rsp proto.Message, options ...Options) errors.Error

	// CustomReq is similar to Req, but without the automatic response unmarshaling, and instead returning a response
	CustomReq(req *Request, options ...Options) (*Response, errors.Error)

	// Push sends a request where we do not wish to wait for a REP (but is still REQ/REP pattern)
	Push(req *Request) error

	// AsyncTopic sends a pub/sub message (a Publication)
	AsyncTopic(pub *Publication) error

	// Pub created and sends a Publication (via AsyncTopic) in one handy step
	Pub(topic string, payload proto.Message) error
}

type client struct {
	sync.RWMutex
	instanceID string
	responses  *inflight
	defaults   Options
	listening  bool
	timeout    *Timeout
	hostname   string
	az         string
}

// Options to send with a client request
type Options map[string]interface{}

// newClient initialises a new Client
func newClient() Client {
	c := &client{}

	uuid, _ := uuid.NewV4()
	c.instanceID = "client-" + uuid.String()
	c.responses = &inflight{m: make(map[string]chan *Response)}
	c.defaults = Options{"retries": 2}
	c.timeout = NewTimeout(c)

	c.hostname, _ = os.Hostname()
	c.az, _ = plutil.GetAwsAZName()

	return c
}

// Req is a wrapper around DefaultClient.Req
func Req(req *Request, rsp proto.Message, options ...Options) errors.Error {
	return DefaultClient.Req(req, rsp, options...)
}

// CustomReq is a wrapper around DefaultClient.CustomReq
func CustomReq(req *Request, options ...Options) (*Response, errors.Error) {
	return DefaultClient.CustomReq(req, options...)
}

// Push is a wrapper around DefaultClient.Push
func Push(req *Request) error {
	return DefaultClient.Push(req)
}

// AsyncTopic is a wrapper around DefaultClient.AsyncTopic
func AsyncTopic(pub *Publication) error {
	return DefaultClient.AsyncTopic(pub)
}

// Pub is a wrapper around DefaultClient.Pub
func Pub(topic string, payload proto.Message) error {
	return DefaultClient.Pub(topic, payload)
}

func (c *client) listen(ch chan bool) {
	c.Lock()
	defer c.Unlock()

	// check if we started listening while locked
	if c.listening {
		ch <- true
		return
	}

	if deliveries, err := raven.Consume(c.instanceID); err != nil {
		log.Criticalf("[Client] Failed to consume: %v", err)
		c.listening = false
		ch <- false
	} else {
		log.Debugf("[Client] Listening on %s", c.instanceID)
		c.listening = true
		ch <- true
		c.Unlock()
		for d := range deliveries {
			log.Tracef("[Client] Inbound message %s on %s", d.CorrelationId, c.instanceID)
			go c.getResponse(d)
		}
		c.Lock()
		log.Errorf("[Client] Stopping listening due to channel closing")
		c.listening = false
	}
}

func (c *client) getResponse(d amqp.Delivery) {
	rsp := newResponseFromDelivery(d)

	if len(rsp.CorrelationID()) == 0 {
		log.Errorf("[Client] No correlation id")
		return
	}

	if rc, ok := c.responses.get(rsp); ok {
		rc <- rsp
		close(rc)
	} else {
		log.Errorf("[Client] Missing message return queue for %s", rsp.CorrelationID())
	}
}

func (c *client) Req(req *Request, rsp proto.Message, options ...Options) errors.Error {
	// if no options supplied, lookup request options
	if len(options) == 0 {
		options = []Options{req.GetOptions()}
	}

	c.traceReq(req)
	t := time.Now()
	responseMsg, err := c.doReq(req, options...)
	if err != nil {
		errors.Track(err.Code(), req.From(), req.Service(), req.Endpoint())
		return err
	}
	if responseMsg == nil {
		return errors.InternalServerError("com.hailocab.kernel.platform.nilresponse", "Nil response")
	}
	c.traceRsp(req, responseMsg, err, time.Now().Sub(t))

	if marshalError := responseMsg.Unmarshal(rsp); marshalError != nil {
		return errors.InternalServerError("com.hailocab.kernel.platform.unmarshal", marshalError.Error())
	}

	return nil
}

func (c *client) CustomReq(req *Request, options ...Options) (*Response, errors.Error) {
	c.traceReq(req)
	t := time.Now()
	rsp, err := c.doReq(req, options...)
	c.traceRsp(req, rsp, err, time.Now().Sub(t))
	return rsp, err
}

// doReq sends a request, with timeout options and retries, waits for response and returns it
func (c *client) doReq(req *Request, options ...Options) (*Response, errors.Error) {

	if circuitbreaker.Open(req.service, req.endpoint) {
		inst.Counter(1.0, fmt.Sprintf("client.error.%s.%s.circuitbroken", req.service, req.endpoint), 1)
		log.Warnf("Broken Circuit for %s.%s", req.service, req.endpoint)
		return nil, errors.CircuitBroken("com.hailocab.kernel.platform.circuitbreaker", "Circuit is open")
	}

	retries := c.defaults["retries"].(int)
	var timeout time.Duration
	timeoutSupplied := false
	if len(options) == 1 {
		if _, ok := options[0]["retries"]; ok {
			retries = options[0]["retries"].(int)
		}
		if _, ok := options[0]["timeout"]; ok {
			timeout = options[0]["timeout"].(time.Duration)
			timeoutSupplied = true
		}
	}

	// setup the response channel
	rc := make(chan *Response, retries)
	c.responses.add(req, rc)
	defer c.responses.removeByRequest(req)

	instPrefix := fmt.Sprintf("client.%s.%s", req.service, req.endpoint)
	tAllRetries := time.Now()

	for i := 1; i <= retries+1; i++ {
		t := time.Now()

		c.RLock()
		con := c.listening
		c.RUnlock()
		if !con {
			log.Debug("[Client] not yet listening, establishing now...")
			ch := make(chan bool)
			go c.listen(ch)
			if online := <-ch; !online {
				log.Error("[Client] Listener failed")
				inst.Timing(1.0, fmt.Sprintf("%s.error", instPrefix), time.Since(t))
				inst.Counter(1.0, "client.error.com.hailocab.kernel.platform.client.listenfail", 1)
				return nil, errors.InternalServerError("com.hailocab.kernel.platform.client.listenfail", "Listener failed")
			}

			log.Info("[Client] Listener online")
		}

		// figure out what timeout to use
		if !timeoutSupplied {
			timeout = c.timeout.Get(req.service, req.endpoint, i)
		}
		log.Tracef("[Client] Sync request attempt %d for %s using timeout %v", i, req.MessageID(), timeout)

		// only bother sending the request if we are listening, otherwise allow to timeout
		if err := raven.SendRequest(req, c.instanceID); err != nil {
			log.Errorf("[Client] Failed to send request: %v", err)
		}

		select {
		case payload := <-rc:
			if payload.IsError() {
				inst.Timing(1.0, fmt.Sprintf("%s.error", instPrefix), time.Since(t))

				errorProto := &pe.PlatformError{}
				if err := payload.Unmarshal(errorProto); err != nil {
					inst.Counter(1.0, "client.error.com.hailocab.kernel.platform.badresponse", 1)
					return nil, errors.BadResponse("com.hailocab.kernel.platform.badresponse", err.Error())
				}

				err := errors.FromProtobuf(errorProto)
				inst.Counter(1.0, fmt.Sprintf("client.error.%s", err.Code()), 1)
				if errorProto.GetType() == pe.PlatformError_INTERNAL_SERVER_ERROR {
					circuitbreaker.Result(req.service, req.endpoint, err)
				} else {
					// consider everything not an internal server error a success
					circuitbreaker.Result(req.service, req.endpoint, nil)
				}
				return nil, err
			}

			inst.Timing(1.0, fmt.Sprintf("%s.success", instPrefix), time.Since(t))
			circuitbreaker.Result(req.service, req.endpoint, nil)
			return payload, nil
		case <-time.After(timeout):
			// timeout
			log.Errorf("[Client] Timeout talking to %s.%s after %v for %s", req.Service(), req.Endpoint(), timeout, req.MessageID())
			inst.Timing(1.0, fmt.Sprintf("%s.error", instPrefix), time.Since(t))
			c.traceAttemptTimeout(req, i, timeout)

			circuitbreaker.Result(req.service, req.endpoint, errors.Timeout("com.hailocab.kernel.platform.timeout",
				fmt.Sprintf("Request timed out talking to %s.%s from %s (most recent timeout %v)", req.Service(), req.Endpoint(), req.From(), timeout),
				req.Service(),
				req.Endpoint()))
		}
	}

	inst.Timing(1.0, fmt.Sprintf("%s.error.timedOut", instPrefix), time.Since(tAllRetries))
	inst.Counter(1.0, "client.error.com.hailocab.kernel.platform.timeout", 1)

	return nil, errors.Timeout(
		"com.hailocab.kernel.platform.timeout",
		fmt.Sprintf("Request timed out talking to %s.%s from %s (most recent timeout %v)", req.Service(), req.Endpoint(), req.From(), timeout),
		req.Service(),
		req.Endpoint(),
	)
}

// traceReq decides if we want to trigger a trace event (when sending a request) and if so deals with it
func (c *client) traceReq(req *Request) {
	if req.shouldTrace() {
		trace.Send(&traceproto.Event{
			Timestamp:       proto.Int64(time.Now().UnixNano()),
			TraceId:         proto.String(req.TraceID()),
			Type:            traceproto.Event_REQ.Enum(),
			MessageId:       proto.String(req.MessageID()),
			ParentMessageId: proto.String(req.ParentMessageID()),
			From:            proto.String(req.From()),
			FromEndpoint:    proto.String(req.FromEndpoint()),
			To:              proto.String(fmt.Sprintf("%v.%v", req.Service(), req.Endpoint())),
			Hostname:        proto.String(c.hostname),
			Az:              proto.String(c.az),
			Payload:         proto.String(""), // @todo
			PersistentTrace: proto.Bool(req.TraceShouldPersist()),
		})
	}
}

// traceRsp decides if we want to trigger a trace event (when processing response) and if so deals with it
func (c *client) traceRsp(req *Request, rsp *Response, err errors.Error, d time.Duration) {
	if req.shouldTrace() {
		e := &traceproto.Event{
			Timestamp:       proto.Int64(time.Now().UnixNano()),
			TraceId:         proto.String(req.TraceID()),
			Type:            traceproto.Event_REP.Enum(),
			MessageId:       proto.String(req.MessageID()),
			From:            proto.String(req.From()),
			FromEndpoint:    proto.String(req.FromEndpoint()),
			To:              proto.String(fmt.Sprintf("%v.%v", req.Service(), req.Endpoint())),
			ParentMessageId: proto.String(req.ParentMessageID()),
			Hostname:        proto.String(c.hostname),
			Az:              proto.String(c.az),
			Payload:         proto.String(""), // @todo
			Duration:        proto.Int64(int64(d)),
			PersistentTrace: proto.Bool(req.TraceShouldPersist()),
		}
		if err != nil {
			e.ErrorCode = proto.String(err.Code())
			e.ErrorDescription = proto.String(err.Description())
		}
		trace.Send(e)
	}
}

// traceAttemptTimeout decides if we want to trigger a trace event for an attempt timeout, and processes it
func (c *client) traceAttemptTimeout(req *Request, attemptNum int, timeout time.Duration) {
	if req.shouldTrace() {
		desc := fmt.Sprintf("Attempt %v timeout talking to '%s.%s' after '%v' for '%s'", attemptNum, req.Service(), req.Endpoint(), timeout, req.MessageID())
		trace.Send(&traceproto.Event{
			Timestamp:        proto.Int64(time.Now().UnixNano()),
			TraceId:          proto.String(req.TraceID()),
			Type:             traceproto.Event_ATTEMPT_TIMEOUT.Enum(),
			MessageId:        proto.String(req.MessageID()),
			From:             proto.String(req.From()),
			FromEndpoint:     proto.String(req.FromEndpoint()),
			To:               proto.String(fmt.Sprintf("%v.%v", req.Service(), req.Endpoint())),
			ParentMessageId:  proto.String(req.ParentMessageID()),
			Hostname:         proto.String(c.hostname),
			Az:               proto.String(c.az),
			Payload:          proto.String(""), // @todo
			ErrorCode:        proto.String("com.hailocab.kernel.platform.attemptTimeout"),
			ErrorDescription: proto.String(desc),
			Duration:         proto.Int64(int64(timeout)),
			PersistentTrace:  proto.Bool(req.TraceShouldPersist()),
		})
	}
}

// Push sends a request where we do not wish to wait for a REP (but is still REQ/REP pattern)
func (c *client) Push(req *Request) error {
	return raven.SendRequest(req, c.instanceID)
}

// AsyncTopic sends a pub/sub message (a Publication)
func (c *client) AsyncTopic(pub *Publication) error {
	return raven.SendPublication(pub, c.instanceID)
}

// Pub created and sends a Publication (via AsyncTopic) in one handy step
func (c *client) Pub(topic string, payload proto.Message) error {
	p, err := NewPublication(topic, payload)
	if err != nil {
		return err
	}
	return c.AsyncTopic(p)
}
