package multiclient

import (
	"fmt"
	"sync"
	"testing"

	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/errors"
	ptesting "github.com/hailo-platform/H2O/platform/testing"

	hcproto "github.com/hailo-platform/H2O/platform/proto/healthcheck"
)

func TestMultiClientSuite(t *testing.T) {
	ptesting.RunSuite(t, new(multiClientSuite))
}

type multiClientSuite struct {
	ptesting.Suite
}

func (suite *multiClientSuite) TestAnyErrorsIgnoring() {
	cases := []struct {
		errs         map[string]errors.Error
		types, codes []string
		isErr        bool
	}{
		{
			errs:  map[string]errors.Error{},
			types: []string{},
			codes: []string{},
			isErr: false,
		},
		// allow nils
		{
			errs:  map[string]errors.Error{},
			types: nil,
			codes: nil,
			isErr: false,
		},
		// no errors = always false
		{
			errs:  map[string]errors.Error{},
			types: []string{errors.ErrorForbidden, errors.ErrorBadResponse},
			codes: []string{"com.hailocab.service.foo.bar"},
			isErr: false,
		},
		{
			errs:  map[string]errors.Error{},
			types: []string{errors.ErrorForbidden, errors.ErrorBadResponse},
			codes: []string{},
			isErr: false,
		},
		{
			errs:  map[string]errors.Error{},
			types: []string{},
			codes: []string{"com.hailocab.service.foo.bar"},
			isErr: false,
		},
		{
			errs:  map[string]errors.Error{},
			types: []string{errors.ErrorForbidden, errors.ErrorBadResponse},
			codes: nil,
			isErr: false,
		},
		{
			errs:  map[string]errors.Error{},
			types: nil,
			codes: []string{"com.hailocab.service.foo.bar"},
			isErr: false,
		},
		// types
		{
			errs: map[string]errors.Error{
				"foo": errors.BadResponse("foo.bar", "ZOMG"),
			},
			types: []string{errors.ErrorForbidden, errors.ErrorBadResponse},
			codes: []string{},
			isErr: false,
		},
		{
			errs: map[string]errors.Error{
				"foo": errors.BadRequest("foo.bar", "ZOMG"),
			},
			types: []string{errors.ErrorForbidden, errors.ErrorBadResponse},
			codes: []string{},
			isErr: true,
		},
		{
			errs: map[string]errors.Error{
				"foo": errors.BadResponse("foo.bar", "ZOMG"),
				"bar": errors.BadRequest("foo.bar", "ZOMG"),
			},
			types: []string{errors.ErrorForbidden, errors.ErrorBadResponse},
			codes: []string{},
			isErr: true,
		},
		// codes
		{
			errs: map[string]errors.Error{
				"foo": errors.BadResponse("foo.bar", "ZOMG"),
			},
			types: nil,
			codes: []string{"foo.bar"},
			isErr: false,
		},
		{
			errs: map[string]errors.Error{
				"foo": errors.BadResponse("foo.bar", "ZOMG"),
			},
			types: nil,
			codes: []string{"foo.bar.baz"},
			isErr: true,
		},
		{
			errs: map[string]errors.Error{
				"foo": errors.BadResponse("foo.bar", "ZOMG"),
				"bar": errors.BadResponse("foo.bar.baz", "ZOMG"),
			},
			types: nil,
			codes: []string{"foo.bar"},
			isErr: true,
		},
	}

	cl := &defClient{
		errors: &errorsImpl{},
	}

	for _, tc := range cases {
		for uid, err := range tc.errs {
			cl.errors.set(uid, &client.Request{}, err, nil)
		}

		res := cl.AnyErrorsIgnoring(tc.types, tc.codes)
		suite.Assertions.Equal(tc.isErr, res, fmt.Sprintf("Wrong result for errors: %v - types: %v - codes: %v",
			tc.errs, tc.types, tc.codes))
	}
}

func (suite *multiClientSuite) TestSetCallerAndReset() {
	cl := &defClient{
		requests:  make(map[string]*client.Request),
		responses: make(map[string]proto.Message),
		errors:    &errorsImpl{},
	}

	c := ErrorCaller(errors.Forbidden("bad.person", "Much forbid"))

	cl.SetCaller(c)
	suite.Assertions.NotNil(cl.caller, "Set caller did not update caller :(")

	// do a call!
	cl.AddScopedReq(&ScopedReq{
		Service:  "com.hailocab.service.foo",
		Endpoint: "health",
		Req:      &hcproto.Request{},
		Rsp:      &hcproto.Response{},
	})
	cl.Execute()

	err := cl.Succeeded("")
	suite.Assertions.NotNil(err, "Error caller we set did not result in error!")
	suite.Assertions.Equal("bad.person", err.Code())

	// now we should be able to RESET this, and use the _same_ caller again
	suite.Assertions.True(cl.AnyErrors(), "Expecting us to HAVE errors, before we reset")
	suite.Assertions.False(cl.Reset().AnyErrors(), "Expecting us NOT to have errors, after we reset")

	// make same call -- crucial point is that the SetCaller thing shouldn't be reset
	cl.AddScopedReq(&ScopedReq{
		Service:  "com.hailocab.service.foo",
		Endpoint: "health",
		Req:      &hcproto.Request{},
		Rsp:      &hcproto.Response{},
	})
	cl.Execute()

	err = cl.Succeeded("")
	suite.Assertions.NotNil(err, "Error caller we set did not result in error!")
	suite.Assertions.Equal("bad.person", err.Code())
}

func (suite *multiClientSuite) TestPlatformError() {
	cl := &defClient{
		requests:  make(map[string]*client.Request),
		responses: make(map[string]proto.Message),
		errors:    &errorsImpl{},
	}
	cl.SetCaller(ErrorCaller(errors.Forbidden("bad.person", "Much forbid")))

	// A single request should not add the suffix provided to PlatformError() but return it verbatim
	cl.AddScopedReq(&ScopedReq{
		Service:  "com.hailocab.service.foo",
		Endpoint: "health",
		Req:      &hcproto.Request{},
		Rsp:      &hcproto.Response{},
	})
	cl.Execute()
	err := cl.PlatformError("suffixy")
	suite.Assertions.NotNil(err)
	suite.Assertions.True(errors.IsForbidden(err))
	suite.Assertions.Equal("bad.person", err.Code())
	suite.Assertions.Equal("Much forbid", err.Description())

	// Multiple errors should use the suffix provided
	cl.Reset()
	cl.AddScopedReq(&ScopedReq{
		Uid:      "uid1",
		Service:  "com.hailocab.service.foo",
		Endpoint: "health",
		Req:      &hcproto.Request{},
		Rsp:      &hcproto.Response{},
	})
	cl.AddScopedReq(&ScopedReq{
		Uid:      "uid2",
		Service:  "com.hailocab.service.foo",
		Endpoint: "bar",
		Req:      &hcproto.Request{},
		Rsp:      &hcproto.Response{},
	})
	cl.Execute()
	err = cl.PlatformError("suffixy")
	suite.Assertions.NotNil(err)
	suite.Assertions.True(errors.IsInternalServerError(err))
	suite.Assertions.Equal("suffixy", err.Code())

	// Multiple errors with a defaultScopeFrom should join the two
	cl.Reset()
	cl.DefaultScopeFrom(ExplicitScoper().SetContext("prefixy"))
	cl.AddScopedReq(&ScopedReq{
		Uid:      "uid1",
		Service:  "com.hailocab.service.foo",
		Endpoint: "health",
		Req:      &hcproto.Request{},
		Rsp:      &hcproto.Response{},
	})
	cl.AddScopedReq(&ScopedReq{
		Uid:      "uid2",
		Service:  "com.hailocab.service.foo",
		Endpoint: "bar",
		Req:      &hcproto.Request{},
		Rsp:      &hcproto.Response{},
	})
	cl.Execute()
	err = cl.PlatformError("suffixy")
	suite.Assertions.NotNil(err)
	suite.Assertions.True(errors.IsInternalServerError(err))
	suite.Assertions.Equal("prefixy.suffixy", err.Code())
}

func (suite *multiClientSuite) TestSucceeded() {
	cl := &defClient{
		requests:  make(map[string]*client.Request),
		responses: make(map[string]proto.Message),
		errors:    &errorsImpl{},
	}
	cl.SetCaller(ErrorCaller(errors.Forbidden("bad.person", "Much forbid")))

	// A single request should not add any context to the code
	cl.DefaultScopeFrom(ExplicitScoper().SetContext("prefixy"))
	cl.AddScopedReq(&ScopedReq{
		Service:  "com.hailocab.service.foo",
		Endpoint: "health",
		Req:      &hcproto.Request{},
		Rsp:      &hcproto.Response{},
	})
	cl.Execute()
	err := cl.Succeeded("")
	suite.Assertions.NotNil(err)
	suite.Assertions.True(errors.IsForbidden(err))
	suite.Assertions.Equal("bad.person", err.Code())
	suite.Assertions.Equal("Much forbid", err.Description())
}

func (suite *multiClientSuite) TestConcurrenctRequests() {
	mu := sync.Mutex{}
	i := 0

	cl := &defClient{
		requests:  make(map[string]*client.Request),
		responses: make(map[string]proto.Message),
		errors:    &errorsImpl{},
	}
	cl.SetConcurrency(2)
	cl.SetCaller(func(req *client.Request, rsp proto.Message) errors.Error {
		mu.Lock()
		i++
		mu.Unlock()

		return nil
	})

	// A single request should not add any context to the code
	cl.DefaultScopeFrom(ExplicitScoper().SetContext("prefixy"))
	cl.AddScopedReq(&ScopedReq{
		Uid:      "a",
		Service:  "com.hailocab.service.foo",
		Endpoint: "a",
		Req:      &hcproto.Request{},
		Rsp:      &hcproto.Response{},
	})
	cl.AddScopedReq(&ScopedReq{
		Uid:      "b",
		Service:  "com.hailocab.service.foo",
		Endpoint: "b",
		Req:      &hcproto.Request{},
		Rsp:      &hcproto.Response{},
	})
	cl.AddScopedReq(&ScopedReq{
		Uid:      "c",
		Service:  "com.hailocab.service.foo",
		Endpoint: "c",
		Req:      &hcproto.Request{},
		Rsp:      &hcproto.Response{},
	})
	cl.AddScopedReq(&ScopedReq{
		Uid:      "d",
		Service:  "com.hailocab.service.foo",
		Endpoint: "c",
		Req:      &hcproto.Request{},
		Rsp:      &hcproto.Response{},
	})
	cl.AddScopedReq(&ScopedReq{
		Uid:      "e",
		Service:  "com.hailocab.service.foo",
		Endpoint: "c",
		Req:      &hcproto.Request{},
		Rsp:      &hcproto.Response{},
	})
	cl.AddScopedReq(&ScopedReq{
		Uid:      "f",
		Service:  "com.hailocab.service.foo",
		Endpoint: "c",
		Req:      &hcproto.Request{},
		Rsp:      &hcproto.Response{},
	})
	cl.Execute()
	err := cl.Succeeded("")
	suite.Assertions.Nil(err)
	suite.Assertions.Equal(6, i)
}
