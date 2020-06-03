// Code generated by protoc-gen-go.
// source: github.com/hailo-platform/H2O/provisioning-manager-service/proto/setrunlevel/setrunlevel.proto
// DO NOT EDIT!

/*
Package com_hailo-platform/H2O_kernel_provisioningmanager_setrunlevel is a generated protocol buffer package.

It is generated from these files:
	github.com/hailo-platform/H2O/provisioning-manager-service/proto/setrunlevel/setrunlevel.proto

It has these top-level messages:
	Request
	Response
*/
package com_hailo-platform/H2O_kernel_provisioningmanager_setrunlevel

import proto "github.com/hailo-platform/H2O/protobuf/proto"
import json "encoding/json"
import math "math"
import com_hailo-platform/H2O_kernel_provisioningmanager "github.com/hailo-platform/H2O/provisioning-manager-service/proto"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

type Request struct {
	Region           *string                                        `protobuf:"bytes,1,req,name=region" json:"region,omitempty"`
	Level            *com_hailo-platform/H2O_kernel_provisioningmanager.Level `protobuf:"varint,2,req,name=level,enum=com.hailo-platform/H2O.kernel.provisioningmanager.Level" json:"level,omitempty"`
	XXX_unrecognized []byte                                         `json:"-"`
}

func (m *Request) Reset()         { *m = Request{} }
func (m *Request) String() string { return proto.CompactTextString(m) }
func (*Request) ProtoMessage()    {}

func (m *Request) GetRegion() string {
	if m != nil && m.Region != nil {
		return *m.Region
	}
	return ""
}

func (m *Request) GetLevel() com_hailo-platform/H2O_kernel_provisioningmanager.Level {
	if m != nil && m.Level != nil {
		return *m.Level
	}
	return com_hailo-platform/H2O_kernel_provisioningmanager.Level_HALT
}

type Response struct {
	XXX_unrecognized []byte `json:"-"`
}

func (m *Response) Reset()         { *m = Response{} }
func (m *Response) String() string { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()    {}

func init() {
}
