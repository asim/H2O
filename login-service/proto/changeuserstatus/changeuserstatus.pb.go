// Code generated by protoc-gen-go.
// source: github.com/hailo-platform/H2O/login-service/proto/changeuserstatus/changeuserstatus.proto
// DO NOT EDIT!

/*
Package com_hailo-platform/H2O_service_login_changeuserstatus is a generated protocol buffer package.

It is generated from these files:
	github.com/hailo-platform/H2O/login-service/proto/changeuserstatus/changeuserstatus.proto

It has these top-level messages:
	Request
	Response
*/
package com_hailo-platform/H2O_service_login_changeuserstatus

import proto "github.com/hailo-platform/H2O/protobuf/proto"
import json "encoding/json"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

type Request struct {
	Application *string `protobuf:"bytes,1,req,name=application" json:"application,omitempty"`
	Uid         *string `protobuf:"bytes,2,req,name=uid" json:"uid,omitempty"`
	// what is new status?
	Status *string `protobuf:"bytes,3,req,name=status" json:"status,omitempty"`
	// for the batch operations
	Uids             []string `protobuf:"bytes,4,rep,name=uids" json:"uids,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *Request) Reset()         { *m = Request{} }
func (m *Request) String() string { return proto.CompactTextString(m) }
func (*Request) ProtoMessage()    {}

func (m *Request) GetApplication() string {
	if m != nil && m.Application != nil {
		return *m.Application
	}
	return ""
}

func (m *Request) GetUid() string {
	if m != nil && m.Uid != nil {
		return *m.Uid
	}
	return ""
}

func (m *Request) GetStatus() string {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return ""
}

func (m *Request) GetUids() []string {
	if m != nil {
		return m.Uids
	}
	return nil
}

// Response is empty if the call was successful
type Response struct {
	Uids             []string `protobuf:"bytes,1,rep,name=uids" json:"uids,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *Response) Reset()         { *m = Response{} }
func (m *Response) String() string { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()    {}

func (m *Response) GetUids() []string {
	if m != nil {
		return m.Uids
	}
	return nil
}

func init() {
}
