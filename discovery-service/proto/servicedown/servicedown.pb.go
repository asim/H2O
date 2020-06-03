// Code generated by protoc-gen-go.
// source: github.com/hailo-platform/H2O/discovery-service/proto/servicedown/servicedown.proto
// DO NOT EDIT!

/*
Package com_hailo-platform/H2O_kernel_discovery_servicedown is a generated protocol buffer package.

It is generated from these files:
	github.com/hailo-platform/H2O/discovery-service/proto/servicedown/servicedown.proto

It has these top-level messages:
	Request
	Response
*/
package com_hailo-platform/H2O_kernel_discovery_servicedown

import proto "github.com/hailo-platform/H2O/protobuf/proto"
import json "encoding/json"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

type Request struct {
	InstanceId         *string `protobuf:"bytes,1,req,name=instanceId" json:"instanceId,omitempty"`
	Hostname           *string `protobuf:"bytes,2,req,name=hostname" json:"hostname,omitempty"`
	ServiceName        *string `protobuf:"bytes,3,req,name=serviceName" json:"serviceName,omitempty"`
	ServiceDescription *string `protobuf:"bytes,4,opt,name=serviceDescription" json:"serviceDescription,omitempty"`
	ServiceVersion     *uint64 `protobuf:"varint,5,req,name=serviceVersion" json:"serviceVersion,omitempty"`
	EndpointName       *string `protobuf:"bytes,6,opt,name=endpointName" json:"endpointName,omitempty"`
	AzName             *string `protobuf:"bytes,7,req,name=azName" json:"azName,omitempty"`
	XXX_unrecognized   []byte  `json:"-"`
}

func (m *Request) Reset()         { *m = Request{} }
func (m *Request) String() string { return proto.CompactTextString(m) }
func (*Request) ProtoMessage()    {}

func (m *Request) GetInstanceId() string {
	if m != nil && m.InstanceId != nil {
		return *m.InstanceId
	}
	return ""
}

func (m *Request) GetHostname() string {
	if m != nil && m.Hostname != nil {
		return *m.Hostname
	}
	return ""
}

func (m *Request) GetServiceName() string {
	if m != nil && m.ServiceName != nil {
		return *m.ServiceName
	}
	return ""
}

func (m *Request) GetServiceDescription() string {
	if m != nil && m.ServiceDescription != nil {
		return *m.ServiceDescription
	}
	return ""
}

func (m *Request) GetServiceVersion() uint64 {
	if m != nil && m.ServiceVersion != nil {
		return *m.ServiceVersion
	}
	return 0
}

func (m *Request) GetEndpointName() string {
	if m != nil && m.EndpointName != nil {
		return *m.EndpointName
	}
	return ""
}

func (m *Request) GetAzName() string {
	if m != nil && m.AzName != nil {
		return *m.AzName
	}
	return ""
}

type Response struct {
	XXX_unrecognized []byte `json:"-"`
}

func (m *Response) Reset()         { *m = Response{} }
func (m *Response) String() string { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()    {}

func init() {
}
