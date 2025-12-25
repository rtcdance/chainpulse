package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Generated protocol buffer enums

// Generated protocol buffer constants

// IndexerServiceClient is the client API for IndexerService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to the documentation
// for ServiceClient.
type IndexerServiceClient interface {
	// Get events with pagination
	GetEvents(ctx context.Context, in *GetEventsRequest, opts ...grpc.CallOption) (*GetEventsResponse, error)
	// Get event by transaction hash
	GetEventByTxHash(ctx context.Context, in *GetEventByTxHashRequest, opts ...grpc.CallOption) (*GetEventByTxHashResponse, error)
	// Get events by block number
	GetEventsByBlockNumber(ctx context.Context, in *GetEventsByBlockNumberRequest, opts ...grpc.CallOption) (*GetEventsByBlockNumberResponse, error)
	// Get contracts
	GetContracts(ctx context.Context, in *GetContractsRequest, opts ...grpc.CallOption) (*GetContractsResponse, error)
	// Get contract by address
	GetContractByAddress(ctx context.Context, in *GetContractByAddressRequest, opts ...grpc.CallOption) (*GetContractByAddressResponse, error)
	// Get indexer statistics
	GetStats(ctx context.Context, in *GetStatsRequest, opts ...grpc.CallOption) (*GetStatsResponse, error)
	// Health check
	Health(ctx context.Context, in *HealthRequest, opts ...grpc.CallOption) (*HealthResponse, error)
}

// IndexerServiceServer is the server API for IndexerService service.
// All implementations must embed UnimplementedIndexerServiceServer
// for forward compatibility
type IndexerServiceServer interface {
	// Get events with pagination
	GetEvents(context.Context, *GetEventsRequest) (*GetEventsResponse, error)
	// Get event by transaction hash
	GetEventByTxHash(context.Context, *GetEventByTxHashRequest) (*GetEventByTxHashResponse, error)
	// Get events by block number
	GetEventsByBlockNumber(context.Context, *GetEventsByBlockNumberRequest) (*GetEventsByBlockNumberResponse, error)
	// Get contracts
	GetContracts(context.Context, *GetContractsRequest) (*GetContractsResponse, error)
	// Get contract by address
	GetContractByAddress(context.Context, *GetContractByAddressRequest) (*GetContractByAddressResponse, error)
	// Get indexer statistics
	GetStats(context.Context, *GetStatsRequest) (*GetStatsResponse, error)
	// Health check
	Health(context.Context, *HealthRequest) (*HealthResponse, error)
}

// UnimplementedIndexerServiceServer should be embedded to have forward compatible implementations.
type UnimplementedIndexerServiceServer struct {
}

func (UnimplementedIndexerServiceServer) GetEvents(context.Context, *GetEventsRequest) (*GetEventsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetEvents not implemented")
}
func (UnimplementedIndexerServiceServer) GetEventByTxHash(context.Context, *GetEventByTxHashRequest) (*GetEventByTxHashResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetEventByTxHash not implemented")
}
func (UnimplementedIndexerServiceServer) GetEventsByBlockNumber(context.Context, *GetEventsByBlockNumberRequest) (*GetEventsByBlockNumberResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetEventsByBlockNumber not implemented")
}
func (UnimplementedIndexerServiceServer) GetContracts(context.Context, *GetContractsRequest) (*GetContractsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetContracts not implemented")
}
func (UnimplementedIndexerServiceServer) GetContractByAddress(context.Context, *GetContractByAddressRequest) (*GetContractByAddressResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetContractByAddress not implemented")
}
func (UnimplementedIndexerServiceServer) GetStats(context.Context, *GetStatsRequest) (*GetStatsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetStats not implemented")
}
func (UnimplementedIndexerServiceServer) Health(context.Context, *HealthRequest) (*HealthResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Health not implemented")
}
func (UnimplementedIndexerServiceServer) testEmbeddedByUnimplemented() {}

// UnsafeIndexerServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to IndexerServiceServer will
// result in compilation errors.
type UnsafeIndexerServiceServer interface {
	// testEmbeddedByUnimplemented prevents us from accidentally including an
	// unimplemented code generator which would otherwise cause methods to be skipped
	testEmbeddedByUnimplemented()
}

func RegisterIndexerServiceServer(s grpc.ServiceRegistrar, srv IndexerServiceServer) {
	s.RegisterService(&IndexerService_ServiceDesc, srv)
}

func _IndexerService_GetEvents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetEventsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServiceServer).GetEvents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/indexer.IndexerService/GetEvents",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServiceServer).GetEvents(ctx, req.(*GetEventsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IndexerService_GetEventByTxHash_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetEventByTxHashRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServiceServer).GetEventByTxHash(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/indexer.IndexerService/GetEventByTxHash",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServiceServer).GetEventByTxHash(ctx, req.(*GetEventByTxHashRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IndexerService_GetEventsByBlockNumber_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetEventsByBlockNumberRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServiceServer).GetEventsByBlockNumber(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/indexer.IndexerService/GetEventsByBlockNumber",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServiceServer).GetEventsByBlockNumber(ctx, req.(*GetEventsByBlockNumberRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IndexerService_GetContracts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetContractsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServiceServer).GetContracts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/indexer.IndexerService/GetContracts",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServiceServer).GetContracts(ctx, req.(*GetContractsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IndexerService_GetContractByAddress_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetContractByAddressRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServiceServer).GetContractByAddress(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/indexer.IndexerService/GetContractByAddress",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServiceServer).GetContractByAddress(ctx, req.(*GetContractByAddressRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IndexerService_GetStats_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetStatsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServiceServer).GetStats(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/indexer.IndexerService/GetStats",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServiceServer).GetStats(ctx, req.(*GetStatsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IndexerService_Health_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HealthRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IndexerServiceServer).Health(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/indexer.IndexerService/Health",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IndexerServiceServer).Health(ctx, req.(*HealthRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// IndexerService_ServiceDesc is the grpc.ServiceDesc for IndexerService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var IndexerService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "indexer.IndexerService",
	HandlerType: (*IndexerServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetEvents",
			Handler:    _IndexerService_GetEvents_Handler,
		},
		{
			MethodName: "GetEventByTxHash",
			Handler:    _IndexerService_GetEventByTxHash_Handler,
		},
		{
			MethodName: "GetEventsByBlockNumber",
			Handler:    _IndexerService_GetEventsByBlockNumber_Handler,
		},
		{
			MethodName: "GetContracts",
			Handler:    _IndexerService_GetContracts_Handler,
		},
		{
			MethodName: "GetContractByAddress",
			Handler:    _IndexerService_GetContractByAddress_Handler,
		},
		{
			MethodName: "GetStats",
			Handler:    _IndexerService_GetStats_Handler,
		},
		{
			MethodName: "Health",
			Handler:    _IndexerService_Health_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "indexer.proto",
}

// Request/Response messages for events
type GetEventsRequest struct {
	Page  int32 `protobuf:"varint,1,opt,name=page,proto3" json:"page,omitempty"`
	Limit int32 `protobuf:"varint,2,opt,name=limit,proto3" json:"limit,omitempty"`
}

type GetEventsResponse struct {
	Events []*Event `protobuf:"bytes,1,rep,name=events,proto3" json:"events,omitempty"`
	Page   int32    `protobuf:"varint,2,opt,name=page,proto3" json:"page,omitempty"`
	Limit  int32    `protobuf:"varint,3,opt,name=limit,proto3" json:"limit,omitempty"`
	Total  int64    `protobuf:"varint,4,opt,name=total,proto3" json:"total,omitempty"`
}

type GetEventByTxHashRequest struct {
	TxHash string `protobuf:"bytes,1,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash,omitempty"`
}

type GetEventByTxHashResponse struct {
	Event *Event `protobuf:"bytes,1,opt,name=event,proto3" json:"event,omitempty"`
}

type GetEventsByBlockNumberRequest struct {
	BlockNumber int64 `protobuf:"varint,1,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
}

type GetEventsByBlockNumberResponse struct {
	Events      []*Event `protobuf:"bytes,1,rep,name=events,proto3" json:"events,omitempty"`
	BlockNumber int64    `protobuf:"varint,2,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
	Total       int64    `protobuf:"varint,3,opt,name=total,proto3" json:"total,omitempty"`
}

// Request/Response messages for contracts
type GetContractsRequest struct{}

type GetContractsResponse struct {
	Contracts []*Contract `protobuf:"bytes,1,rep,name=contracts,proto3" json:"contracts,omitempty"`
	Total     int64       `protobuf:"varint,2,opt,name=total,proto3" json:"total,omitempty"`
}

type GetContractByAddressRequest struct {
	Address string `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
}

type GetContractByAddressResponse struct {
	Contract *Contract `protobuf:"bytes,1,opt,name=contract,proto3" json:"contract,omitempty"`
}

// Request/Response messages for stats
type GetStatsRequest struct{}

type GetStatsResponse struct {
	TotalEvents    int64 `protobuf:"varint,1,opt,name=total_events,json=totalEvents,proto3" json:"total_events,omitempty"`
	TotalContracts int64 `protobuf:"varint,2,opt,name=total_contracts,json=totalContracts,proto3" json:"total_contracts,omitempty"`
	LatestBlock    int64 `protobuf:"varint,3,opt,name=latest_block,json=latestBlock,proto3" json:"latest_block,omitempty"`
}

// Request/Response messages for health
type HealthRequest struct{}

type HealthResponse struct {
	Status  string `protobuf:"bytes,1,opt,name=status,proto3" json:"status,omitempty"`
	Service string `protobuf:"bytes,2,opt,name=service,proto3" json:"service,omitempty"`
	Time    string `protobuf:"bytes,3,opt,name=time,proto3" json:"time,omitempty"`
}

// Common data types
type Event struct {
	Id          uint32 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	BlockNumber string `protobuf:"bytes,2,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
	TxHash      string `protobuf:"bytes,3,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash,omitempty"`
	EventName   string `protobuf:"bytes,4,opt,name=event_name,json=eventName,proto3" json:"event_name,omitempty"`
	Contract    string `protobuf:"bytes,5,opt,name=contract,proto3" json:"contract,omitempty"`
	From        string `protobuf:"bytes,6,opt,name=from,proto3" json:"from,omitempty"`
	To          string `protobuf:"bytes,7,opt,name=to,proto3" json:"to,omitempty"`
	TokenId     string `protobuf:"bytes,8,opt,name=token_id,json=tokenId,proto3" json:"token_id,omitempty"`
	Value       string `protobuf:"bytes,9,opt,name=value,proto3" json:"value,omitempty"`
	Timestamp   int64  `protobuf:"varint,10,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
}

type Contract struct {
	Id        uint32 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Address   string `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"`
	Name      string `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	Symbol    string `protobuf:"bytes,4,opt,name=symbol,proto3" json:"symbol,omitempty"`
	Type      string `protobuf:"bytes,5,opt,name=type,proto3" json:"type,omitempty"`
	CreatedAt int64  `protobuf:"varint,11,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	UpdatedAt int64  `protobuf:"varint,12,opt,name=updated_at,json=updatedAt,proto3" json:"updated_at,omitempty"`
}