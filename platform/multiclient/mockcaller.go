package multiclient

import (
	"sync"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/errors"
)

type Mock struct {
	sync.RWMutex

	proxy Caller
	stubs []*Stub
}

// Responder is a function that yields a mocked response (successful in the form of proto.Message or
// unsuccessful in the form of errors.Error), in response to some inbound client `req`. The
// `invocation` parameter will contain the call sequence number, indexed from 1, for example
// it will be `1` the first time we are asked for a response, `2` on the next invocation etc..
type Responder func(invocation int, req *client.Request) (proto.Message, errors.Error)

type Stub struct {
	Service, Endpoint string

	Response        proto.Message     // Response to return on every matched request
	Error           errors.Error      // Error to return on every matched request
	Responder       Responder         // Responder function to use to give more control over exact response
	invocationLimit int               // Max number of times this stub can be called
	payload         proto.Message     // Expected request payload
	matched         []*client.Request // matched is a list of all matched requests
}

func NewMock() *Mock {
	return &Mock{stubs: make([]*Stub, 0)}
}

// Proxy adds a default proxy to this mock, which will be used as the caller if
// no stubs are matched (if none provided, then the default will be to return an error)
// - thread safe
func (m *Mock) Proxy(p Caller) *Mock {
	m.Lock()
	defer m.Unlock()
	m.proxy = p
	return m
}

// Stub adds a stubbed endpoint to this mock - thread safe
func (m *Mock) Stub(s *Stub) *Mock {
	m.Lock()
	defer m.Unlock()
	m.stubs = append(m.stubs, s)
	return m
}

func (m *Mock) On(service, endpoint string) *Stub {
	s := &Stub{
		Service:  service,
		Endpoint: endpoint,
	}
	m.Stub(s)
	return s
}

// Caller returns something that implements `Caller` - allowing us to use this as our
// gateway to service calls - the returned `Caller` is thread safe
func (m *Mock) Caller() Caller {
	return func(req *client.Request, rsp proto.Message) errors.Error {
		m.Lock()
		defer m.Unlock()
		for _, s := range m.stubs {
			if s.matches(req) {
				if s.Responder != nil {
					numMatched := len(s.matched)
					responderRsp, err := s.Responder(numMatched, s.matched[numMatched-1])
					if err != nil {
						return err
					}
					// put the responderRsp INTO the rsp
					b, _ := proto.Marshal(responderRsp)
					proto.Unmarshal(b, rsp)

					return nil
				}

				if s.Error != nil {
					return s.Error
				}

				// put the response INTO the rsp
				b, _ := proto.Marshal(s.Response)
				proto.Unmarshal(b, rsp)

				return nil
			}
		}
		// no match found - do default action
		if m.proxy != nil {
			return m.proxy(req, rsp)
		}
		// no default - return error
		return errors.NotFound("mock.notfound", "No mocked service registered to handle request.")
	}
}

// matches tests if this stub matches our request - not thread safe
func (s *Stub) matches(req *client.Request) bool {
	if req == nil {
		return false
	}
	if s.Service != req.Service() || s.Endpoint != req.Endpoint() {
		return false
	}
	if s.invocationLimit > 0 && len(s.matched) >= s.invocationLimit {
		return false
	}
	if s.payload != nil {
		clone := proto.Clone(s.payload)
		req.Unmarshal(clone)
		if !proto.Equal(s.payload, clone) {
			return false
		}
	}

	// got a match
	if s.matched == nil {
		s.matched = make([]*client.Request, 0)
	}
	s.matched = append(s.matched, req)
	return true
}

// CountCalls returns the number of calls that this stub has handled
func (s *Stub) CountCalls() int {
	if s == nil || s.matched == nil {
		return 0
	}
	return len(s.matched)
}

// Request returns the request object of call n, where n is zero-indexed,
// or nil if there was none (eg: what request caused this stub to be triggered)
func (s *Stub) Request(n int) *client.Request {
	if len(s.matched) >= n {
		return s.matched[n]
	}
	return nil
}

func (s *Stub) Payload(payload proto.Message) *Stub {
	s.payload = payload
	return s
}

func (s *Stub) Return(response proto.Message) *Stub {
	s.Response = response
	return s
}

func (s *Stub) Times(limit int) *Stub {
	s.invocationLimit = limit
	return s
}

func (s *Stub) Once() *Stub {
	return s.Times(1)
}

func (s *Stub) Fail(err errors.Error) *Stub {
	s.Error = err
	return s
}
