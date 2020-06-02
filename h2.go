package h2

import (
	"github.com/hailocab/platform-layer/errors"
	"github.com/hailocab/platform-layer/multiclient"
	"github.com/hailocab/platform-layer/server"
	"github.com/hailocab/protobuf/proto"
)

type Error errors.Error

type Request interface {
	Data() proto.Message
	//Scope() Scope
}

// type Scope struct {
//	r *server.Request
// }

type Endpoint struct {
	Name string
	Func func(Request) (proto.Message, Error)
	Req  proto.Message
	Resp proto.Message
}

// ServiceStarter starts a single service.
// It is a singleton, so it can't run multiple services from the same process
type ServiceStarter interface {
	AddEndpoints(e ...Endpoint) ServiceStarter
	SetName(string) ServiceStarter
	SetDescription(string) ServiceStarter
	// FromType
	// ServiceInfo() ServiceInfo
	Run() error
}

type serviceStarter struct {
	name        string
	description string
	endpoints   []Endpoint
}

// NewService returns a ServiceStarter that you can use to start a service
func NewService(name, description string) ServiceStarter {
	return &serviceStarter{
		name:        name,
		description: description,
	}
}

func (s *serviceStarter) SetName(name string) ServiceStarter {
	s.name = name
	return s
}

func (s *serviceStarter) SetDescription(description string) ServiceStarter {
	s.description = description
	return s
}

// Registers and endpoint
func (s *serviceStarter) AddEndpoints(e ...Endpoint) ServiceStarter {
	s.endpoints = append(s.endpoints, e...)
	return s
}

func register(e Endpoint) {
	tmp := server.Endpoint{
		Name:             e.Name,
		Mean:             5000,
		Upper95:          10000,
		RequestProtocol:  e.Req,
		ResponseProtocol: e.Resp,
		Handler: func(r *server.Request) (proto.Message, errors.Error) {
			ret, err := e.Func(Request(r))
			if err != nil {
				return nil, Error(err)
			}
			return ret, nil
		},
	}
	server.Register(&tmp)
}

func (s *serviceStarter) Run() error {
	server.Name = s.name
	server.Description = s.description
	server.Init()
	for _, v := range s.endpoints {
		register(v)
	}
	server.Run()
	return nil
}

// Call a service. No scoping/tracing yet.
func Call(service, endpoint string, request, response proto.Message) error {
	call := multiclient.New().DefaultScopeFrom(server.Scoper()).AddScopedReq(&multiclient.ScopedReq{
		Uid:      "dummy",
		Service:  service,
		Endpoint: endpoint,
		Req:      request,
		Rsp:      response,
	}).Execute()
	if call.AnyErrors() {
		return call.Succeeded("dummy")
	}
	return nil
}
