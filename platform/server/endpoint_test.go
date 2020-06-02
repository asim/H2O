package server

import (
	"testing"

	"github.com/HailoOSS/protobuf/proto"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"

	perrors "github.com/HailoOSS/platform/errors"
	ptesting "github.com/HailoOSS/platform/testing"
)

type testEndpointPayload struct {
	TestPayload
	Foo *string `protobuf:"bytes,2,req,name=foo" json:"foo,omitempty"`
}

func TestEndpoint(t *testing.T) {
	ptesting.RunSuite(t, new(EndpointSuite))
}

type EndpointSuite struct {
	ptesting.Suite
	origRegistry *registry
	registry     *registry
	origName     string
}

func (suite *EndpointSuite) SetupTest() {
	suite.origName = Name
	Name = "com.HailoOSS.service.foo"
	suite.origRegistry = reg
	suite.registry = newRegistry()
	reg = suite.registry
	tokens = make(map[string]chan bool)
}

func (suite *EndpointSuite) TearDownTest() {
	reg = suite.origRegistry
	suite.origRegistry = nil
	suite.registry = nil
	Name = suite.origName
	suite.origName = ""
}

func (suite *EndpointSuite) TestProtocolRegistration() {
	t := suite.T()
	inFoo := ""

	reg.add(&Endpoint{
		Name:             "foobar",
		Mean:             100,
		Upper95:          100,
		RequestProtocol:  new(testEndpointPayload),
		ResponseProtocol: new(testEndpointPayload),
		Authoriser:       OpenToTheWorldAuthoriser(),
		Handler: func(req *Request) (proto.Message, perrors.Error) {
			request := req.Data().(*testEndpointPayload)
			inFoo = *request.Foo
			return request, nil
		},
	})

	req := NewRequestFromDelivery(amqp.Delivery{
		ContentType: "application/json",
		Body:        []byte(`{"foo":"bar"}`),
		MessageId:   "123",
		Headers: amqp.Table{
			"service":  Name,
			"endpoint": "foobar",
		},
	})
	HandleRequest(req)
	assert.Equal(t, "bar", inFoo)
}
