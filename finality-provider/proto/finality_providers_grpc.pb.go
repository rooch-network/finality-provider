// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: finality_providers.proto

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

const (
	FinalityProviders_GetInfo_FullMethodName                   = "/proto.FinalityProviders/GetInfo"
	FinalityProviders_CreateFinalityProvider_FullMethodName    = "/proto.FinalityProviders/CreateFinalityProvider"
	FinalityProviders_RegisterFinalityProvider_FullMethodName  = "/proto.FinalityProviders/RegisterFinalityProvider"
	FinalityProviders_AddFinalitySignature_FullMethodName      = "/proto.FinalityProviders/AddFinalitySignature"
	FinalityProviders_QueryFinalityProvider_FullMethodName     = "/proto.FinalityProviders/QueryFinalityProvider"
	FinalityProviders_QueryFinalityProviderList_FullMethodName = "/proto.FinalityProviders/QueryFinalityProviderList"
)

// FinalityProvidersClient is the client API for FinalityProviders service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type FinalityProvidersClient interface {
	// GetInfo returns the information of the daemon
	GetInfo(ctx context.Context, in *GetInfoRequest, opts ...grpc.CallOption) (*GetInfoResponse, error)
	// CreateFinalityProvider generates and saves a finality provider object
	CreateFinalityProvider(ctx context.Context, in *CreateFinalityProviderRequest, opts ...grpc.CallOption) (*CreateFinalityProviderResponse, error)
	// RegisterFinalityProvider sends a transactions to Babylon to register a BTC
	// finality provider
	RegisterFinalityProvider(ctx context.Context, in *RegisterFinalityProviderRequest, opts ...grpc.CallOption) (*RegisterFinalityProviderResponse, error)
	// AddFinalitySignature sends a transactions to Babylon to add a Finality
	// signature for a block
	AddFinalitySignature(ctx context.Context, in *AddFinalitySignatureRequest, opts ...grpc.CallOption) (*AddFinalitySignatureResponse, error)
	// QueryFinalityProvider queries the finality provider
	QueryFinalityProvider(ctx context.Context, in *QueryFinalityProviderRequest, opts ...grpc.CallOption) (*QueryFinalityProviderResponse, error)
	// QueryFinalityProviderList queries a list of finality providers
	QueryFinalityProviderList(ctx context.Context, in *QueryFinalityProviderListRequest, opts ...grpc.CallOption) (*QueryFinalityProviderListResponse, error)
}

type finalityProvidersClient struct {
	cc grpc.ClientConnInterface
}

func NewFinalityProvidersClient(cc grpc.ClientConnInterface) FinalityProvidersClient {
	return &finalityProvidersClient{cc}
}

func (c *finalityProvidersClient) GetInfo(ctx context.Context, in *GetInfoRequest, opts ...grpc.CallOption) (*GetInfoResponse, error) {
	out := new(GetInfoResponse)
	err := c.cc.Invoke(ctx, FinalityProviders_GetInfo_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *finalityProvidersClient) CreateFinalityProvider(ctx context.Context, in *CreateFinalityProviderRequest, opts ...grpc.CallOption) (*CreateFinalityProviderResponse, error) {
	out := new(CreateFinalityProviderResponse)
	err := c.cc.Invoke(ctx, FinalityProviders_CreateFinalityProvider_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *finalityProvidersClient) RegisterFinalityProvider(ctx context.Context, in *RegisterFinalityProviderRequest, opts ...grpc.CallOption) (*RegisterFinalityProviderResponse, error) {
	out := new(RegisterFinalityProviderResponse)
	err := c.cc.Invoke(ctx, FinalityProviders_RegisterFinalityProvider_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *finalityProvidersClient) AddFinalitySignature(ctx context.Context, in *AddFinalitySignatureRequest, opts ...grpc.CallOption) (*AddFinalitySignatureResponse, error) {
	out := new(AddFinalitySignatureResponse)
	err := c.cc.Invoke(ctx, FinalityProviders_AddFinalitySignature_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *finalityProvidersClient) QueryFinalityProvider(ctx context.Context, in *QueryFinalityProviderRequest, opts ...grpc.CallOption) (*QueryFinalityProviderResponse, error) {
	out := new(QueryFinalityProviderResponse)
	err := c.cc.Invoke(ctx, FinalityProviders_QueryFinalityProvider_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *finalityProvidersClient) QueryFinalityProviderList(ctx context.Context, in *QueryFinalityProviderListRequest, opts ...grpc.CallOption) (*QueryFinalityProviderListResponse, error) {
	out := new(QueryFinalityProviderListResponse)
	err := c.cc.Invoke(ctx, FinalityProviders_QueryFinalityProviderList_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FinalityProvidersServer is the server API for FinalityProviders service.
// All implementations must embed UnimplementedFinalityProvidersServer
// for forward compatibility
type FinalityProvidersServer interface {
	// GetInfo returns the information of the daemon
	GetInfo(context.Context, *GetInfoRequest) (*GetInfoResponse, error)
	// CreateFinalityProvider generates and saves a finality provider object
	CreateFinalityProvider(context.Context, *CreateFinalityProviderRequest) (*CreateFinalityProviderResponse, error)
	// RegisterFinalityProvider sends a transactions to Babylon to register a BTC
	// finality provider
	RegisterFinalityProvider(context.Context, *RegisterFinalityProviderRequest) (*RegisterFinalityProviderResponse, error)
	// AddFinalitySignature sends a transactions to Babylon to add a Finality
	// signature for a block
	AddFinalitySignature(context.Context, *AddFinalitySignatureRequest) (*AddFinalitySignatureResponse, error)
	// QueryFinalityProvider queries the finality provider
	QueryFinalityProvider(context.Context, *QueryFinalityProviderRequest) (*QueryFinalityProviderResponse, error)
	// QueryFinalityProviderList queries a list of finality providers
	QueryFinalityProviderList(context.Context, *QueryFinalityProviderListRequest) (*QueryFinalityProviderListResponse, error)
	mustEmbedUnimplementedFinalityProvidersServer()
}

// UnimplementedFinalityProvidersServer must be embedded to have forward compatible implementations.
type UnimplementedFinalityProvidersServer struct {
}

func (UnimplementedFinalityProvidersServer) GetInfo(context.Context, *GetInfoRequest) (*GetInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetInfo not implemented")
}
func (UnimplementedFinalityProvidersServer) CreateFinalityProvider(context.Context, *CreateFinalityProviderRequest) (*CreateFinalityProviderResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateFinalityProvider not implemented")
}
func (UnimplementedFinalityProvidersServer) RegisterFinalityProvider(context.Context, *RegisterFinalityProviderRequest) (*RegisterFinalityProviderResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterFinalityProvider not implemented")
}
func (UnimplementedFinalityProvidersServer) AddFinalitySignature(context.Context, *AddFinalitySignatureRequest) (*AddFinalitySignatureResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddFinalitySignature not implemented")
}
func (UnimplementedFinalityProvidersServer) QueryFinalityProvider(context.Context, *QueryFinalityProviderRequest) (*QueryFinalityProviderResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method QueryFinalityProvider not implemented")
}
func (UnimplementedFinalityProvidersServer) QueryFinalityProviderList(context.Context, *QueryFinalityProviderListRequest) (*QueryFinalityProviderListResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method QueryFinalityProviderList not implemented")
}
func (UnimplementedFinalityProvidersServer) mustEmbedUnimplementedFinalityProvidersServer() {}

// UnsafeFinalityProvidersServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to FinalityProvidersServer will
// result in compilation errors.
type UnsafeFinalityProvidersServer interface {
	mustEmbedUnimplementedFinalityProvidersServer()
}

func RegisterFinalityProvidersServer(s grpc.ServiceRegistrar, srv FinalityProvidersServer) {
	s.RegisterService(&FinalityProviders_ServiceDesc, srv)
}

func _FinalityProviders_GetInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FinalityProvidersServer).GetInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: FinalityProviders_GetInfo_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FinalityProvidersServer).GetInfo(ctx, req.(*GetInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FinalityProviders_CreateFinalityProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateFinalityProviderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FinalityProvidersServer).CreateFinalityProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: FinalityProviders_CreateFinalityProvider_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FinalityProvidersServer).CreateFinalityProvider(ctx, req.(*CreateFinalityProviderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FinalityProviders_RegisterFinalityProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterFinalityProviderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FinalityProvidersServer).RegisterFinalityProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: FinalityProviders_RegisterFinalityProvider_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FinalityProvidersServer).RegisterFinalityProvider(ctx, req.(*RegisterFinalityProviderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FinalityProviders_AddFinalitySignature_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddFinalitySignatureRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FinalityProvidersServer).AddFinalitySignature(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: FinalityProviders_AddFinalitySignature_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FinalityProvidersServer).AddFinalitySignature(ctx, req.(*AddFinalitySignatureRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FinalityProviders_QueryFinalityProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryFinalityProviderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FinalityProvidersServer).QueryFinalityProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: FinalityProviders_QueryFinalityProvider_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FinalityProvidersServer).QueryFinalityProvider(ctx, req.(*QueryFinalityProviderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FinalityProviders_QueryFinalityProviderList_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryFinalityProviderListRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FinalityProvidersServer).QueryFinalityProviderList(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: FinalityProviders_QueryFinalityProviderList_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FinalityProvidersServer).QueryFinalityProviderList(ctx, req.(*QueryFinalityProviderListRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// FinalityProviders_ServiceDesc is the grpc.ServiceDesc for FinalityProviders service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var FinalityProviders_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.FinalityProviders",
	HandlerType: (*FinalityProvidersServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetInfo",
			Handler:    _FinalityProviders_GetInfo_Handler,
		},
		{
			MethodName: "CreateFinalityProvider",
			Handler:    _FinalityProviders_CreateFinalityProvider_Handler,
		},
		{
			MethodName: "RegisterFinalityProvider",
			Handler:    _FinalityProviders_RegisterFinalityProvider_Handler,
		},
		{
			MethodName: "AddFinalitySignature",
			Handler:    _FinalityProviders_AddFinalitySignature_Handler,
		},
		{
			MethodName: "QueryFinalityProvider",
			Handler:    _FinalityProviders_QueryFinalityProvider_Handler,
		},
		{
			MethodName: "QueryFinalityProviderList",
			Handler:    _FinalityProviders_QueryFinalityProviderList_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "finality_providers.proto",
}
