// Code generated by protoc-gen-go.
// source: github.com/hailo-platform/H2O/login-service/proto/endpointauth/endpointauth.proto
// DO NOT EDIT!

/*
Package com_hailo-platform/H2O_service_login_endpointauth is a generated protocol buffer package.

It is generated from these files:
	github.com/hailo-platform/H2O/login-service/proto/endpointauth/endpointauth.proto

It has these top-level messages:
	Request
	Response
*/
package com_hailo-platform/H2O_service_login_endpointauth

import proto "github.com/hailo-platform/H2O/protobuf/proto"
import json "encoding/json"
import math "math"
import com_hailo-platform/H2O_service_login "github.com/hailo-platform/H2O/login-service/proto"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

type Request struct {
	Service          *string `protobuf:"bytes,1,req,name=service" json:"service,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Request) Reset()         { *m = Request{} }
func (m *Request) String() string { return proto.CompactTextString(m) }
func (*Request) ProtoMessage()    {}

func (m *Request) GetService() string {
	if m != nil && m.Service != nil {
		return *m.Service
	}
	return ""
}

type Response struct {
	Endpoints        []*com_hailo-platform/H2O_service_login.Endpoint `protobuf:"bytes,1,rep,name=endpoints" json:"endpoints,omitempty"`
	XXX_unrecognized []byte                                 `json:"-"`
}

func (m *Response) Reset()         { *m = Response{} }
func (m *Response) String() string { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()    {}

func (m *Response) GetEndpoints() []*com_hailo-platform/H2O_service_login.Endpoint {
	if m != nil {
		return m.Endpoints
	}
	return nil
}

func init() {
}
