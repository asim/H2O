package server

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/hailo-platform/H2O/protobuf/proto"

	perrors "github.com/hailo-platform/H2O/platform/errors"
)

// Endpoint containing the name and handler to call with the request
type Endpoint struct {
	// Name is the endpoint name, which should just be a single word, eg: "register"
	Name string
	// Mean is the mean average response time (time to generate response) promised for this endpoint
	Mean int32
	// Upper95 is 95th percentile response promised for this endpoint
	Upper95 int32
	// Handler is the function that will be fed requests to respond to
	Handler Handler
	// RequestProtocol is a struct type into which an inbound request for this endpoint can be unmarshaled
	RequestProtocol proto.Message
	// ResponseProtocol is the struct type defining the response format for this endpoint
	ResponseProtocol proto.Message
	// Subscribe indicates this endpoint should subscribe to a PUB stream - and gives us the stream address to SUB from
	//(topic name)
	Subscribe string
	// Authoriser is something that can check authorisation for this endpoint -- defaulting to ADMIN only (if nothing
	//specified by service)
	Authoriser Authoriser

	protoTMtx sync.RWMutex
	reqProtoT reflect.Type // cached type
	rspProtoT reflect.Type // cached type
}

func (ep *Endpoint) GetName() string {
	return ep.Name
}

func (ep *Endpoint) GetMean() int32 {
	return ep.Mean
}

func (ep *Endpoint) GetUpper95() int32 {
	return ep.Upper95
}

// ProtoTypes returns the Types of the registered request and response protocols
func (ep *Endpoint) ProtoTypes() (reflect.Type, reflect.Type) {
	ep.protoTMtx.RLock()
	reqT, rspT := ep.reqProtoT, ep.rspProtoT
	ep.protoTMtx.RUnlock()

	if (reqT == nil && ep.RequestProtocol != nil) || (rspT == nil && ep.ResponseProtocol != nil) {
		ep.protoTMtx.Lock()
		reqT, rspT = ep.reqProtoT, ep.rspProtoT // Prevent thundering herd
		if reqT == nil && ep.RequestProtocol != nil {
			reqT = reflect.TypeOf(ep.RequestProtocol)
			ep.reqProtoT = reqT
		}
		if rspT == nil && ep.ResponseProtocol != nil {
			rspT = reflect.TypeOf(ep.ResponseProtocol)
			ep.rspProtoT = rspT
		}
		ep.protoTMtx.Unlock()
	}

	return reqT, rspT
}

// unmarshalRequest reads a request's payload into a RequestProtocol object
func (ep *Endpoint) unmarshalRequest(req *Request) (proto.Message, perrors.Error) {
	reqProtoT, _ := ep.ProtoTypes()

	if reqProtoT == nil { // No registered protocol
		return nil, nil
	}

	result := reflect.New(reqProtoT.Elem()).Interface().(proto.Message)
	if err := req.Unmarshal(result); err != nil {
		return nil, perrors.InternalServerError(fmt.Sprintf("%s.%s.unmarshal", Name, ep.Name), err.Error())
	}

	return result, nil
}
