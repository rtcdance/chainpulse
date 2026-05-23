package consistency

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// MockLogger implements core.Logger for testing
type MockLogger struct{}

func (ml *MockLogger) Debug(msg string, args ...any) {}
func (ml *MockLogger) Info(msg string, args ...any)  {}
func (ml *MockLogger) Warn(msg string, args ...any)  {}
func (ml *MockLogger) Error(msg string, args ...any) {}
func (ml *MockLogger) Fatal(msg string, args ...any) {}
func (ml *MockLogger) WithCorrelationID(id string) core.Logger {
	return ml
}

// MockDatabasePlugin implements core.DatabasePlugin for testing
type MockDatabasePlugin struct {
	events []*core.BlockchainEvent
	blocks []*core.Block
}

func (mdp *MockDatabasePlugin) StoreEvent(ctx context.Context, event any) error {
	if e, ok := event.(*core.BlockchainEvent); ok {
		mdp.events = append(mdp.events, e)
	}
	return nil
}

func (mdp *MockDatabasePlugin) GetEvent(ctx context.Context, id string) (*core.BlockchainEvent, error) {
	for _, e := range mdp.events {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, nil
}

func (mdp *MockDatabasePlugin) GetAllEvents(ctx context.Context) ([]*core.BlockchainEvent, error) {
	return mdp.events, nil
}

func (mdp *MockDatabasePlugin) GetEventsByBlockRange(ctx context.Context, from, to uint64) ([]*core.BlockchainEvent, error) {
	var result []*core.BlockchainEvent
	for _, e := range mdp.events {
		if e.BlockNumber >= from && e.BlockNumber <= to {
			result = append(result, e)
		}
	}
	return result, nil
}

func (mdp *MockDatabasePlugin) QueryEvents(ctx context.Context, filter any) ([]any, error) {
	return nil, nil
}

func (mdp *MockDatabasePlugin) DeleteEvent(ctx context.Context, id string) error {
	for i, e := range mdp.events {
		if e.ID == id {
			mdp.events = append(mdp.events[:i], mdp.events[i+1:]...)
			return nil
		}
	}
	return nil
}

func (mdp *MockDatabasePlugin) GetBlock(ctx context.Context, number uint64) (*core.Block, error) {
	for _, b := range mdp.blocks {
		if b.Number == number {
			return b, nil
		}
	}
	return nil, nil
}

func (mdp *MockDatabasePlugin) GetAllBlocks(ctx context.Context) ([]*core.Block, error) {
	return mdp.blocks, nil
}

func (mdp *MockDatabasePlugin) StoreBlock(ctx context.Context, block *core.Block) error {
	mdp.blocks = append(mdp.blocks, block)
	return nil
}

func (mdp *MockDatabasePlugin) BatchStoreEvents(ctx context.Context, events []any) error {
	for _, event := range events {
		if e, ok := event.(*core.BlockchainEvent); ok {
			mdp.events = append(mdp.events, e)
		}
	}
	return nil
}

func (mdp *MockDatabasePlugin) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return 0, nil
}

func (mdp *MockDatabasePlugin) MarkEventsAsReorged(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return 0, nil
}

func (mdp *MockDatabasePlugin) GetLatestBlock(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (mdp *MockDatabasePlugin) GetReorgStats(ctx context.Context) (*core.ReorgStats, error) {
	return &core.ReorgStats{}, nil
}

func (mdp *MockDatabasePlugin) Health(ctx context.Context) error {
	_ = ctx
	return nil
}

func (mdp *MockDatabasePlugin) Name() string {
	return "MockDatabasePlugin"
}

func (mdp *MockDatabasePlugin) Version() string {
	return "1.0.0"
}

func (mdp *MockDatabasePlugin) Initialize(ctx context.Context, config core.Config) error {
	_ = ctx
	_ = config
	return nil
}

func (mdp *MockDatabasePlugin) Start(ctx context.Context) error {
	_ = ctx
	return nil
}

func (mdp *MockDatabasePlugin) Stop(ctx context.Context) error {
	_ = ctx
	return nil
}

func (mdp *MockDatabasePlugin) Close() error { return nil }

func TestNewConsistencyChecker(t *testing.T) {
	db := &MockDatabasePlugin{}
	logger := &MockLogger{}

	cc := NewConsistencyChecker(db, logger)

	assert.NotNil(t, cc)
	assert.Equal(t, db, cc.database)
	assert.Equal(t, logger, cc.logger)
}

func TestCheckConsistencyHealthy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping consistency check test in short mode")
	}

	db := &MockDatabasePlugin{}
	logger := &MockLogger{}
	cc := NewConsistencyChecker(db, logger)

	// Add some events
	event1 := &core.BlockchainEvent{
		ID:              "event1",
		BlockNumber:     100,
		LogIndex:        0,
		TransactionHash: common.HexToHash("0x1234"),
		EventSignature:  common.HexToHash("0x5678"),
	}
	event2 := &core.BlockchainEvent{
		ID:              "event2",
		BlockNumber:     101,
		LogIndex:        0,
		TransactionHash: common.HexToHash("0x1235"),
		EventSignature:  common.HexToHash("0x5679"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = db.StoreEvent(ctx, event1)
	_ = db.StoreEvent(ctx, event2)

	report, err := cc.CheckConsistency(ctx)

	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, "healthy", report.Status)
	assert.Equal(t, int64(2), report.TotalEvents)
	assert.Equal(t, int64(0), report.DuplicateEvents)
}

func TestCheckConsistencyWithDuplicates(t *testing.T) {
	db := &MockDatabasePlugin{}
	logger := &MockLogger{}
	cc := NewConsistencyChecker(db, logger)

	// Add duplicate events
	event1 := &core.BlockchainEvent{
		ID:              "event1",
		BlockNumber:     100,
		LogIndex:        0,
		TransactionHash: common.HexToHash("0x1234"),
		EventSignature:  common.HexToHash("0x5678"),
	}
	event2 := &core.BlockchainEvent{
		ID:              "event2",
		BlockNumber:     100,
		LogIndex:        0,
		TransactionHash: common.HexToHash("0x1234"),
		EventSignature:  common.HexToHash("0x5678"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = db.StoreEvent(ctx, event1)
	_ = db.StoreEvent(ctx, event2)

	report, err := cc.CheckConsistency(ctx)

	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, int64(1), report.DuplicateEvents)
	assert.Equal(t, "healthy", report.Status)
}

func TestVerifyEventSequence(t *testing.T) {
	db := &MockDatabasePlugin{}
	logger := &MockLogger{}
	cc := NewConsistencyChecker(db, logger)

	// Add events with gaps
	event1 := &core.BlockchainEvent{
		ID:              "event1",
		BlockNumber:     100,
		LogIndex:        0,
		TransactionHash: common.HexToHash("0x1234"),
		EventSignature:  common.HexToHash("0x5678"),
	}
	event2 := &core.BlockchainEvent{
		ID:              "event2",
		BlockNumber:     102,
		LogIndex:        0,
		TransactionHash: common.HexToHash("0x1235"),
		EventSignature:  common.HexToHash("0x5679"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = db.StoreEvent(ctx, event1)
	_ = db.StoreEvent(ctx, event2)

	report, err := cc.CheckConsistency(ctx)

	require.NoError(t, err)
	assert.Greater(t, len(report.Issues), 0)
}

func TestVerifyBlockSequence(t *testing.T) {
	db := &MockDatabasePlugin{}
	logger := &MockLogger{}
	cc := NewConsistencyChecker(db, logger)

	// Add blocks with gaps
	block1 := &core.Block{
		Number:     100,
		Hash:       common.HexToHash("0x1234"),
		ParentHash: common.HexToHash("0x0000"),
	}
	block2 := &core.Block{
		Number:     102,
		Hash:       common.HexToHash("0x1235"),
		ParentHash: common.HexToHash("0x1234"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = db.StoreBlock(ctx, block1)
	_ = db.StoreBlock(ctx, block2)

	report, err := cc.CheckConsistency(ctx)

	require.NoError(t, err)
	assert.Greater(t, len(report.Issues), 0)
}

func TestRepairInconsistencies(t *testing.T) {
	db := &MockDatabasePlugin{}
	logger := &MockLogger{}
	cc := NewConsistencyChecker(db, logger)

	// Add duplicate events
	event1 := &core.BlockchainEvent{
		ID:              "event1",
		BlockNumber:     100,
		LogIndex:        0,
		TransactionHash: common.HexToHash("0x1234"),
		EventSignature:  common.HexToHash("0x5678"),
	}
	event2 := &core.BlockchainEvent{
		ID:              "event2",
		BlockNumber:     100,
		LogIndex:        0,
		TransactionHash: common.HexToHash("0x1234"),
		EventSignature:  common.HexToHash("0x5678"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = db.StoreEvent(ctx, event1)
	_ = db.StoreEvent(ctx, event2)

	// Store a block so events aren't considered orphaned
	_ = db.StoreBlock(ctx, &core.Block{
		Number: 100,
		Hash:   common.HexToHash("0x1234"),
	})

	report, err := cc.RepairInconsistencies(ctx)

	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, int64(1), report.RepairAttempts)
	assert.Equal(t, int64(1), report.SuccessfulRepairs)
}

func TestGetEventConsistency(t *testing.T) {
	db := &MockDatabasePlugin{}
	logger := &MockLogger{}
	cc := NewConsistencyChecker(db, logger)

	event := &core.BlockchainEvent{
		ID:              "event1",
		BlockNumber:     100,
		LogIndex:        0,
		TransactionHash: common.HexToHash("0x1234"),
		EventSignature:  common.HexToHash("0x5678"),
	}

	block := &core.Block{
		Number:     100,
		Hash:       common.HexToHash("0x1234"),
		ParentHash: common.HexToHash("0x0000"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = db.StoreEvent(ctx, event)
	_ = db.StoreBlock(ctx, block)

	consistency, err := cc.GetEventConsistency(ctx, "event1")

	require.NoError(t, err)
	assert.NotNil(t, consistency)
	assert.Equal(t, "event1", consistency.EventID)
	assert.True(t, consistency.IsValid)
}

func TestGetEventConsistencyNotFound(t *testing.T) {
	db := &MockDatabasePlugin{}
	logger := &MockLogger{}
	cc := NewConsistencyChecker(db, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	consistency, err := cc.GetEventConsistency(ctx, "nonexistent")

	require.NoError(t, err)
	assert.NotNil(t, consistency)
	assert.False(t, consistency.IsValid)
	assert.Greater(t, len(consistency.Issues), 0)
}

func TestGetBlockConsistency(t *testing.T) {
	db := &MockDatabasePlugin{}
	logger := &MockLogger{}
	cc := NewConsistencyChecker(db, logger)

	block := &core.Block{
		Number:     0,
		Hash:       common.HexToHash("0x1234"),
		ParentHash: common.HexToHash("0x0000"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = db.StoreBlock(ctx, block)

	consistency, err := cc.GetBlockConsistency(ctx, 0)

	require.NoError(t, err)
	assert.NotNil(t, consistency)
	assert.Equal(t, uint64(0), consistency.BlockNumber)
	assert.True(t, consistency.IsValid)
}

func TestGetBlockConsistencyNotFound(t *testing.T) {
	db := &MockDatabasePlugin{}
	logger := &MockLogger{}
	cc := NewConsistencyChecker(db, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	consistency, err := cc.GetBlockConsistency(ctx, 999)

	require.NoError(t, err)
	assert.NotNil(t, consistency)
	assert.False(t, consistency.IsValid)
}

func TestGetBlockConsistencyParentMismatch(t *testing.T) {
	db := &MockDatabasePlugin{}
	logger := &MockLogger{}
	cc := NewConsistencyChecker(db, logger)

	block1 := &core.Block{
		Number:     100,
		Hash:       common.HexToHash("0x1234"),
		ParentHash: common.HexToHash("0x0000"),
	}
	block2 := &core.Block{
		Number:     101,
		Hash:       common.HexToHash("0x1235"),
		ParentHash: common.HexToHash("0x9999"), // Wrong parent
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = db.StoreBlock(ctx, block1)
	_ = db.StoreBlock(ctx, block2)

	consistency, err := cc.GetBlockConsistency(ctx, 101)

	require.NoError(t, err)
	assert.NotNil(t, consistency)
	assert.False(t, consistency.HasValidParent)
	assert.False(t, consistency.IsValid)
}

func TestConsistencyReportStructure(t *testing.T) {
	db := &MockDatabasePlugin{}
	logger := &MockLogger{}
	cc := NewConsistencyChecker(db, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	report, err := cc.CheckConsistency(ctx)

	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.NotNil(t, report.CheckedAt)
	assert.NotNil(t, report.Issues)
	assert.GreaterOrEqual(t, report.TotalEvents, int64(0))
	assert.GreaterOrEqual(t, report.DuplicateEvents, int64(0))
}

func TestEventConsistencyStructure(t *testing.T) {
	db := &MockDatabasePlugin{}
	logger := &MockLogger{}
	cc := NewConsistencyChecker(db, logger)

	event := &core.BlockchainEvent{
		ID:              "event1",
		BlockNumber:     100,
		LogIndex:        0,
		TransactionHash: common.HexToHash("0x1234"),
		EventSignature:  common.HexToHash("0x5678"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = db.StoreEvent(ctx, event)

	consistency, err := cc.GetEventConsistency(ctx, "event1")

	require.NoError(t, err)
	assert.NotNil(t, consistency)
	assert.NotEmpty(t, consistency.EventID)
	assert.NotNil(t, consistency.Issues)
}

func TestBlockConsistencyStructure(t *testing.T) {
	db := &MockDatabasePlugin{}
	logger := &MockLogger{}
	cc := NewConsistencyChecker(db, logger)

	block := &core.Block{
		Number:     100,
		Hash:       common.HexToHash("0x1234"),
		ParentHash: common.HexToHash("0x0000"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = db.StoreBlock(ctx, block)

	consistency, err := cc.GetBlockConsistency(ctx, 100)

	require.NoError(t, err)
	assert.NotNil(t, consistency)
	assert.Equal(t, uint64(100), consistency.BlockNumber)
	assert.NotNil(t, consistency.Issues)
}
