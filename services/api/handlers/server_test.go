package handlers

import (
	"bytes"
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	json "github.com/goccy/go-json"

	"chainpulse/shared/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
)

// MockIndexerService for testing
type MockIndexerService struct {
	events []types.IndexedEvent
}

func (m *MockIndexerService) StartIndexing(ctx context.Context, contractAddresses []common.Address) error {
	return nil
}

func (m *MockIndexerService) ProcessHistoricalEvents(ctx context.Context, contractAddresses []common.Address, fromBlock, toBlock *big.Int) error {
	return nil
}

func (m *MockIndexerService) GetEvents(filter *types.EventFilter) ([]types.IndexedEvent, error) {
	return m.events, nil
}

func (m *MockIndexerService) GetEventByID(id uint) (*types.IndexedEvent, error) {
	if len(m.events) > 0 && int(id) <= len(m.events) {
		return &m.events[id-1], nil
	}
	return nil, nil
}

func (m *MockIndexerService) GetEventsByBlockRange(fromBlock, toBlock *big.Int) ([]types.IndexedEvent, error) {
	return m.events, nil
}

func (m *MockIndexerService) GetLastProcessedBlock() (*big.Int, error) {
	return big.NewInt(1000), nil
}

func (m *MockIndexerService) ResumeEvents(ctx context.Context, fromBlock, toBlock *big.Int) error {
	return nil
}

func TestNewServer(t *testing.T) {
	mockIndexerService := &MockIndexerService{}
	
	server := NewServer(mockIndexerService, "test-secret")
	
	if server == nil {
		t.Error("Expected Server instance, got nil")
	}
	
	if server.indexerService == nil {
		t.Error("Expected indexerService to be set")
	}
	
	if server.router == nil {
		t.Error("Expected router to be initialized")
	}
}

func TestGetEventsHandler(t *testing.T) {
	mockIndexerService := &MockIndexerService{
		events: []types.IndexedEvent{
			{
				ID:          1,
				BlockNumber: big.NewInt(100),
				TxHash:      "0x1",
				EventName:   "Transfer",
				Contract:    "0xContract1",
				From:        "0xFrom1",
				To:          "0xTo1",
				TokenID:     "1",
				Value:       "100",
				Timestamp:   time.Now(),
			},
		},
	}
	
	server := NewServer(mockIndexerService, "test-secret")
	
	req, err := http.NewRequest("GET", "/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.GetEventsHandler)
	
	handler.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
	
	// Check if the response is valid JSON
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}
	
	// Check if events are in the response
	events, exists := response["events"]
	if !exists {
		t.Error("Expected 'events' field in response")
	}
	
	eventsSlice, ok := events.([]interface{})
	if !ok {
		t.Error("Expected 'events' to be an array")
	}
	
	if len(eventsSlice) != 1 {
		t.Errorf("Expected 1 event, got %d", len(eventsSlice))
	}
}

func TestGetEventHandler(t *testing.T) {
	mockIndexerService := &MockIndexerService{
		events: []types.IndexedEvent{
			{
				ID:          1,
				BlockNumber: big.NewInt(100),
				TxHash:      "0x1",
				EventName:   "Transfer",
				Contract:    "0xContract1",
				From:        "0xFrom1",
				To:          "0xTo1",
				TokenID:     "1",
				Value:       "100",
				Timestamp:   time.Now(),
			},
		},
	}
	
	server := NewServer(mockIndexerService, "test-secret")
	
	req, err := http.NewRequest("GET", "/events/1", nil)
	if err != nil {
		t.Fatal(err)
	}
	
	// Set up the route with a variable
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.GetEventHandler)
	
	handler.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
	
	// Check if the response is valid JSON
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}
	
	// Check if event is in the response
	event, exists := response["event"]
	if !exists {
		t.Error("Expected 'event' field in response")
	}
	
	if event == nil {
		t.Error("Expected event to not be nil")
	}
}

func TestGetEventsByBlockRangeHandler(t *testing.T) {
	mockIndexerService := &MockIndexerService{
		events: []types.IndexedEvent{
			{
				ID:          1,
				BlockNumber: big.NewInt(100),
				TxHash:      "0x1",
				EventName:   "Transfer",
				Contract:    "0xContract1",
				From:        "0xFrom1",
				To:          "0xTo1",
				TokenID:     "1",
				Value:       "100",
				Timestamp:   time.Now(),
			},
		},
	}
	
	server := NewServer(mockIndexerService, "test-secret")
	
	// Create request with query parameters
	req, err := http.NewRequest("GET", "/events/block-range?from=100&to=200", nil)
	if err != nil {
		t.Fatal(err)
	}
	
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.GetEventsByBlockRangeHandler)
	
	handler.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
	
	// Check if the response is valid JSON
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}
	
	// Check if events are in the response
	events, exists := response["events"]
	if !exists {
		t.Error("Expected 'events' field in response")
	}
	
	eventsSlice, ok := events.([]interface{})
	if !ok {
		t.Error("Expected 'events' to be an array")
	}
	
	if len(eventsSlice) != 1 {
		t.Errorf("Expected 1 event, got %d", len(eventsSlice))
	}
}

func TestGetLastProcessedBlockHandler(t *testing.T) {
	mockIndexerService := &MockIndexerService{}
	
	server := NewServer(mockIndexerService, "test-secret")
	
	req, err := http.NewRequest("GET", "/status/last-block", nil)
	if err != nil {
		t.Fatal(err)
	}
	
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.GetLastProcessedBlockHandler)
	
	handler.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
	
	// Check if the response is valid JSON
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}
	
	// Check if block_number is in the response
	blockNum, exists := response["block_number"]
	if !exists {
		t.Error("Expected 'block_number' field in response")
	}
	
	if blockNum == nil {
		t.Error("Expected block_number to not be nil")
	}
}

func TestReplayEventsHandler(t *testing.T) {
	mockIndexerService := &MockIndexerService{}
	
	server := NewServer(mockIndexerService, "test-secret")
	
	// Create request with JSON body
	payload := map[string]string{
		"from_block": "100",
		"to_block":   "200",
	}
	jsonPayload, _ := json.Marshal(payload)
	
	req, err := http.NewRequest("POST", "/events/replay", bytes.NewBuffer(jsonPayload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.ReplayEventsHandler)
	
	handler.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
	
	// Check if the response is valid JSON
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Expected valid JSON response, got error: %v", err)
	}
	
	// Check if success is in the response
	success, exists := response["success"]
	if !exists {
		t.Error("Expected 'success' field in response")
	}
	
	successBool, ok := success.(bool)
	if !ok {
		t.Error("Expected 'success' to be a boolean")
	}
	
	if !successBool {
		t.Error("Expected success to be true")
	}
}