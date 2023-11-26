// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v4.23.4
// source: raft.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// RaferServiceClient is the client API for RaferService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RaferServiceClient interface {
	Join(ctx context.Context, in *RaftClusterJoinRequest, opts ...grpc.CallOption) (*RaftClusterJoinResponse, error)
}

type raferServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewRaferServiceClient(cc grpc.ClientConnInterface) RaferServiceClient {
	return &raferServiceClient{cc}
}

func (c *raferServiceClient) Join(ctx context.Context, in *RaftClusterJoinRequest, opts ...grpc.CallOption) (*RaftClusterJoinResponse, error) {
	out := new(RaftClusterJoinResponse)
	err := c.cc.Invoke(ctx, "/RaferService/Join", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RaferServiceServer is the server API for RaferService service.
// All implementations must embed UnimplementedRaferServiceServer
// for forward compatibility
type RaferServiceServer interface {
	Join(context.Context, *RaftClusterJoinRequest) (*RaftClusterJoinResponse, error)
	mustEmbedUnimplementedRaferServiceServer()
}

// UnimplementedRaferServiceServer must be embedded to have forward compatible implementations.
type UnimplementedRaferServiceServer struct {
}

func (UnimplementedRaferServiceServer) Join(context.Context, *RaftClusterJoinRequest) (*RaftClusterJoinResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Join not implemented")
}
func (UnimplementedRaferServiceServer) mustEmbedUnimplementedRaferServiceServer() {}

// UnsafeRaferServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RaferServiceServer will
// result in compilation errors.
type UnsafeRaferServiceServer interface {
	mustEmbedUnimplementedRaferServiceServer()
}

func RegisterRaferServiceServer(s grpc.ServiceRegistrar, srv RaferServiceServer) {
	s.RegisterService(&RaferService_ServiceDesc, srv)
}

func _RaferService_Join_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RaftClusterJoinRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RaferServiceServer).Join(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/RaferService/Join",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RaferServiceServer).Join(ctx, req.(*RaftClusterJoinRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// RaferService_ServiceDesc is the grpc.ServiceDesc for RaferService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RaferService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "RaferService",
	HandlerType: (*RaferServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Join",
			Handler:    _RaferService_Join_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "raft.proto",
}
