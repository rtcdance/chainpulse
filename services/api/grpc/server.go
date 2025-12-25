package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"chainpulse/shared/database"
	"chainpulse/shared/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server implements the gRPC IndexerService
type Server struct {
	UnimplementedIndexerServiceServer
	db *database.DB
}

// NewServer creates a new gRPC server instance
func NewServer(db *database.DB) *Server {
	return &Server{
		db: db,
	}
}

// GetEvents returns a list of events with pagination
func (s *Server) GetEvents(ctx context.Context, req *GetEventsRequest) (*GetEventsResponse, error) {
	page := int(req.GetPage())
	limit := int(req.GetLimit())

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	offset := (page - 1) * limit

	events, err := s.db.GetEvents(limit, offset)
	if err != nil {
		return nil, err
	}

	protoEvents := make([]*Event, len(events))
	for i, event := range events {
		protoEvents[i] = convertToProtoEvent(event)
	}

	return &GetEventsResponse{
		Events: protoEvents,
		Page:   int32(page),
		Limit:  int32(limit),
		Total:  int64(len(events)),
	}, nil
}

// GetEventByTxHash returns an event by its transaction hash
func (s *Server) GetEventByTxHash(ctx context.Context, req *GetEventByTxHashRequest) (*GetEventByTxHashResponse, error) {
	txHash := req.GetTxHash()
	if txHash == "" {
		return nil, fmt.Errorf("tx_hash is required")
	}

	event, err := s.db.GetEventByTxHash(txHash)
	if err != nil {
		return nil, err
	}

	if event == nil {
		return &GetEventByTxHashResponse{}, nil
	}

	return &GetEventByTxHashResponse{
		Event: convertToProtoEvent(*event),
	}, nil
}

// GetEventsByBlockNumber returns events from a specific block number
func (s *Server) GetEventsByBlockNumber(ctx context.Context, req *GetEventsByBlockNumberRequest) (*GetEventsByBlockNumberResponse, error) {
	blockNumber := req.GetBlockNumber()

	events, err := s.db.GetEventsByBlockNumber(blockNumber)
	if err != nil {
		return nil, err
	}

	protoEvents := make([]*Event, len(events))
	for i, event := range events {
		protoEvents[i] = convertToProtoEvent(event)
	}

	return &GetEventsByBlockNumberResponse{
		Events:      protoEvents,
		BlockNumber: blockNumber,
		Total:       int64(len(events)),
	}, nil
}

// GetContracts returns a list of contracts
func (s *Server) GetContracts(ctx context.Context, req *GetContractsRequest) (*GetContractsResponse, error) {
	contracts, err := s.db.GetContracts()
	if err != nil {
		return nil, err
	}

	protoContracts := make([]*Contract, len(contracts))
	for i, contract := range contracts {
		protoContracts[i] = convertToProtoContract(contract)
	}

	return &GetContractsResponse{
		Contracts: protoContracts,
		Total:     int64(len(contracts)),
	}, nil
}

// GetContractByAddress returns a contract by its address
func (s *Server) GetContractByAddress(ctx context.Context, req *GetContractByAddressRequest) (*GetContractByAddressResponse, error) {
	address := req.GetAddress()
	if address == "" {
		return nil, fmt.Errorf("address is required")
	}

	contract, err := s.db.GetContractByAddress(address)
	if err != nil {
		return nil, err
	}

	if contract == nil {
		return &GetContractByAddressResponse{}, nil
	}

	return &GetContractByAddressResponse{
		Contract: convertToProtoContract(*contract),
	}, nil
}

// GetStats returns indexer statistics
func (s *Server) GetStats(ctx context.Context, req *GetStatsRequest) (*GetStatsResponse, error) {
	stats, err := s.db.GetStats()
	if err != nil {
		return nil, err
	}

	return &GetStatsResponse{
		TotalEvents:    stats.TotalEvents,
		TotalContracts: stats.TotalContracts,
		LatestBlock:    stats.LatestBlock,
	}, nil
}

// Health returns the health status of the service
func (s *Server) Health(ctx context.Context, req *HealthRequest) (*HealthResponse, error) {
	return &HealthResponse{
		Status:  "healthy",
		Service: "indexer-grpc",
		Time:    time.Now().Format(time.RFC3339),
	}, nil
}

// convertToProtoEvent converts an IndexedEvent to a protobuf Event
func convertToProtoEvent(event types.IndexedEvent) *Event {
	return &Event{
		Id:          uint32(event.ID),
		BlockNumber: event.BlockNumber.String(),
		TxHash:      event.TxHash,
		EventName:   event.EventName,
		Contract:    event.Contract,
		From:        event.From,
		To:          event.To,
		TokenId:     event.TokenID,
		Value:       event.Value,
		Timestamp:   event.Timestamp.Unix(),
	}
}

// convertToProtoContract converts a Contract to a protobuf Contract
func convertToProtoContract(contract types.Contract) *Contract {
	return &Contract{
		Id:        uint32(contract.ID),
		Address:   contract.Address,
		Name:      contract.Name,
		Symbol:    contract.Symbol,
		Type:      contract.Type,
		CreatedAt: contract.CreatedAt.Unix(),
		UpdatedAt: contract.UpdatedAt.Unix(),
	}
}

// StartGRPCServer starts the gRPC server on the specified port
func (s *Server) StartGRPCServer(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	RegisterIndexerServiceServer(grpcServer, s)
	
	// Register reflection service for debugging
	reflection.Register(grpcServer)

	log.Printf("Starting gRPC server on port %s", port)
	return grpcServer.Serve(lis)
}