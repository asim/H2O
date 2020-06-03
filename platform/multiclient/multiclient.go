package multiclient

import (
	"fmt"
	"sync"

	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/service/config"
)

// MultiClient represents a session where we want to make one or more requests
type MultiClient interface {
	// DefaultScopeFrom defines a server request that we can leverage for scoping, meaning that any
	// `ScopedReq`s added don't have to include the From scope
	DefaultScopeFrom(s Scoper) MultiClient
	// SetCaller optionally defines a caller for use with this multiclient request (overrides the default)
	SetCaller(c Caller) MultiClient
	// AddScopedReq adds a server-scoped request (from the server request `from`) to our multi-client
	// with the `uid` that uniquely identifies the request within the group (for getting response from `Outcome`)
	AddScopedReq(sr *ScopedReq) MultiClient
	// Reset removes all scoped requests/results ready for re-use
	Reset() MultiClient
	// Execute runs all requests in parallel, blocking until all have completed
	Execute() MultiClient
	// AnyErrors will return true if ANY request resulted in an error
	AnyErrors() bool
	// AnyErrorsIgnoring will return true if ANY request resulted in an error, but ignoring the supplied types/codes
	AnyErrorsIgnoring(types []string, codes []string) bool
	// PlatformError mints a new platform error, with the specified code suffix (and server namespace prefix),
	// describing all errors that occurred
	PlatformError(codeSuffix string) errors.Error
	// Errors returns Errors representing all errors that occurred
	Errors() Errors
	// Succeeded tests if request with this uid had an error
	Succeeded(uid string) errors.Error
	// Response returns the response for request uid
	Response(uid string) proto.Message
}

// ScopedReq describes a scoped request
type ScopedReq struct {
	Uid      string
	From     Scoper
	Service  string
	Endpoint string
	Req      proto.Message
	Rsp      proto.Message
	Options  client.Options
}

// defClient default MultiClient implementation
type defClient struct {
	sync.RWMutex
	defaultFromScope Scoper
	done             bool
	concurrency      int
	requests         map[string]*client.Request
	responses        map[string]proto.Message
	caller           Caller
	errors           *errorsImpl
}

// singleRsp used internally to pass back the err/uid for each request fired off
type singleRsp struct {
	uid string
	err errors.Error
}

// singleReq used internally to pass back the uid/req to the worker pool
type singleReq struct {
	uid string
	req *client.Request
}

// New mints a new default MultiClient
func New() MultiClient {
	return &defClient{
		requests:  make(map[string]*client.Request),
		responses: make(map[string]proto.Message),
		errors:    &errorsImpl{},
		caller:    getCaller(),
	}
}

// Reset removes all scoped requests/results ready for re-use
func (c *defClient) Reset() MultiClient {
	c.Lock()
	defer c.Unlock()
	c.requests = make(map[string]*client.Request)
	c.responses = make(map[string]proto.Message)
	c.errors = &errorsImpl{
		defaultScoper: c.defaultFromScope,
	}
	c.done = false
	return c
}

// DefaultScopeFrom defines a server request that we can leverage for scoping, meaning that any
// `ScopedReq`s added don't have to include the From scope
func (c *defClient) DefaultScopeFrom(s Scoper) MultiClient {
	c.Lock()
	defer c.Unlock()
	c.defaultFromScope = s
	c.errors.Lock()
	defer c.errors.Unlock()
	c.errors.defaultScoper = s
	return c
}

// SetCaller optionally defines a caller for use with this multiclient request (overrides the default)
func (c *defClient) SetCaller(caller Caller) MultiClient {
	c.Lock()
	defer c.Unlock()
	c.caller = caller
	return c
}

// SetConcurrency optionally defines a caller for use with this multiclient request (overrides the default)
func (c *defClient) SetConcurrency(concurrency int) MultiClient {
	c.Lock()
	defer c.Unlock()
	c.concurrency = concurrency
	return c
}

// AddScopedReq adds a server-scoped request (from the server request `from`) to our multi-client
// with the `uid` that uniquely identifies the request within the group (for getting response from `Outcome`)
func (c *defClient) AddScopedReq(sr *ScopedReq) MultiClient {
	c.Lock()
	defer c.Unlock()
	if _, exists := c.requests[sr.Uid]; exists {
		panic(fmt.Sprintf("Cannot add scoped request with UID '%v' - already exists within this MultiClient", sr.Uid))
	}
	from := sr.From
	if from == nil {
		from = c.defaultFromScope
	}

	var clientReq *client.Request
	var err error

	// if no from, just use normal client request
	if from == nil {
		clientReq, err = client.NewRequest(sr.Service, sr.Endpoint, sr.Req)
	} else {
		clientReq, err = from.ScopedRequest(sr.Service, sr.Endpoint, sr.Req)
	}

	c.requests[sr.Uid] = clientReq
	c.responses[sr.Uid] = sr.Rsp
	if err != nil {
		c.errors.set(sr.Uid, clientReq,
			errors.InternalServerError("com.hailo-platform/H2O.kernel.multirequest.badrequest", err.Error()), from)
	} else {
		clientReq.SetOptions(sr.Options)
	}

	return c
}

// Execute runs all requests in parallel, blocking until all have completed
func (c *defClient) Execute() MultiClient {
	c.Lock()
	defer c.Unlock()
	if c.done {
		panic("Cannot repeat Execute() on a MultiClient - not supported")
	}
	c.done = true

	requests := make(chan *singleReq)
	responses := make(chan *singleRsp)
	stop := make(chan struct{})
	inFlight := 0

	// Start request workers
	concurrency := config.AtPath("hailo", "platform", "request", "concurrency").AsInt(10)
	if c.concurrency > 0 {
		concurrency = c.concurrency
	}

	for i := 0; i < concurrency; i++ {
		go c.startRequestWorker(stop, requests, responses)
	}

	for uid, req := range c.requests {
		// already an err creating req?
		if exists := c.errors.ForUid(uid) != nil; exists {
			continue
		} else if req == nil {
			log.Warnf("[Multiclient] Not expecting nil Request within MultiClient")
			c.errors.set(uid, req, errors.InternalServerError(
				"com.hailo-platform/H2O.kernel.multirequest.badrequest.nil",
				fmt.Sprintf("Response for uid %s is nil", uid)), nil)
			continue
		}

		// Send request to requests channel
		inFlight++
		go c.addRequestToQueue(uid, req, requests)
	}

	// Wait for responses
	for i := 0; i < inFlight; i++ {
		if r := <-responses; r.err != nil {
			c.errors.set(r.uid, c.requests[r.uid], r.err, nil)
		}
	}

	// Stop all the request workers
	close(stop)

	return c
}

func (c *defClient) startRequestWorker(stop chan struct{}, requests chan *singleReq, responses chan *singleRsp) {
	for {
		select {
		case r := <-requests:
			err := c.caller(r.req, c.responses[r.uid])

			responses <- &singleRsp{r.uid, err}
		case <-stop:
			return
		}
	}
}

func (c *defClient) addRequestToQueue(uid string, req *client.Request, requests chan *singleReq) {
	requests <- &singleReq{uid, req}
}

// AnyErrors will return true if ANY request resulted in an error
func (c *defClient) AnyErrors() bool {
	return c.errors.Count() > 0
}

// AnyErrorsIgnoring will return true if ANY request resulted in an error, but
// ignoring the supplied types/codes
func (c *defClient) AnyErrorsIgnoring(types []string, codes []string) bool {
	return c.errors.IgnoreType(types...).IgnoreCode(codes...).Count() > 0
}

// PlatformError mints a new platform error, with the specified code suffix,
// combined with the default scope Context() prefix (eg: to prepend server name)
func (c *defClient) PlatformError(codeSuffix string) errors.Error {
	switch c.errors.Count() {
	case 0:
		return nil
	case 1:
		return c.errors.Combined()
	default:
		return c.errors.Suffix(codeSuffix).Combined()
	}
}

func (c *defClient) Errors() Errors {
	return c.errors
}

// Succeeded tests if request with this uid had an error
func (c *defClient) Succeeded(uid string) errors.Error {
	return c.errors.ForUid(uid)
}

// Response returns the response for request uid
func (c *defClient) Response(uid string) proto.Message {
	c.RLock()
	defer c.RUnlock()
	if rsp, exists := c.responses[uid]; exists {
		return rsp
	}
	return nil
}

// inList checks if string s is in array a
func inList(s string, a []string) bool {
	for _, test := range a {
		if s == test {
			return true
		}
	}
	return false
}
