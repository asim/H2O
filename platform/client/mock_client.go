package client

import (
	"encoding/json"
	"sync"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/protobuf/proto"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/mock"

	hailo_errors "github.com/HailoOSS/platform/errors"
)

type MockClient struct {
	mock.Mock
	expectedRequests     map[*Request]proto.Message
	expectedRequestsLock sync.Mutex
}

type MockResponseDelivery struct {
	ContentType   string
	Body          []byte
	MessageId     string
	CorrelationId string
	Headers       amqp.Table
}

// This must be set to an instance of MockClient before tests begin. NewMockClient will always return this pointer
var ActiveMockClient *MockClient = nil

func NewMockClient() Client {
	return ActiveMockClient
}

// Adds a request expectation to the mock, to be returned when a matching request is received (only the service and
// endpoint parameters are compared)
func (m *MockClient) AddRequestExpectation(req *Request, rsp proto.Message) {
	m.expectedRequestsLock.Lock()
	defer m.expectedRequestsLock.Unlock()

	if m.expectedRequests == nil {
		m.expectedRequests = make(map[*Request]proto.Message)
	}

	m.expectedRequests[req] = rsp
}

func (m *MockClient) getMatchingRequestExpectation(req *Request) *proto.Message {
	m.expectedRequestsLock.Lock()
	defer m.expectedRequestsLock.Unlock()

	if m.expectedRequests != nil {
		for matchingReq, matchingRsp := range m.expectedRequests {
			if matchingReq.Service() == req.Service() && matchingReq.Endpoint() == req.Endpoint() {
				delete(m.expectedRequests, matchingReq)
				return &matchingRsp
			}
		}
	}

	return nil
}

func (m *MockClient) Req(req *Request, rsp proto.Message, options ...Options) hailo_errors.Error {
	if matchedRsp := m.getMatchingRequestExpectation(req); matchedRsp != nil {
		// Marshall to JSON and back again, to get it into rsp (which is passed by value). This is fine; these are
		// marshalled and unmarshalled to JSON during normal operation anyway.
		marshalledJson, err := json.Marshal(*matchedRsp)
		if err != nil {
			return hailo_errors.InternalServerError("testing.is.fubard", "Can't marshall response to JSON")
		}
		err = json.Unmarshal(marshalledJson, rsp)
		if err != nil {
			return hailo_errors.InternalServerError("testing.is.fubard", "Can't unmarshall response from JSON")
		}

		log.Tracef("[Service client mock] Matched response to %s:%s", req.Service(), req.Endpoint())
		return nil
	} else {
		log.Warnf("[Service client mock] Couldn't match response for %s:%s", req.Service(), req.Endpoint())
		return hailo_errors.InternalServerError("com.HailoOSS.kernel.platform.nilresponse", "Nil response")
	}
}

func (m *MockClient) CustomReq(req *Request, options ...Options) (*Response, hailo_errors.Error) {
	if matchedRsp := m.getMatchingRequestExpectation(req); matchedRsp != nil {
		marshalledJson, err := json.Marshal(*matchedRsp)
		if err != nil {
			return nil, hailo_errors.InternalServerError("testing.is.fubard", "Can't marshall response to JSON")
		}

		resp := amqp.Delivery{
			ContentType:   "application/json",
			Body:          marshalledJson,
			MessageId:     "oh-what-a-lovely-test",
			CorrelationId: "oh-what-a-lovely-test",
			Headers:       map[string](interface{}){},
		}
		return &Response{resp}, nil
	} else {
		log.Warnf("[Service client mock] Couldn't match response for %s:%s", req.Service(), req.Endpoint())
		return nil, hailo_errors.InternalServerError("com.HailoOSS.kernel.platform.nilresponse", "Nil response")
	}
}

func (m *MockClient) Push(req *Request) error {
	returnArgs := m.Mock.Called(req)
	return returnArgs.Error(0)
}

func (m *MockClient) AsyncTopic(pub *Publication) error {
	returnArgs := m.Mock.Called(pub)
	return returnArgs.Error(0)
}

func (m *MockClient) Pub(topic string, payload proto.Message) error {
	returnArgs := m.Mock.Called(topic, payload)
	return returnArgs.Error(0)
}
