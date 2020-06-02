package server

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	jsonschemaproto "github.com/HailoOSS/platform/proto/jsonschema"
	ptesting "github.com/HailoOSS/platform/testing"
	"github.com/HailoOSS/protobuf/proto"
	"github.com/streadway/amqp"
)

type TestCategory int32

const (
	TestCategory_SUBSCRIBE     TestCategory = 0
	TestCategory_TRANSACTIONAL TestCategory = 1
	TestCategory_LIFECYCLE     TestCategory = 2
	TestCategory_SERVICE       TestCategory = 3
	TestCategory_PROMOS        TestCategory = 4
	TestCategory_DELIVERABLE   TestCategory = 6
)

var TestCategory_name = map[int32]string{
	0: "SUBSCRIBE",
	1: "TRANSACTIONAL",
	2: "LIFECYCLE",
	3: "SERVICE",
	4: "PROMOS",
	6: "DELIVERABLE",
}

func (x TestCategory) Enum() *TestCategory {
	p := new(TestCategory)
	*p = x
	return p
}
func (x TestCategory) String() string {
	return proto.EnumName(TestCategory_name, int32(x))
}

type testResponsePayload struct {
	Id                  *string        `protobuf:"bytes,1,req,name=id" json:"id,omitempty"`
	TestCategory        *TestCategory  `protobuf:"varint,2,req,name=testCategory,enum=TestCategory" json:"testCategory,omitempty"`
	LastUpdateTimestamp *int64         `protobuf:"varint,3,opt,name=lastUpdateTimestamp" json:"lastUpdateTimestamp,omitempty"`
	Unsubscribed        *bool          `protobuf:"varint,4,req,name=unsubscribed" json:"unsubscribed,omitempty"`
	OptedIn             []TestCategory `protobuf:"varint,5,rep,name=optedIn,enum=TestCategory" json:"optedIn,omitempty"`
	Token               *string        `protobuf:"bytes,6,req,name=token" json:"token,omitempty"`
	MarketingHob        *string        `protobuf:"bytes,7,opt,name=marketingHob" json:"marketingHob,omitempty"`
	LastOpenedTimestamp *int64         `protobuf:"varint,8,opt,name=lastOpenedTimestamp" json:"lastOpenedTimestamp,omitempty"`
	XXX_unrecognized    []byte         `json:"-"`
}

func (m *testResponsePayload) Reset()         { *m = testResponsePayload{} }
func (m *testResponsePayload) String() string { return proto.CompactTextString(m) }
func (*testResponsePayload) ProtoMessage()    {}

func TestJsonSchema(t *testing.T) {
	ptesting.RunSuite(t, new(JsonSchemaSuite))
}

type JsonSchemaSuite struct {
	ptesting.Suite
	origRegistry *registry
	registry     *registry
	origName     string
	origVersion  uint64
}

func (suite *JsonSchemaSuite) SetupTest() {
	suite.origName = Name
	Name = "com.HailoOSS.service.foo"
	suite.origVersion = Version
	Version = 201412100000
	suite.origRegistry = reg
	suite.registry = newRegistry()
	reg = suite.registry
	tokens = make(map[string]chan bool)
}

func (suite *JsonSchemaSuite) TearDownTest() {
	reg = suite.origRegistry
	suite.origRegistry = nil
	suite.registry = nil
	Name = suite.origName
	Version = suite.origVersion
	suite.origName = ""
}

func (suite *JsonSchemaSuite) TestJsonSchemaHandler() {
	t := suite.T()

	reg.add(&Endpoint{
		Name:             "foo",
		Mean:             100,
		Upper95:          100,
		ResponseProtocol: new(testResponsePayload),
		RequestProtocol:  new(jsonschemaproto.Request),
		Authoriser:       OpenToTheWorldAuthoriser(),
		Handler:          jsonschemaHandler,
	})

	req := NewRequestFromDelivery(amqp.Delivery{
		ContentType: "application/json",
		Body:        []byte(`{"endpoint":"foo"}`),
		MessageId:   "123",
		Headers: amqp.Table{
			"service":  Name,
			"endpoint": "foo",
		},
	})

	res, err := dummyHandler(req)
	if err != nil {
		t.Fatal(err)
	}

	// Marshal the resp to json
	bytes, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}

	var rsp map[string]interface{}
	var expected map[string]interface{}

	err = json.Unmarshal(bytes, &rsp)
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal([]byte(jsonschemaResponse), &expected)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(rsp, expected) {
		t.Fatalf("Received: %v \n Expected: %v", rsp, expected)
	}
}

func dummyHandler(req *Request) (proto.Message, error) {
	endpoint, ok := reg.find(req.Endpoint())
	if !ok {
		return nil, fmt.Errorf("Unable to find registered handler for %v", req.Endpoint())
	}
	// Unmarshal the request data in advance
	reqData, err := endpoint.unmarshalRequest(req)
	if err != nil {
		return nil, err
	}
	req.unmarshaledData = reqData

	return jsonschemaHandler(req)
}

const jsonschemaResponse = `{"jsonschema":"[{\"id\":\"http://directory.hailoweb.com/service/com.HailoOSS.service.foo-201412100000.json#foo\",\"title\":\"com.HailoOSS.service.foo-201412100000.foo\",\"description\":\"foo endpoint schema for com.HailoOSS.service.foo-201412100000\",\"type\":\"object\",\"$schema\":\"http://json-schema.org/draft-04/schema#\",\"properties\":{\"Request\":{\"type\":\"object\",\"properties\":{\"endpoint\":{\"type\":\"string\"}}},\"Response\":{\"type\":\"object\",\"required\":[\"id\",\"testCategory\",\"unsubscribed\",\"token\"],\"properties\":{\"id\":{\"type\":\"string\"},\"lastOpenedTimestamp\":{\"type\":\"integer\"},\"lastUpdateTimestamp\":{\"type\":\"integer\"},\"marketingHob\":{\"type\":\"string\"},\"optedIn\":{\"type\":\"array\",\"items\":{\"type\":\"string\",\"enum\":[\"SUBSCRIBE\",\"TRANSACTIONAL\",\"LIFECYCLE\",\"SERVICE\",\"PROMOS\"]}},\"testCategory\":{\"type\":\"string\",\"enum\":[\"SUBSCRIBE\",\"TRANSACTIONAL\",\"LIFECYCLE\",\"SERVICE\",\"PROMOS\"]},\"token\":{\"type\":\"string\"},\"unsubscribed\":{\"type\":\"boolean\"}}}}}]"}`
