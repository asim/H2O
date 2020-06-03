// Code generated by protoc-gen-go.
// source: github.com/hailo-platform/H2O/binding-service/proto/setupbroker/setupbroker.proto
// DO NOT EDIT!

/*
Package com_hailo-platform/H2O_kernel_binding_setupbroker is a generated protocol buffer package.

It is generated from these files:
	github.com/hailo-platform/H2O/binding-service/proto/setupbroker/setupbroker.proto

It has these top-level messages:
	Request
	Response
*/
package com_hailo-platform/H2O_kernel_binding_setupbroker

import proto "github.com/hailo-platform/H2O/protobuf/proto"
import json "encoding/json"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

type Request struct {
	Hostname         *string `protobuf:"bytes,1,req,name=hostname" json:"hostname,omitempty"`
	Port             *int32  `protobuf:"varint,2,req,name=port" json:"port,omitempty"`
	Azname           *string `protobuf:"bytes,3,opt,name=azname" json:"azname,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Request) Reset()         { *m = Request{} }
func (m *Request) String() string { return proto.CompactTextString(m) }
func (*Request) ProtoMessage()    {}

func (m *Request) GetHostname() string {
	if m != nil && m.Hostname != nil {
		return *m.Hostname
	}
	return ""
}

func (m *Request) GetPort() int32 {
	if m != nil && m.Port != nil {
		return *m.Port
	}
	return 0
}

func (m *Request) GetAzname() string {
	if m != nil && m.Azname != nil {
		return *m.Azname
	}
	return ""
}

type Response struct {
	Ok               *bool  `protobuf:"varint,1,req,name=ok" json:"ok,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *Response) Reset()         { *m = Response{} }
func (m *Response) String() string { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()    {}

func (m *Response) GetOk() bool {
	if m != nil && m.Ok != nil {
		return *m.Ok
	}
	return false
}

func init() {
}
