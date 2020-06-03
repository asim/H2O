// Code generated by protoc-gen-go.
// source: github.com/hailo-platform/H2O/login-service/proto/listusers/listusers.proto
// DO NOT EDIT!

/*
Package com_hailo-platform/H2O_service_login_listusers is a generated protocol buffer package.

It is generated from these files:
	github.com/hailo-platform/H2O/login-service/proto/listusers/listusers.proto

It has these top-level messages:
	Request
	Response
*/
package com_hailo-platform/H2O_service_login_listusers

import proto "github.com/hailo-platform/H2O/protobuf/proto"
import json "encoding/json"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

type Request struct {
	Application *string `protobuf:"bytes,1,req,name=application" json:"application,omitempty"`
	// specify a time range to search between
	RangeStart *int64 `protobuf:"varint,2,opt,name=rangeStart" json:"rangeStart,omitempty"`
	RangeEnd   *int64 `protobuf:"varint,3,opt,name=rangeEnd" json:"rangeEnd,omitempty"`
	// paginate
	LastId           *string `protobuf:"bytes,4,opt,name=lastId" json:"lastId,omitempty"`
	Count            *int64  `protobuf:"varint,5,opt,name=count,def=10" json:"count,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Request) Reset()         { *m = Request{} }
func (m *Request) String() string { return proto.CompactTextString(m) }
func (*Request) ProtoMessage()    {}

const Default_Request_Count int64 = 10

func (m *Request) GetApplication() string {
	if m != nil && m.Application != nil {
		return *m.Application
	}
	return ""
}

func (m *Request) GetRangeStart() int64 {
	if m != nil && m.RangeStart != nil {
		return *m.RangeStart
	}
	return 0
}

func (m *Request) GetRangeEnd() int64 {
	if m != nil && m.RangeEnd != nil {
		return *m.RangeEnd
	}
	return 0
}

func (m *Request) GetLastId() string {
	if m != nil && m.LastId != nil {
		return *m.LastId
	}
	return ""
}

func (m *Request) GetCount() int64 {
	if m != nil && m.Count != nil {
		return *m.Count
	}
	return Default_Request_Count
}

type Response struct {
	Application      *string          `protobuf:"bytes,1,req,name=application" json:"application,omitempty"`
	Users            []*Response_User `protobuf:"bytes,2,rep,name=users" json:"users,omitempty"`
	LastId           *string          `protobuf:"bytes,3,opt,name=lastId" json:"lastId,omitempty"`
	XXX_unrecognized []byte           `json:"-"`
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

func (m *Response) GetUsers() []*Response_User {
	if m != nil {
		return m.Users
	}
	return nil
}

func (m *Response) GetLastId() string {
	if m != nil && m.LastId != nil {
		return *m.LastId
	}
	return ""
}

type Response_User struct {
	Uid                     *string  `protobuf:"bytes,1,req,name=uid" json:"uid,omitempty"`
	Ids                     []string `protobuf:"bytes,2,rep,name=ids" json:"ids,omitempty"`
	CreatedTimestamp        *int64   `protobuf:"varint,3,req,name=createdTimestamp" json:"createdTimestamp,omitempty"`
	Roles                   []string `protobuf:"bytes,4,rep,name=roles" json:"roles,omitempty"`
	PasswordChangeTimestamp *int64   `protobuf:"varint,5,opt,name=passwordChangeTimestamp" json:"passwordChangeTimestamp,omitempty"`
	XXX_unrecognized        []byte   `json:"-"`
}

func (m *Response_User) Reset()         { *m = Response_User{} }
func (m *Response_User) String() string { return proto.CompactTextString(m) }
func (*Response_User) ProtoMessage()    {}

func (m *Response_User) GetUid() string {
	if m != nil && m.Uid != nil {
		return *m.Uid
	}
	return ""
}

func (m *Response_User) GetIds() []string {
	if m != nil {
		return m.Ids
	}
	return nil
}

func (m *Response_User) GetCreatedTimestamp() int64 {
	if m != nil && m.CreatedTimestamp != nil {
		return *m.CreatedTimestamp
	}
	return 0
}

func (m *Response_User) GetRoles() []string {
	if m != nil {
		return m.Roles
	}
	return nil
}

func (m *Response_User) GetPasswordChangeTimestamp() int64 {
	if m != nil && m.PasswordChangeTimestamp != nil {
		return *m.PasswordChangeTimestamp
	}
	return 0
}

func init() {
}
