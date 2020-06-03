// Code generated by protoc-gen-go.
// source: github.com/hailo-platform/H2O/login-service/proto/readuser/readuser.proto
// DO NOT EDIT!

/*
Package com_hailo-platform/H2O_service_login_readuser is a generated protocol buffer package.

It is generated from these files:
	github.com/hailo-platform/H2O/login-service/proto/readuser/readuser.proto

It has these top-level messages:
	Request
	Response
*/
package com_hailo-platform/H2O_service_login_readuser

import proto "github.com/hailo-platform/H2O/protobuf/proto"
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
	Application             *string  `protobuf:"bytes,1,req,name=application" json:"application,omitempty"`
	Uid                     *string  `protobuf:"bytes,2,req,name=uid" json:"uid,omitempty"`
	Ids                     []string `protobuf:"bytes,3,rep,name=ids" json:"ids,omitempty"`
	CreatedTimestamp        *int64   `protobuf:"varint,4,req,name=createdTimestamp" json:"createdTimestamp,omitempty"`
	Roles                   []string `protobuf:"bytes,5,rep,name=roles" json:"roles,omitempty"`
	Password                *string  `protobuf:"bytes,6,opt,name=password" json:"password,omitempty"`
	PasswordChangeTimestamp *int64   `protobuf:"varint,7,opt,name=passwordChangeTimestamp" json:"passwordChangeTimestamp,omitempty"`
	AccountExpirationDate   *string  `protobuf:"bytes,8,opt,name=accountExpirationDate" json:"accountExpirationDate,omitempty"`
	XXX_unrecognized        []byte   `json:"-"`
}

func (m *Response) Reset()         { *m = Response{} }
func (m *Response) String() string { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()    {}

func (m *Response) GetApplication() string {
	if m != nil && m.Application != nil {
		return *m.Application
	}
	return ""
}

func (m *Response) GetUid() string {
	if m != nil && m.Uid != nil {
		return *m.Uid
	}
	return ""
}

func (m *Response) GetIds() []string {
	if m != nil {
		return m.Ids
	}
	return nil
}

func (m *Response) GetCreatedTimestamp() int64 {
	if m != nil && m.CreatedTimestamp != nil {
		return *m.CreatedTimestamp
	}
	return 0
}

func (m *Response) GetRoles() []string {
	if m != nil {
		return m.Roles
	}
	return nil
}

func (m *Response) GetPassword() string {
	if m != nil && m.Password != nil {
		return *m.Password
	}
	return ""
}

func (m *Response) GetPasswordChangeTimestamp() int64 {
	if m != nil && m.PasswordChangeTimestamp != nil {
		return *m.PasswordChangeTimestamp
	}
	return 0
}

func (m *Response) GetAccountExpirationDate() string {
	if m != nil && m.AccountExpirationDate != nil {
		return *m.AccountExpirationDate
	}
	return ""
}

func init() {
}
