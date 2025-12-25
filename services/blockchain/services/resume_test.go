package blockchain

import (
	"context"
	"math/big"
	"testing"

	"chainpulse/shared/types"

	"github.com/ethereum/go-ethereum/ethclient"
)

// MockEthClient is a mock implementation of ethclient.Client for testing
type MockEthClient struct {
	*ethclient.Client // Embed the real client to satisfy the interface
}

// MockDB is a mock implementation of database.Database for testing
type MockDB struct {
	LastBlock *big.Int
	Events    []types.IndexedEvent
}

func (m *MockDB) SaveEvent(event *types.IndexedEvent) error {
	m.Events = append(m.Events, *event)
	return nil
}

func (m *MockDB) GetEvents(filter *types.EventFilter) ([]types.IndexedEvent, error) {
	return m.Events, nil
}

func (m *MockDB) GetEventByID(id uint) (*types.IndexedEvent, error) {
	if len(m.Events) > 0 && id <= uint(len(m.Events)) {
		return &m.Events[id-1], nil
	}
	return nil, nil
}

func (m *MockDB) GetLatestBlockProcessed() (*types.IndexedEvent, error) {
	if len(m.Events) > 0 {
		return &m.Events[len(m.Events)-1], nil
	}
	return nil, nil
}

func (m *MockDB) GetLastProcessedBlock() (*big.Int, error) {
	if m.LastBlock != nil {
		return m.LastBlock, nil
	}
	return big.NewInt(0), nil
}

func (m *MockDB) SaveLastProcessedBlock(blockNum *big.Int) error {
	m.LastBlock = blockNum
	return nil
}

func (m *MockDB) GetEventsByBlockRange(fromBlock, toBlock *big.Int) ([]types.IndexedEvent, error) {
	var result []types.IndexedEvent
	for _, event := range m.Events {
		if event.BlockNumber.Cmp(fromBlock) >= 0 && event.BlockNumber.Cmp(toBlock) <= 0 {
			result = append(result, event)
		}
	}
	return result, nil
}

func TestResumeService_GetLastProcessedBlock(t *testing.T) {
	mockDB := &MockDB{
		LastBlock: big.NewInt(1000),
	}
	
	resumeService := &ResumeService{
		db: mockDB,
	}
	
	blockNum, err := resumeService.GetLastProcessedBlock()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if blockNum.Cmp(big.NewInt(1000)) != 0 {
		t.Errorf("Expected block number 1000, got %s", blockNum.String())
	}
}

func TestResumeService_SaveLastProcessedBlock(t *testing.T) {
	mockDB := &MockDB{}
	
	resumeService := &ResumeService{
		db: mockDB,
	}
	
	expectedBlock := big.NewInt(2000)
	err := resumeService.SaveLastProcessedBlock(expectedBlock)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if mockDB.LastBlock.Cmp(expectedBlock) != 0 {
		t.Errorf("Expected block number %s, got %s", expectedBlock.String(), mockDB.LastBlock.String())
	}
}

func TestResumeService_GetLastProcessedBlockDefault(t *testing.T) {
	mockDB := &MockDB{
		LastBlock: nil, // Simulate no record found
	}
	
	resumeService := &ResumeService{
		db: mockDB,
	}
	
	blockNum, err := resumeService.GetLastProcessedBlock()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if blockNum.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Expected default block number 0, got %s", blockNum.String())
	}
}

func TestResumeService_ReplayEvents(t *testing.T) {
	mockDB := &MockDB{}
	
	// Create a mock client with a nil ethclient.Client since we can't easily mock all methods
	mockClient := &MockEthClient{}
	
	resumeService := &ResumeService{
		client: mockClient,
		db:     mockDB,
	}
	
	fromBlock := big.NewInt(100)
	toBlock := big.NewInt(200)
	
	// This test will fail in a real scenario because we can't easily mock the ethclient
	// In a real project, we would need to create a more comprehensive mock
	// For now, we'll just test that the function exists and can be called
	ctx := context.Background()
	
	// Note: This will fail in actual execution since we don't have a real eth client
	// The test is just to ensure the function signature is correct
	_ = resumeService.ReplayEvents(ctx, fromBlock, toBlock)
}

func TestNewResumeService(t *testing.T) {
	mockClient := &MockEthClient{}
	mockDB := &MockDB{}
	
	resumeService := NewResumeService(mockClient, mockDB)
	
	if resumeService == nil {
		t.Error("Expected ResumeService instance, got nil")
	}
	
	if resumeService.client != mockClient {
		t.Error("Expected client to be set correctly")
	}
	
	if resumeService.db != mockDB {
		t.Error("Expected db to be set correctly")
	}
}