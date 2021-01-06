// Code generated by protoc-gen-go. DO NOT EDIT.
// source: service.proto

/*
Package ygoservice is a generated protocol buffer package.

It is generated from these files:
	service.proto

It has these top-level messages:
	Request
	Reply
*/
package ygoservice

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Request struct {
	Method string            `protobuf:"bytes,1,opt,name=method" json:"method,omitempty"`
	Params map[string]string `protobuf:"bytes,2,rep,name=params" json:"params,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *Request) Reset()                    { *m = Request{} }
func (m *Request) String() string            { return proto.CompactTextString(m) }
func (*Request) ProtoMessage()               {}
func (*Request) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Request) GetMethod() string {
	if m != nil {
		return m.Method
	}
	return ""
}

func (m *Request) GetParams() map[string]string {
	if m != nil {
		return m.Params
	}
	return nil
}

type Reply struct {
	Response string `protobuf:"bytes,1,opt,name=response" json:"response,omitempty"`
}

func (m *Reply) Reset()                    { *m = Reply{} }
func (m *Reply) String() string            { return proto.CompactTextString(m) }
func (*Reply) ProtoMessage()               {}
func (*Reply) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *Reply) GetResponse() string {
	if m != nil {
		return m.Response
	}
	return ""
}

func init() {
	proto.RegisterType((*Request)(nil), "ygoservice.Request")
	proto.RegisterType((*Reply)(nil), "ygoservice.Reply")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for YGOService service

type YGOServiceClient interface {
	Call(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Reply, error)
}

type yGOServiceClient struct {
	cc *grpc.ClientConn
}

func NewYGOServiceClient(cc *grpc.ClientConn) YGOServiceClient {
	return &yGOServiceClient{cc}
}

func (c *yGOServiceClient) Call(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Reply, error) {
	out := new(Reply)
	err := grpc.Invoke(ctx, "/ygoservice.YGOService/Call", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for YGOService service

type YGOServiceServer interface {
	Call(context.Context, *Request) (*Reply, error)
}

func RegisterYGOServiceServer(s *grpc.Server, srv YGOServiceServer) {
	s.RegisterService(&_YGOService_serviceDesc, srv)
}

func _YGOService_Call_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YGOServiceServer).Call(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ygoservice.YGOService/Call",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YGOServiceServer).Call(ctx, req.(*Request))
	}
	return interceptor(ctx, in, info, handler)
}

var _YGOService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "ygoservice.YGOService",
	HandlerType: (*YGOServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Call",
			Handler:    _YGOService_Call_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "service.proto",
}

func init() { proto.RegisterFile("service.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 210 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x2d, 0x4e, 0x2d, 0x2a,
	0xcb, 0x4c, 0x4e, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0xaa, 0x4c, 0xcf, 0x87, 0x8a,
	0x28, 0x4d, 0x65, 0xe4, 0x62, 0x0f, 0x4a, 0x2d, 0x2c, 0x4d, 0x2d, 0x2e, 0x11, 0x12, 0xe3, 0x62,
	0xcb, 0x4d, 0x2d, 0xc9, 0xc8, 0x4f, 0x91, 0x60, 0x54, 0x60, 0xd4, 0xe0, 0x0c, 0x82, 0xf2, 0x84,
	0xcc, 0xb9, 0xd8, 0x0a, 0x12, 0x8b, 0x12, 0x73, 0x8b, 0x25, 0x98, 0x14, 0x98, 0x35, 0xb8, 0x8d,
	0xe4, 0xf5, 0x10, 0x06, 0xe8, 0x41, 0x35, 0xeb, 0x05, 0x80, 0x55, 0xb8, 0xe6, 0x95, 0x14, 0x55,
	0x06, 0x41, 0x95, 0x4b, 0x59, 0x72, 0x71, 0x23, 0x09, 0x0b, 0x09, 0x70, 0x31, 0x67, 0xa7, 0x56,
	0x42, 0x0d, 0x07, 0x31, 0x85, 0x44, 0xb8, 0x58, 0xcb, 0x12, 0x73, 0x4a, 0x53, 0x25, 0x98, 0xc0,
	0x62, 0x10, 0x8e, 0x15, 0x93, 0x05, 0xa3, 0x92, 0x32, 0x17, 0x6b, 0x50, 0x6a, 0x41, 0x4e, 0xa5,
	0x90, 0x14, 0x17, 0x47, 0x51, 0x6a, 0x71, 0x41, 0x7e, 0x5e, 0x71, 0x2a, 0x54, 0x27, 0x9c, 0x6f,
	0x64, 0xc7, 0xc5, 0x15, 0xe9, 0xee, 0x1f, 0x0c, 0x71, 0x89, 0x90, 0x01, 0x17, 0x8b, 0x73, 0x62,
	0x4e, 0x8e, 0x90, 0x30, 0x16, 0xe7, 0x49, 0x09, 0xa2, 0x0a, 0x16, 0xe4, 0x54, 0x2a, 0x31, 0x24,
	0xb1, 0x81, 0xc3, 0xc3, 0x18, 0x10, 0x00, 0x00, 0xff, 0xff, 0xb0, 0x48, 0xd0, 0xc3, 0x20, 0x01,
	0x00, 0x00,
}
