package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/hailo-platform/H2O/protobuf/proto"
	"github.com/stretchr/testify/assert"

	checkinproto "github.com/hailo-platform/H2O/api-throttling-service/proto/checkin"
	"github.com/hailo-platform/H2O/platform/client"
	platformtesting "github.com/hailo-platform/H2O/platform/testing"
	"github.com/hailo-platform/H2O/service/config"
)

const throttlingTestConfigJson = `{
    "controlplane": {
        "configVersion": 10001,
        "rules": {
            "test-throttle": {
                "match": {
                    "path": "/throttle",
                    "proportion": 1.0
                },
                "action": 3,
                "payload": {
                    "httpStatus": 200,
                    "body": "ok"
                }
            }
        },
        "regions": {
            "eu-west-1": {
                "id": "eu-west-1",
                "status": "ONLINE",
                "apps": {
                    "default": {
                        "api": "api-driver-london.elasticride.com"
                    }
                }
            }
        },
        "hobRegions": {
            "LON": "eu-west-1"
        }
    }
}`

func TestThrottlingSuite(t *testing.T) {
	suite := new(throttlingSuite)
	suite.LogLevel = "off"
	platformtesting.RunSuite(t, suite)
}

type throttlingSuite struct {
	platformtesting.Suite
}

// Check that requests without a session ID do not get bucketed
func (suite *throttlingSuite) TestRecording_NoSessionId() {
	t := suite.T()

	// Setup a dummy service client (over which the thin API will make calls)
	mockServiceClient := &client.MockClient{}
	client.ActiveMockClient = mockServiceClient
	client.NewClient = client.NewMockClient
	client.DefaultClient = client.NewClient()
	defer func() {
		client.ActiveMockClient = nil
		client.NewClient = client.NewDefaultClient
		client.DefaultClient = client.NewClient()
	}()

	// ...config to "throttle" a successful request (but return a 200 response; not really throttling, just abusing
	// the throttle rules in the controlplane)
	configBuf := bytes.NewBufferString(throttlingTestConfigJson)
	origConfigBuf := bytes.NewBuffer(config.Raw())
	config.Load(configBuf)
	defer config.Load(origConfigBuf)

	// ...and a dummy thin API server
	server, apiClient := SetupTestServerAndClient(t)
	defer TeardownTestServer(t, server)

	request, err := http.NewRequest("GET", fmt.Sprintf("%s/throttle", server.URL), nil)
	assert.NoError(t, err, "Request construction error")

	response, err := apiClient.Do(request)
	assert.NoError(t, err, "Request error")
	assert.Equal(t, 200, response.StatusCode, "Status code not as expected")

	// Swap the pointer out to avoid a data race (could equally kill the server, but this is a bit nicer as the shutdown
	// behaviour isn't necessairly *defined* to leave the throttling state in tact)
	newBuf := make(bucketBufferT, defaultBufferSize)
	buf := *(*bucketBufferT)(atomic.SwapPointer(&(server.ThrottlingHandler.bucketBuffer), (unsafe.Pointer)(&newBuf)))
	assert.Equal(t, 0, len(buf), "No bucket should be recorded without a session ID")
}

// Check that requests with a session ID do get bucketed
func TestRecording(t *testing.T) {
	// Setup a dummy service client (over which the thin API will make calls)
	mockServiceClient := &client.MockClient{}
	client.ActiveMockClient = mockServiceClient
	client.NewClient = client.NewMockClient
	client.DefaultClient = client.NewClient()
	defer func() {
		client.ActiveMockClient = nil
		client.NewClient = client.NewDefaultClient
		client.DefaultClient = client.NewClient()
	}()

	// ...config to "throttle" a successful request (but return a 200 response; not really throttling, just abusing
	// the throttle rules in the controlplane)
	configBuf := bytes.NewBufferString(throttlingTestConfigJson)
	origConfigBuf := bytes.NewBuffer(config.Raw())
	config.Load(configBuf)
	defer config.Load(origConfigBuf)

	// ...and a dummy thin API server
	server, apiClient := SetupTestServerAndClient(t)
	defer TeardownTestServer(t, server)

	request, err := http.NewRequest("GET", fmt.Sprintf("%s/throttle?session_id=abc", server.URL), nil)
	assert.NoError(t, err, "Request construction error")

	response, err := apiClient.Do(request)
	assert.NoError(t, err, "Request error")
	assert.Equal(t, 200, response.StatusCode, "Status code not as expected")

	// Swap the pointer out to avoid a data race (could equally kill the server, but this is a bit nicer as the shutdown
	// behaviour isn't necessairly *defined* to leave the throttling state in tact)
	newBuf := make(bucketBufferT, defaultBufferSize)
	buf := *(*bucketBufferT)(atomic.SwapPointer(&(server.ThrottlingHandler.bucketBuffer), (unsafe.Pointer)(&newBuf)))
	assert.Equal(t, 1, len(buf), "Session ID should create a throttling bucket")
	val, ok := buf["sessId:abc"]
	assert.True(t, ok, "Expected bucket key is not present")
	assert.Equal(t, uint64(1), *val, "Bucket depth not as expected")
}

func TestSynchronisation(t *testing.T) {
	// Setup a dummy service client (over which the thin API will make calls)
	mockServiceClient := &client.MockClient{}
	client.ActiveMockClient = mockServiceClient
	client.NewClient = client.NewMockClient
	client.DefaultClient = client.NewClient()
	defer func() {
		client.ActiveMockClient = nil
		client.NewClient = client.NewDefaultClient
		client.DefaultClient = client.NewClient()
	}()

	// ...config to "throttle" a successful request (but return a 200 response; not really throttling, just abusing
	// the throttle rules in the controlplane)
	configBuf := bytes.NewBufferString(throttlingTestConfigJson)
	origConfigBuf := bytes.NewBuffer(config.Raw())
	config.Load(configBuf)
	defer config.Load(origConfigBuf)

	// ...and a dummy thin API server
	server, apiClient := SetupTestServerAndClient(t)
	defer TeardownTestServer(t, server)

	// Expect a service-to-service call
	expectedRequestPayload := &checkinproto.Request{
		BucketRequests: []*checkinproto.BucketRequests{
			&checkinproto.BucketRequests{
				BucketKey:    proto.String("sessId:abc"),
				RequestCount: proto.Uint64(10),
			},
		},
	}
	expectedRequest, err := client.NewRequest("com.hailo-platform/H2O.service.api-throttling", "checkin", expectedRequestPayload)
	assert.NoError(t, err, "Error constructing expected service request")
	expectedResponse := &checkinproto.Response{
		ThrottledBuckets: []string{"sessId:abc"},
	}
	mockServiceClient.AddRequestExpectation(expectedRequest, expectedResponse)

	// Make 10 requests with the same session ID
	for i := 0; i < 10; i++ {
		request, err := http.NewRequest("GET", fmt.Sprintf("%s/throttle?session_id=abc", server.URL), nil)
		assert.NoError(t, err, "Request construction error")

		response, err := apiClient.Do(request)
		assert.NoError(t, err, "Request error")
		assert.Equal(t, 200, response.StatusCode, "Status code not as expected")
	}

	time.Sleep(synchronisationInterval + 500*time.Millisecond)
	// Should have now synchronised

	request, err := http.NewRequest("GET", fmt.Sprintf("%s/throttle?session_id=abc", server.URL), nil)
	assert.NoError(t, err, "Request construction error")
	response, err := apiClient.Do(request)
	assert.NoError(t, err, "Request error")
	// Throttling has been disabled.
	// assert.Equal(t, 429, response.StatusCode, "Status code not as expected")

	time.Sleep(synchronisationInterval + 500*time.Millisecond)
	// Should have now synchronised again, and since there was no expectation, the call will have failed, and we should
	// not have any buckets throttled

	request, err = http.NewRequest("GET", fmt.Sprintf("%s/throttle?session_id=abc", server.URL), nil)
	assert.NoError(t, err, "Request construction error")
	response, err = apiClient.Do(request)
	assert.NoError(t, err, "Request error")
	assert.Equal(t, 200, response.StatusCode, "Status code not as expected")
}
