// Code generated by protoc-gen-go.
// source: github.com/hailo-platform/H2O/discovery-service/proto/unregister/unregister.proto
// DO NOT EDIT!

/*
Package com_hailo-platform/H2O_kernel_discovery_unregister is a generated protocol buffer package.

It is generated from these files:
	github.com/hailo-platform/H2O/discovery-service/proto/unregister/unregister.proto

It has these top-level messages:
	Request
	Response
*/
package com_hailo-platform/H2O_kernel_discovery_unregister

import proto "github.com/hailo-platform/H2O/protobuf/proto"
import json "encoding/json"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

type Request struct {
	InstanceId       *string `protobuf:"bytes,1,req,name=instanceId" json:"instanceId,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
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

type Response struct {
	XXX_unrecognized []byte `json:"-"`
}

func (m *Response) Reset()         { *m = Response{} }
func (m *Response) String() string { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()    {}

func init() {
}
