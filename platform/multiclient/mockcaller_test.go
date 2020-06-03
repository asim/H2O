package multiclient

import (
	"testing"

	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/stretchr/testify/assert"

	hcproto "github.com/hailo-platform/H2O/platform/proto/healthcheck"
)

const (
	mockFooService     = "com.hailo-platform/H2O.service.foo"
	mockHealthEndpoint = "health"
)

func TestMockCallerDefaultAction(t *testing.T) {
	mock := NewMock()

	caller := mock.Caller()
	e := caller(nil, nil)
	assert.NotNil(t, e,
		"Default action of mock caller should be to trigger an error")
	assert.Equal(t, e.Type(), errors.ErrorNotFound,
		"Default error of mock caller should be NotFound")
	assert.Equal(t, e.Code(), "mock.notfound",
		"Default code of mock caller should be mock.notfound")
}

func TestMockCallerDefaultActionSupplied(t *testing.T) {
	// we proxy the default action to our error generator - so we should get code we expect
	mock := NewMock().Proxy(ErrorCaller(nil))

	caller := mock.Caller()
	e := caller(nil, nil)
	assert.NotNil(t, e, "Expecting mock caller to trigger an error")
	assert.Equal(t, e.Type(), errors.ErrorNotFound,
		"Expecting mock caller to trigger NotFound")
	assert.Equal(t, e.Code(), "errorcaller.notfound",
		"Expecting code of mock caller to be errorcaller.notfound")
}

func TestMockCallerWithOneStubResponse(t *testing.T) {
	req, _ := client.NewRequest(mockFooService, mockHealthEndpoint, &hcproto.Request{})
	stub := &Stub{
		Service:  mockFooService,
		Endpoint: mockHealthEndpoint,
		Response: &hcproto.Response{},
	}
	mock := NewMock().Stub(stub)

	caller := mock.Caller()
	rsp := &hcproto.Response{}
	e := caller(req, rsp)
	assert.Nil(t, e,
		"Expecting our mocked call to be intercepted and stubbed response returned, got err: %v", e)

	// ensure stub has what we expect
	assert.Len(t, stub.matched, 1,
		"Expecting 1 match payload to be stored after execution")

	assert.Equal(t, stub.CountCalls(), 1, "CountCalls should return 1 too")

	// try something else that _shouldn't_ match
	req, _ = client.NewRequest(mockFooService, "baz", &hcproto.Request{})
	rsp = &hcproto.Response{}
	e = caller(req, rsp)
	assert.NotNil(t, e, "Expecting different endpoint name NOT to match")
	assert.Equal(t, e.Code(), "mock.notfound",
		"Expecting code of mock caller to be mock.notfound")
}

func TestMockCallerPopulatesResponse(t *testing.T) {
	req, _ := client.NewRequest(mockFooService, mockHealthEndpoint, &hcproto.Request{})
	stub := &Stub{
		Service:  mockFooService,
		Endpoint: mockHealthEndpoint,
		Response: &hcproto.Response{
			Healthchecks: []*hcproto.HealthCheck{
				{
					Timestamp:      proto.Int64(1403629015),
					ServiceName:    proto.String("foo"),
					ServiceVersion: proto.Uint64(1403629015),
					Hostname:       proto.String("localhost"),
					InstanceId:     proto.String("foobar"),
					HealthCheckId:  proto.String("boom"),
					IsHealthy:      proto.Bool(true),
				},
			},
		},
	}
	mock := NewMock().Stub(stub)

	caller := mock.Caller()
	rsp := &hcproto.Response{}
	e := caller(req, rsp)
	assert.Nil(t, e,
		"Expecting our mocked call to be intercepted and stubbed response returned, got err: %v", e)

	// ensure stub has what we expect
	assert.Len(t, stub.matched, 1,
		"Expecting 1 match payload to be stored after execution")

	assert.Equal(t, stub.CountCalls(), 1, "CountCalls should return 1 too")

	assert.Len(t, rsp.GetHealthchecks(), 1,
		"Response does not contain our mocked content: no healthchecks")
}

func TestResponder(t *testing.T) {
	stub := &Stub{
		Service:  mockFooService,
		Endpoint: mockHealthEndpoint,
		Responder: func(invocation int, req *client.Request) (proto.Message, errors.Error) {
			if invocation == 1 {
				return &hcproto.Response{
					Healthchecks: []*hcproto.HealthCheck{
						{
							Timestamp:      proto.Int64(1403629015),
							ServiceName:    proto.String("foo"),
							ServiceVersion: proto.Uint64(1403629015),
							Hostname:       proto.String("localhost"),
							InstanceId:     proto.String("foobar"),
							HealthCheckId:  proto.String("boom"),
							IsHealthy:      proto.Bool(true),
						},
					},
				}, nil
			}
			return nil, errors.InternalServerError("only.one.allowed", "First call only works")
		},
	}
	mock := NewMock().Stub(stub)

	caller := mock.Caller()
	req, _ := client.NewRequest(mockFooService, mockHealthEndpoint, &hcproto.Request{})
	rsp := &hcproto.Response{}
	e := caller(req, rsp)

	assert.Nil(t, e,
		"Expecting our mocked call to be intercepted and stubbed response returned, got err: %v", e)

	assert.Len(t, rsp.GetHealthchecks(), 1,
		"Response does not contain our mocked content: no healthchecks")

	// now repeat, and we SHOULD get an error
	e = caller(req, rsp)
	assert.NotNil(t, e,
		"Expecting our mocked call to be intercepted and error response returned on 2nd call")

	assert.Equal(t, e.Code(), "only.one.allowed",
		"Expecting code 'only.one.allowed', got '%s'", e.Code())
}

// --- Fluent interface tests

type Dummy struct {
	Id *string `protobuf:"bytes,1,req,name=Id"`
}

func NewDummy(id string) *Dummy { return &Dummy{&id} }
func (m *Dummy) Reset()         { *m = Dummy{} }
func (m *Dummy) String() string { return proto.CompactTextString(m) }
func (*Dummy) ProtoMessage()    {}

func TestFluentStubbingUnlimited(t *testing.T) {
	mock := NewMock()
	mock.
		On(mockFooService, mockHealthEndpoint).
		Return(NewDummy("pong"))

	caller := mock.Caller()

	req, _ := client.NewRequest(mockFooService, mockHealthEndpoint, NewDummy("ping"))
	rsp := &Dummy{}

	// Succeed 1st call
	err := caller(req, rsp)
	assert.Nil(t, err)
	assert.Equal(t, *rsp.Id, "pong")

	// Succeed 2nd call
	err = caller(req, rsp)
	assert.Nil(t, err)
	assert.Equal(t, *rsp.Id, "pong")
}

func TestFluentStubbingWithTimes(t *testing.T) {
	mock := NewMock()
	mock.
		On(mockFooService, mockHealthEndpoint).
		Return(NewDummy("pong")).
		Once()

	caller := mock.Caller()

	req, _ := client.NewRequest(mockFooService, mockHealthEndpoint, NewDummy("ping"))
	rsp := &Dummy{}

	// Succeed 1st call
	assert.Nil(t, caller(req, rsp))
	assert.Equal(t, *rsp.Id, "pong")

	// Fail 2nd call
	err := caller(req, rsp)
	assert.NotNil(t, err)
	assert.Equal(t, err.Code(), "mock.notfound")
}

func TestFluentStubbingSequence(t *testing.T) {
	mock := NewMock()
	mock.
		On(mockFooService, mockHealthEndpoint).
		Return(NewDummy("pong-1")).
		Once()

	mock.
		On(mockFooService, mockHealthEndpoint).
		Return(NewDummy("pong-2")).
		Once()

	caller := mock.Caller()

	req, _ := client.NewRequest(mockFooService, mockHealthEndpoint, NewDummy("ping"))
	rsp := &Dummy{}

	// Succeed 1st call
	assert.Nil(t, caller(req, rsp))
	assert.Equal(t, *rsp.Id, "pong-1")

	// Succeed 2nd call with different response
	assert.Nil(t, caller(req, rsp))
	assert.Equal(t, *rsp.Id, "pong-2")
}

func TestFluentStubbingWithPayload(t *testing.T) {
	mock := NewMock()
	mock.
		On(mockFooService, mockHealthEndpoint).
		Payload(NewDummy("ping")).
		Return(NewDummy("pong"))

	caller := mock.Caller()

	// Fail as payload does not match
	req, _ := client.NewRequest(mockFooService, mockHealthEndpoint, NewDummy("pong"))
	rsp := &Dummy{}
	err := caller(req, rsp)
	assert.NotNil(t, err)
	assert.Equal(t, err.Code(), "mock.notfound")

	// Succeed as payload matches
	req, _ = client.NewRequest(mockFooService, mockHealthEndpoint, NewDummy("ping"))
	assert.Nil(t, caller(req, rsp))
	assert.Equal(t, *rsp.Id, "pong")
}

func TestFluentStubbingWithError(t *testing.T) {
	mock := NewMock()
	mock.
		On(mockFooService, mockHealthEndpoint).
		Fail(errors.BadRequest("code", "description"))

	req, _ := client.NewRequest(mockFooService, mockHealthEndpoint, NewDummy("ping"))
	rsp := &Dummy{}

	// Fail with given error
	err := mock.Caller()(req, rsp)
	assert.NotNil(t, err)
	assert.Equal(t, err.Code(), "code")
	assert.Equal(t, err.Description(), "description")
}
