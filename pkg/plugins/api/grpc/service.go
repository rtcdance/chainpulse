package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"chainpulse/pkg/core"
	"chainpulse/pkg/services/query"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ChainPulseService implements the ChainPulseAPI gRPC service.
// It delegates to the EventRetrievalService and HealthChecker.
type ChainPulseService struct {
	retrievalService *query.EventRetrievalService
	healthChecker    HealthChecker
	subscriptionMgr  SubscriptionManager
}

// HealthChecker interface for health check delegation
type HealthChecker interface {
	IsHealthy(ctx context.Context) bool
	StatusMessage(ctx context.Context) string
}

// SubscriptionManager interface for event subscriptions
type SubscriptionManager interface {
	Subscribe(topic string) *Subscription
	Unsubscribe(sub *Subscription)
}

// Subscription represents an active subscription
type Subscription struct {
	ID      string
	Channel chan interface{}
	Cancel  context.CancelFunc
}

// NewChainPulseService creates a new gRPC service implementation
func NewChainPulseService(
	retrievalService *query.EventRetrievalService,
	healthChecker HealthChecker,
	subscriptionMgr SubscriptionManager,
) *ChainPulseService {
	return &ChainPulseService{
		retrievalService: retrievalService,
		healthChecker:    healthChecker,
		subscriptionMgr:  subscriptionMgr,
	}
}

// RegisterWithServer registers the ChainPulseAPI service with a gRPC server.
func (s *ChainPulseService) RegisterWithServer(server *grpc.Server) {
	desc := &grpc.ServiceDesc{
		ServiceName: "chainpulse.ChainPulseAPI",
		HandlerType: (*ChainPulseService)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "GetEvent",
				Handler:    s.methodGetEvent,
			},
			{
				MethodName: "GetTokenBalance",
				Handler:    s.methodGetTokenBalance,
			},
			{
				MethodName: "HealthCheck",
				Handler:    s.methodHealthCheck,
			},
		},
		Streams: []grpc.StreamDesc{
			{
				StreamName:    "ListEvents",
				Handler:       s.streamListEvents,
				ServerStreams: true,
			},
			{
				StreamName:    "SubscribeEvents",
				Handler:       s.streamSubscribeEvents,
				ServerStreams: true,
			},
		},
		Metadata: "chainpulse.proto",
	}
	server.RegisterService(desc, s)
}

// --- Proto message types (JSON-encoded, matching chainpulse.proto) ---

// EventMessage represents a blockchain event in gRPC responses
type EventMessage struct {
	ID              string `json:"id"`
	Blockchain      string `json:"blockchain"`
	ContractAddress string `json:"contract_address"`
	EventName       string `json:"event_name"`
	BlockNumber     int64  `json:"block_number"`
	TransactionHash string `json:"transaction_hash"`
	LogIndex        string `json:"log_index"`
	Data            string `json:"data"`
	Timestamp       int64  `json:"timestamp"`
}

// BalanceMessage represents a token balance
type BalanceMessage struct {
	TokenAddress   string `json:"token_address"`
	AccountAddress string `json:"account_address"`
	Balance        string `json:"balance"`
}

// HealthCheckResponseMessage represents a health check response
type HealthCheckResponseMessage struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// --- Unary RPC Handlers (MethodHandler signature: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error)) ---

func (s *ChainPulseService) methodGetEvent(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) { //nolint:revive // gRPC handler signature
	var request struct {
		EventID string `json:"event_id"`
	}
	if err := dec(&request); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	if request.EventID == "" {
		return nil, status.Error(codes.InvalidArgument, "event_id is required")
	}

	result, err := s.retrievalService.GetEventWithMetadata(ctx, request.EventID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get event: %v", err)
	}
	if result == nil || result.Event == nil {
		return nil, status.Error(codes.NotFound, "event not found")
	}

	msg := eventToMessage(result.Event)
	data, _ := json.Marshal(msg)
	return data, nil
}

func (s *ChainPulseService) methodGetTokenBalance(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) { //nolint:revive // gRPC handler signature
	return nil, status.Error(codes.Unimplemented, "GetTokenBalance is not yet implemented")
}

func (s *ChainPulseService) methodHealthCheck(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) { //nolint:revive // gRPC handler signature
	resp := &HealthCheckResponseMessage{
		Status:  "healthy",
		Message: "service is operational",
	}
	if s.healthChecker != nil && !s.healthChecker.IsHealthy(ctx) {
		resp.Status = "unhealthy"
		resp.Message = s.healthChecker.StatusMessage(ctx)
	}

	data, _ := json.Marshal(resp)
	return data, nil
}

// --- Server Streaming RPC Handlers (StreamHandler signature: func(srv any, stream grpc.ServerStream) error) ---

func (s *ChainPulseService) streamListEvents(srv any, stream grpc.ServerStream) error {
	// Decode request from stream context metadata or initial message
	// For simplicity, use default pagination when no filter is provided
	limit := 100
	offset := 0
	blockchain := ""

	ctx := stream.Context()

	// Try to read request from gRPC metadata (custom convention for streaming RPCs without proto-generated code)
	// Clients can pass filters via gRPC metadata
	md := extractMetadata(ctx)
	if v := md["limit"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := md["offset"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	if v := md["blockchain"]; v != "" {
		blockchain = v
	}

	if limit > 1000 {
		limit = 1000
	}

	var events []*core.BlockchainEvent

	if blockchain != "" {
		chainID, _ := strconv.Atoi(blockchain)
		result, e := s.retrievalService.GetEventsByChainWithMetadata(ctx, chainID, limit, offset)
		if e != nil {
			return status.Errorf(codes.Internal, "failed to query events: %v", e)
		}
		for _, r := range result {
			events = append(events, r.Event)
		}
	} else {
		result, e := s.retrievalService.GetEventsByChainWithMetadata(ctx, 0, limit, offset)
		if e != nil {
			return status.Errorf(codes.Internal, "failed to query events: %v", e)
		}
		for _, r := range result {
			events = append(events, r.Event)
		}
	}

	for _, event := range events {
		msg := eventToMessage(event)
		data, marshalErr := json.Marshal(msg)
		if marshalErr != nil {
			continue
		}
		if sendErr := stream.SendMsg(data); sendErr != nil {
			return sendErr
		}
	}

	return nil
}

func (s *ChainPulseService) streamSubscribeEvents(srv any, stream grpc.ServerStream) error {
	if s.subscriptionMgr == nil {
		return status.Error(codes.Unimplemented, "subscriptions are not available")
	}

	topic := "event:created"
	ctx := stream.Context()

	// Allow topic filter via gRPC metadata
	md := extractMetadata(ctx)
	if v := md["event_name"]; v != "" {
		topic = "event:" + v
	}

	sub := s.subscriptionMgr.Subscribe(topic)
	defer s.subscriptionMgr.Unsubscribe(sub)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case data, ok := <-sub.Channel:
			if !ok {
				return nil
			}
			msgData, marshalErr := json.Marshal(data)
			if marshalErr != nil {
				continue
			}
			if sendErr := stream.SendMsg(msgData); sendErr != nil {
				if sendErr == io.EOF {
					return nil
				}
				return sendErr
			}
		}
	}
}

// --- Helper functions ---

func eventToMessage(event *core.BlockchainEvent) *EventMessage {
	data := "{}"
	if event.DecodedData != nil {
		if b, err := json.Marshal(event.DecodedData); err == nil {
			data = string(b)
		}
	}

	return &EventMessage{
		ID:              event.ID,
		Blockchain:      event.ChainID,
		ContractAddress: event.ContractAddress.Hex(),
		EventName:       event.EventName,
		BlockNumber:     int64(event.BlockNumber),
		TransactionHash: event.TransactionHash.Hex(),
		LogIndex:        strconv.Itoa(int(event.LogIndex)),
		Data:            data,
		Timestamp:       event.BlockTimestamp,
	}
}

// extractMetadata extracts a simple key-value map from gRPC metadata
func extractMetadata(ctx context.Context) map[string]string {
	result := make(map[string]string)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return result
	}
	for key, values := range md {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}

// DefaultHealthChecker provides a basic health checker implementation
type DefaultHealthChecker struct {
	dbCheck func(ctx context.Context) bool
}

// NewDefaultHealthChecker creates a health checker with a database check function
func NewDefaultHealthChecker(dbCheck func(ctx context.Context) bool) *DefaultHealthChecker {
	return &DefaultHealthChecker{dbCheck: dbCheck}
}

// IsHealthy returns true if the database is accessible
func (h *DefaultHealthChecker) IsHealthy(ctx context.Context) bool {
	if h.dbCheck == nil {
		return true
	}
	return h.dbCheck(ctx)
}

// StatusMessage returns a human-readable health status
func (h *DefaultHealthChecker) StatusMessage(ctx context.Context) string {
	if h.IsHealthy(ctx) {
		return "all components healthy"
	}
	return "database connection failed"
}

// String implements fmt.Stringer for debugging
func (e *EventMessage) String() string {
	return fmt.Sprintf("Event{id=%s, name=%s, block=%d}", e.ID, e.EventName, e.BlockNumber)
}

// Ensure imports are used
var (
	_ = json.Marshal
	_ = strconv.Itoa
	_ = fmt.Sprintf
	_ = io.EOF
	_ = status.Error
	_ = codes.OK
)
