// Code generated by protoc-gen-go.
// source: github.com/hailocab/h2/proto/login/proto/expirepassword/expirepassword.proto
// DO NOT EDIT!

/*
Package com_hailocab_service_login_expirepassword is a generated protocol buffer package.

It is generated from these files:
	github.com/hailocab/h2/proto/login/proto/expirepassword/expirepassword.proto

It has these top-level messages:
	Request
	Response
*/
package com_hailocab_service_login_expirepassword

import proto "github.com/hailocab/protobuf/proto"
import json "encoding/json"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

type Request struct {
	Application      *string `protobuf:"bytes,1,req,name=application" json:"application,omitempty"`
	Uid              *string `protobuf:"bytes,2,req,name=uid" json:"uid,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
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

type Response struct {
	XXX_unrecognized []byte `json:"-"`
}

func (m *Response) Reset()         { *m = Response{} }
func (m *Response) String() string { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()    {}

func init() {
}
