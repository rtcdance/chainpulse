package grpc

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"net"

	"chainpulse/services/api/handlers/auth"
	"chainpulse/shared/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	// Import the generated protobuf code package
	// Since we can't generate it automatically, we'll define the interfaces here
)

// Define the protobuf-generated interfaces manually
type Event struct {
	Id          uint64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	BlockNumber string `protobuf:"bytes,2,opt,name=block_number,json=blockNumber,proto3" json:"blockNumber,omitempty"`
	TxHash      string `protobuf:"bytes,3,opt,name=tx_hash,json=txHash,proto3" json:"txHash,omitempty"`
	EventName   string `protobuf:"bytes,4,opt,name=event_name,json=eventName,proto3" json:"eventName,omitempty"`
	Contract    string `protobuf:"bytes,5,opt,name=contract,proto3" json:"contract,omitempty"`
	From        string `protobuf:"bytes,6,opt,name=from,proto3" json:"from,omitempty"`
	To          string `protobuf:"bytes,7,opt,name=to,proto3" json:"to,omitempty"`
	TokenId     string `protobuf:"bytes,8,opt,name=token_id,json=tokenId,proto3" json:"tokenId,omitempty"`
	Value       string `protobuf:"bytes,9,opt,name=value,proto3" json:"value,omitempty"`
	Timestamp   int64  `protobuf:"varint,10,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	CreatedAt   int64  `protobuf:"varint,11,opt,name=created_at,json=createdAt,proto3" json:"createdAt,omitempty"`
	UpdatedAt   int64  `protobuf:"varint,12,opt,name=updated_at,json=updatedAt,proto3" json:"updatedAt,omitempty"`
}

type EventFilter struct {
	EventType string `protobuf:"bytes,1,opt,name=event_type,json=eventType,proto3" json:"eventType,omitempty"`
	Contract  string `protobuf:"bytes,2,opt,name=contract,proto3" json:"contract,omitempty"`
	FromBlock string `protobuf:"bytes,3,opt,name=from_block,json=fromBlock,proto3" json:"fromBlock,omitempty"`
	ToBlock   string `protobuf:"bytes,4,opt,name=to_block,json=toBlock,proto3" json:"toBlock,omitempty"`
	Limit     int32  `protobuf:"varint,5,opt,name=limit,proto3" json:"limit,omitempty"`
	Offset    int32  `protobuf:"varint,6,opt,name=offset,proto3" json:"offset,omitempty"`
}

type GetEventRequest struct {
	Id uint64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
}

type GetEventResponse struct {
	Event *Event `protobuf:"bytes,1,opt,name=event,proto3" json:"event,omitempty"`
}

type GetEventsRequest struct {
	Filter *EventFilter `protobuf:"bytes,1,opt,name=filter,proto3" json:"filter,omitempty"`
}

type GetEventsResponse struct {
	Events []*Event `protobuf:"bytes,1,rep,name=events,proto3" json:"events,omitempty"`
}

type GetNFTEventsRequest struct {
	Filter *EventFilter `protobuf:"bytes,1,opt,name=filter,proto3" json:"filter,omitempty"`
}

type GetNFTEventsResponse struct {
	Events []*Event `protobuf:"bytes,1,rep,name=events,proto3" json:"events,omitempty"`
}

type GetTokenEventsRequest struct {
	Filter *EventFilter `protobuf:"bytes,1,opt,name=filter,proto3" json:"filter,omitempty"`
}

type GetTokenEventsResponse struct {
	Events []*Event `protobuf:"bytes,1,rep,name=events,proto3" json:"events,omitempty"`
}

type GetEventsByBlockRangeRequest struct {
	FromBlock string `protobuf:"bytes,1,opt,name=from_block,json=fromBlock,proto3" json:"fromBlock,omitempty"`
	ToBlock   string `protobuf:"bytes,2,opt,name=to_block,json=toBlock,proto3" json:"toBlock,omitempty"`
}

type GetEventsByBlockRangeResponse struct {
	Events []*Event `protobuf:"bytes,1,rep,name=events,proto3" json:"events,omitempty"`
}

type GetLastProcessedBlockRequest struct {
}

type GetLastProcessedBlockResponse struct {
	BlockNumber string `protobuf:"bytes,1,opt,name=block_number,json=blockNumber,proto3" json:"blockNumber,omitempty"`
}

type ReplayEventsRequest struct {
	FromBlock string `protobuf:"bytes,1,opt,name=from_block,json=fromBlock,proto3" json:"fromBlock,omitempty"`
	ToBlock   string `protobuf:"bytes,2,opt,name=to_block,json=toBlock,proto3" json:"toBlock,omitempty"`
}

type ReplayEventsResponse struct {
	Success bool   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Message string `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
}

// EventServiceServer is the gRPC server implementation
type EventServiceServer struct {
	UnimplementedEventServiceServer
	IndexerService *service.IndexerService
	Auth           *auth.AuthMiddleware
	Metrics        *metrics.Metrics
}

// UnimplementedEventServiceServer defines the unimplemented methods
type UnimplementedEventServiceServer struct{}

// GetEvent returns a single event by ID
func (s *EventServiceServer) GetEvent(ctx context.Context, req *GetEventRequest) (*GetEventResponse, error) {
	startTime := time.Now()
	// TODO: Implement this method using the database
	// For now, return an empty response
	log.Printf("GetEvent called with ID: %d", req.Id)
	
	// This is a placeholder - in a real implementation, you would fetch from the database
	if s.Metrics != nil {
		duration := time.Since(startTime).Seconds()
		s.Metrics.RecordAPIRequest("GET", "/event.EventService/GetEvent", "200")
		s.Metrics.RecordAPIRequestDuration("GET", "/event.EventService/GetEvent", duration)
	}
	
	return &GetEventResponse{
		Event: nil, // Would fetch from DB
	}, nil
}

// GetEvents returns multiple events based on filters
func (s *EventServiceServer) GetEvents(ctx context.Context, req *GetEventsRequest) (*GetEventsResponse, error) {
	startTime := time.Now()
	log.Printf("GetEvents called with filter: %+v", req.Filter)
	
	// This is a placeholder - in a real implementation, you would fetch from the database
	if s.Metrics != nil {
		duration := time.Since(startTime).Seconds()
		s.Metrics.RecordAPIRequest("GET", "/event.EventService/GetEvents", "200")
		s.Metrics.RecordAPIRequestDuration("GET", "/event.EventService/GetEvents", duration)
	}
	
	return &GetEventsResponse{
		Events: []*Event{}, // Would fetch from DB
	}, nil
}

// GetNFTEvents returns NFT transfer events based on filters
func (s *EventServiceServer) GetNFTEvents(ctx context.Context, req *GetNFTEventsRequest) (*GetNFTEventsResponse, error) {
	startTime := time.Now()
	log.Printf("GetNFTEvents called with filter: %+v", req.Filter)
	
	// This is a placeholder - in a real implementation, you would fetch from the database
	if s.Metrics != nil {
		duration := time.Since(startTime).Seconds()
		s.Metrics.RecordAPIRequest("GET", "/event.EventService/GetNFTEvents", "200")
		s.Metrics.RecordAPIRequestDuration("GET", "/event.EventService/GetNFTEvents", duration)
	}
	
	return &GetNFTEventsResponse{
		Events: []*Event{}, // Would fetch from DB
	}, nil
}

// GetTokenEvents returns token transfer events based on filters
func (s *EventServiceServer) GetTokenEvents(ctx context.Context, req *GetTokenEventsRequest) (*GetTokenEventsResponse, error) {
	startTime := time.Now()
	log.Printf("GetTokenEvents called with filter: %+v", req.Filter)
	
	// This is a placeholder - in a real implementation, you would fetch from the database
	if s.Metrics != nil {
		duration := time.Since(startTime).Seconds()
		s.Metrics.RecordAPIRequest("GET", "/event.EventService/GetTokenEvents", "200")
		s.Metrics.RecordAPIRequestDuration("GET", "/event.EventService/GetTokenEvents", duration)
	}
	
	return &GetTokenEventsResponse{
		Events: []*Event{}, // Would fetch from DB
	}, nil
}

// GetEventsByBlockRange returns events within a block range
func (s *EventServiceServer) GetEventsByBlockRange(ctx context.Context, req *GetEventsByBlockRangeRequest) (*GetEventsByBlockRangeResponse, error) {
	startTime := time.Now()
	log.Printf("GetEventsByBlockRange called from %s to %s", req.FromBlock, req.ToBlock)
	
	// Convert string block numbers to big.Int
	fromBlock := new(big.Int)
	fromBlock.SetString(req.FromBlock, 10)
	
	toBlock := new(big.Int)
	toBlock.SetString(req.ToBlock, 10)
	
	// Get events from database
	events, err := s.IndexerService.Database.GetEventsByBlockRange(fromBlock, toBlock)
	if err != nil {
		if s.Metrics != nil {
			s.Metrics.IncrementError("grpc", "get_events_by_block_range_failed")
		}
		return nil, err
	}
	
	// Convert to protobuf format
	protoEvents := make([]*Event, len(events))
	for i, event := range events {
		protoEvents[i] = &Event{
			Id:          uint64(event.ID),
			BlockNumber: event.BlockNumber.String(),
			TxHash:      event.TxHash,
			EventName:   event.EventName,
			Contract:    event.Contract,
			From:        event.From,
			To:          event.To,
			TokenId:     event.TokenID,
			Value:       event.Value,
			Timestamp:   event.Timestamp.Unix(),
			CreatedAt:   event.CreatedAt.Unix(),
			UpdatedAt:   event.UpdatedAt.Unix(),
		}
	}
	
	if s.Metrics != nil {
		duration := time.Since(startTime).Seconds()
		s.Metrics.RecordAPIRequest("GET", "/event.EventService/GetEventsByBlockRange", "200")
		s.Metrics.RecordAPIRequestDuration("GET", "/event.EventService/GetEventsByBlockRange", duration)
	}
	
	return &GetEventsByBlockRangeResponse{
		Events: protoEvents,
	}, nil
}

// GetLastProcessedBlock returns the last processed block number
func (s *EventServiceServer) GetLastProcessedBlock(ctx context.Context, req *GetLastProcessedBlockRequest) (*GetLastProcessedBlockResponse, error) {
	startTime := time.Now()
	log.Println("GetLastProcessedBlock called")
	
	lastBlock, err := s.IndexerService.Resume.GetLastProcessedBlock()
	if err != nil {
		if s.Metrics != nil {
			s.Metrics.IncrementError("grpc", "get_last_processed_block_failed")
		}
		return nil, err
	}
	
	if s.Metrics != nil {
		duration := time.Since(startTime).Seconds()
		s.Metrics.RecordAPIRequest("GET", "/event.EventService/GetLastProcessedBlock", "200")
		s.Metrics.RecordAPIRequestDuration("GET", "/event.EventService/GetLastProcessedBlock", duration)
	}
	
	return &GetLastProcessedBlockResponse{
		BlockNumber: lastBlock.String(),
	}, nil
}

// ReplayEvents replays events from a specific block range
func (s *EventServiceServer) ReplayEvents(ctx context.Context, req *ReplayEventsRequest) (*ReplayEventsResponse, error) {
	startTime := time.Now()
	log.Printf("ReplayEvents called from %s to %s", req.FromBlock, req.ToBlock)
	
	// Convert string block numbers to big.Int
	fromBlock := new(big.Int)
	fromBlock.SetString(req.FromBlock, 10)
	
	toBlock := new(big.Int)
	toBlock.SetString(req.ToBlock, 10)
	
	// Call the resume service to replay events
	err := s.IndexerService.Resume.ReplayEvents(ctx, fromBlock, toBlock)
	if err != nil {
		if s.Metrics != nil {
			s.Metrics.IncrementError("grpc", "replay_events_failed")
		}
		return &ReplayEventsResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to replay events: %v", err),
		}, nil
	}
	
	if s.Metrics != nil {
		duration := time.Since(startTime).Seconds()
		s.Metrics.RecordAPIRequest("POST", "/event.EventService/ReplayEvents", "200")
		s.Metrics.RecordAPIRequestDuration("POST", "/event.EventService/ReplayEvents", duration)
	}
	
	return &ReplayEventsResponse{
		Success: true,
		Message: "Successfully replayed events",
	}, nil
}

// StartGRPCServer starts the gRPC server
func StartGRPCServer(indexerService *service.IndexerService, port string, jwtSecret string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	// Create auth middleware
	authMiddleware := auth.NewAuthMiddleware(jwtSecret)
	unaryInterceptor, streamInterceptor := authMiddleware.GetGRPCAuthInterceptors()

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
		grpc.StreamInterceptor(streamInterceptor),
	)
	eventServiceServer := &EventServiceServer{
		IndexerService: indexerService,
		Auth:           authMiddleware,
		Metrics:        indexerService.Metrics,
	}
	RegisterEventServiceServer(grpcServer, eventServiceServer)
	
	// Register reflection service for debugging tools
	reflection.Register(grpcServer)

	log.Printf("Starting gRPC server on port %s", port)
	return grpcServer.Serve(lis)
}

// RegisterEventServiceServer registers the service implementation
func RegisterEventServiceServer(s *grpc.Server, srv EventServiceServer) {
	s.RegisterService(&EventService_ServiceDesc, srv)
}

// EventService_ServiceDesc is the gRPC service description
var EventService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "event.EventService",
	HandlerType: (*EventServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetEvent",
			Handler:    _EventService_GetEvent_Handler,
		},
		{
			MethodName: "GetEvents",
			Handler:    _EventService_GetEvents_Handler,
		},
		{
			MethodName: "GetNFTEvents",
			Handler:    _EventService_GetNFTEvents_Handler,
		},
		{
			MethodName: "GetTokenEvents",
			Handler:    _EventService_GetTokenEvents_Handler,
		},
		{
			MethodName: "GetEventsByBlockRange",
			Handler:    _EventService_GetEventsByBlockRange_Handler,
		},
		{
			MethodName: "GetLastProcessedBlock",
			Handler:    _EventService_GetLastProcessedBlock_Handler,
		},
		{
			MethodName: "ReplayEvents",
			Handler:    _EventService_ReplayEvents_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}

// Handler functions
func _EventService_GetEvent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetEventRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EventServiceServer).GetEvent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/event.EventService/GetEvent",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EventServiceServer).GetEvent(ctx, req.(*GetEventRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EventService_GetEvents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetEventsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EventServiceServer).GetEvents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/event.EventService/GetEvents",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EventServiceServer).GetEvents(ctx, req.(*GetEventsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EventService_GetNFTEvents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetNFTEventsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EventServiceServer).GetNFTEvents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/event.EventService/GetNFTEvents",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EventServiceServer).GetNFTEvents(ctx, req.(*GetNFTEventsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EventService_GetTokenEvents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTokenEventsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EventServiceServer).GetTokenEvents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/event.EventService/GetTokenEvents",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EventServiceServer).GetTokenEvents(ctx, req.(*GetTokenEventsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EventService_GetEventsByBlockRange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetEventsByBlockRangeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EventServiceServer).GetEventsByBlockRange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/event.EventService/GetEventsByBlockRange",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EventServiceServer).GetEventsByBlockRange(ctx, req.(*GetEventsByBlockRangeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EventService_GetLastProcessedBlock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetLastProcessedBlockRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EventServiceServer).GetLastProcessedBlock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/event.EventService/GetLastProcessedBlock",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EventServiceServer).GetLastProcessedBlock(ctx, req.(*GetLastProcessedBlockRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EventService_ReplayEvents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReplayEventsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EventServiceServer).ReplayEvents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/event.EventService/ReplayEvents",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EventServiceServer).ReplayEvents(ctx, req.(*ReplayEventsRequest))
	}
	return interceptor(ctx, in, info, handler)
}