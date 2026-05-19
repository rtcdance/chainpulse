package query

import (
	"context"
	"errors"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// mockEventStore is a mock implementation of EventStore for testing
type mockEventStore struct {
	shouldFail bool
}

func (m *mockEventStore) Initialize(ctx context.Context) error {
	if m.shouldFail {
		return errors.New("initialize failed")
	}
	return nil
}

func (m *mockEventStore) InsertEvent(ctx context.Context, event *core.BlockchainEvent) error {
	if m.shouldFail {
		return errors.New("insert failed")
	}
	return nil
}

func (m *mockEventStore) InsertEventBatch(ctx context.Context, events []*core.BlockchainEvent) error {
	if m.shouldFail {
		return errors.New("batch insert failed")
	}
	return nil
}

func (m *mockEventStore) GetEvent(ctx context.Context, eventID string) (*core.BlockchainEvent, error) {
	if m.shouldFail {
		return nil, errors.New("get failed")
	}
	return nil, nil
}

func (m *mockEventStore) GetEventsByChain(ctx context.Context, chainID int, limit int, offset int) ([]*core.BlockchainEvent, error) {
	if m.shouldFail {
		return nil, errors.New("get by chain failed")
	}
	return nil, nil
}

func (m *mockEventStore) GetEventsByContract(ctx context.Context, contractAddress string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	if m.shouldFail {
		return nil, errors.New("get by contract failed")
	}
	return nil, nil
}

func (m *mockEventStore) GetEventsByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	if m.shouldFail {
		return nil, errors.New("get by event name failed")
	}
	return nil, nil
}

func (m *mockEventStore) GetEventsByBlock(ctx context.Context, blockNumber int64) ([]*core.BlockchainEvent, error) {
	if m.shouldFail {
		return nil, errors.New("get by block failed")
	}
	return nil, nil
}

func (m *mockEventStore) GetEventsByAddress(ctx context.Context, address string, limit int) ([]*core.BlockchainEvent, error) {
	if m.shouldFail {
		return nil, errors.New("get by address failed")
	}
	return nil, nil
}

func (m *mockEventStore) GetEventsByName(ctx context.Context, eventName string, limit int) ([]*core.BlockchainEvent, error) {
	if m.shouldFail {
		return nil, errors.New("get by name failed")
	}
	return nil, nil
}

func (m *mockEventStore) GetEventsPaginated(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error) {
	if m.shouldFail {
		return nil, false, errors.New("get paginated failed")
	}
	return nil, false, nil
}

func (m *mockEventStore) CountEvents(ctx context.Context) (int64, error) {
	if m.shouldFail {
		return 0, errors.New("count events failed")
	}
	return 0, nil
}

func (m *mockEventStore) DeleteExpiredEvents(ctx context.Context) (int64, error) {
	if m.shouldFail {
		return 0, errors.New("delete expired failed")
	}
	return 0, nil
}

func (m *mockEventStore) Health(ctx context.Context) *core.HealthStatus {
	return &core.HealthStatus{
		Status: "healthy",
	}
}

func (m *mockEventStore) Close(ctx context.Context) error {
	return nil
}

func (m *mockEventStore) GetEventsByCorrelationID(ctx context.Context, correlationID string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

// mockMetadataStore is a mock implementation of EventMetadataStore for testing
type mockMetadataStore struct {
	shouldFail bool
}

func (m *mockMetadataStore) Initialize(ctx context.Context) error {
	if m.shouldFail {
		return errors.New("initialize failed")
	}
	return nil
}

func (m *mockMetadataStore) InsertMetadata(ctx context.Context, metadata *EventMetadata) error {
	if m.shouldFail {
		return errors.New("insert failed")
	}
	return nil
}

func (m *mockMetadataStore) InsertMetadataBatch(ctx context.Context, metadataList []*EventMetadata) error {
	if m.shouldFail {
		return errors.New("batch insert failed")
	}
	return nil
}

func (m *mockMetadataStore) GetMetadata(ctx context.Context, eventID string) (*EventMetadata, error) {
	if m.shouldFail {
		return nil, errors.New("get failed")
	}
	return nil, nil
}

func (m *mockMetadataStore) GetMetadataByChain(ctx context.Context, chainID int, limit int, offset int) ([]*EventMetadata, error) {
	if m.shouldFail {
		return nil, errors.New("get by chain failed")
	}
	return nil, nil
}

func (m *mockMetadataStore) GetMetadataBatch(ctx context.Context, eventIDs []string) (map[string]*EventMetadata, error) {
	if m.shouldFail {
		return nil, errors.New("get batch failed")
	}
	return nil, nil
}

func (m *mockMetadataStore) UpdateMetadata(ctx context.Context, metadata *EventMetadata) error {
	if m.shouldFail {
		return errors.New("update failed")
	}
	return nil
}

func (m *mockMetadataStore) Health(ctx context.Context) *core.HealthStatus {
	return &core.HealthStatus{
		Status: "healthy",
	}
}

func (m *mockMetadataStore) Close(ctx context.Context) error {
	return nil
}

// TestConsistencyCheckerInitialize tests initialization
func TestConsistencyCheckerInitialize(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	eventStore := &mockEventStore{}
	metadataStore := &mockMetadataStore{}

	checker := NewConsistencyChecker(eventStore, metadataStore, logger, metrics)

	if checker.initialized {
		t.Error("Checker should not be initialized before Initialize() call")
	}

	ctx := context.Background()
	err := checker.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize should succeed: %v", err)
	}

	if !checker.initialized {
		t.Error("Checker should be initialized after Initialize() call")
	}
}

// TestConsistencyCheckerInitializeWithNilStores tests initialization with nil stores
func TestConsistencyCheckerInitializeWithNilStores(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	// Test with nil event store
	checker := NewConsistencyChecker(nil, &mockMetadataStore{}, logger, metrics)
	ctx := context.Background()
	err := checker.Initialize(ctx)
	if err == nil {
		t.Error("Initialize should fail with nil event store")
	}

	// Test with nil metadata store
	checker = NewConsistencyChecker(&mockEventStore{}, nil, logger, metrics)
	err = checker.Initialize(ctx)
	if err == nil {
		t.Error("Initialize should fail with nil metadata store")
	}
}

// TestConsistencyCheckerCheckConsistency tests consistency check
func TestConsistencyCheckerCheckConsistency(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	eventStore := &mockEventStore{}
	metadataStore := &mockMetadataStore{}

	checker := NewConsistencyChecker(eventStore, metadataStore, logger, metrics)
	ctx := context.Background()

	// Should fail because checker is not initialized
	_, err := checker.CheckConsistency(ctx)
	if err == nil {
		t.Error("CheckConsistency should fail when not initialized")
	}

	// Initialize checker
	if err := checker.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize checker: %v", err)
	}

	// Should succeed with empty stores
	result, err := checker.CheckConsistency(ctx)
	if err != nil {
		t.Errorf("CheckConsistency should succeed: %v", err)
	}

	if result == nil {
		t.Fatalf("Result should not be nil")
	}

	if !result.IsConsistent {
		t.Error("Result should be consistent for empty stores")
	}
}

// TestConsistencyCheckerHealth tests health check
func TestConsistencyCheckerHealth(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	eventStore := &mockEventStore{}
	metadataStore := &mockMetadataStore{}

	checker := NewConsistencyChecker(eventStore, metadataStore, logger, metrics)
	ctx := context.Background()

	// Should be unhealthy when not initialized
	health := checker.Health(ctx)
	if health.Status != "unhealthy" {
		t.Errorf("Health should be unhealthy when not initialized, got %s", health.Status)
	}

	// Initialize checker
	if err := checker.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize checker: %v", err)
	}

	// Should be healthy when initialized
	health = checker.Health(ctx)
	if health.Status != "healthy" {
		t.Errorf("Health should be healthy when initialized, got %s", health.Status)
	}
}

// TestConsistencyCheckerClose tests closure
func TestConsistencyCheckerClose(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	eventStore := &mockEventStore{}
	metadataStore := &mockMetadataStore{}

	checker := NewConsistencyChecker(eventStore, metadataStore, logger, metrics)
	ctx := context.Background()

	// Initialize checker
	if err := checker.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize checker: %v", err)
	}

	if !checker.initialized {
		t.Error("Checker should be initialized")
	}

	// Close checker
	err := checker.Close(ctx)
	if err != nil {
		t.Errorf("Close should succeed: %v", err)
	}

	if checker.initialized {
		t.Error("Checker should be uninitialized after Close()")
	}
}

// TestConsistencyCheckResultIsConsistent tests result consistency flag
func TestConsistencyCheckResultIsConsistent(t *testing.T) {
	t.Parallel()
	result := &ConsistencyCheckResult{
		EventCount:        10,
		MetadataCount:     10,
		OrphanedMetadata:  0,
		MissingMetadata:   0,
		CorruptedEvents:   0,
		CorruptedMetadata: 0,
		Issues:            []string{},
	}

	result.IsConsistent = len(result.Issues) == 0 && result.OrphanedMetadata == 0 && result.MissingMetadata == 0 && result.CorruptedEvents == 0

	if !result.IsConsistent {
		t.Error("Result should be consistent when all checks pass")
	}

	// Add an issue
	result.Issues = append(result.Issues, "Test issue")
	result.IsConsistent = len(result.Issues) == 0 && result.OrphanedMetadata == 0 && result.MissingMetadata == 0 && result.CorruptedEvents == 0

	if result.IsConsistent {
		t.Error("Result should not be consistent when issues exist")
	}
}

// TestConsistencyCheckResultOrphanedMetadata tests orphaned metadata detection
func TestConsistencyCheckResultOrphanedMetadata(t *testing.T) {
	t.Parallel()
	result := &ConsistencyCheckResult{
		EventCount:        10,
		MetadataCount:     15,
		OrphanedMetadata:  5,
		MissingMetadata:   0,
		CorruptedEvents:   0,
		CorruptedMetadata: 0,
		Issues:            []string{},
	}

	result.IsConsistent = len(result.Issues) == 0 && result.OrphanedMetadata == 0 && result.MissingMetadata == 0 && result.CorruptedEvents == 0

	if result.IsConsistent {
		t.Error("Result should not be consistent when orphaned metadata exists")
	}

	if result.OrphanedMetadata != 5 {
		t.Errorf("Expected 5 orphaned metadata, got %d", result.OrphanedMetadata)
	}
}

// TestConsistencyCheckResultMissingMetadata tests missing metadata detection
func TestConsistencyCheckResultMissingMetadata(t *testing.T) {
	t.Parallel()
	result := &ConsistencyCheckResult{
		EventCount:        15,
		MetadataCount:     10,
		OrphanedMetadata:  0,
		MissingMetadata:   5,
		CorruptedEvents:   0,
		CorruptedMetadata: 0,
		Issues:            []string{},
	}

	result.IsConsistent = len(result.Issues) == 0 && result.OrphanedMetadata == 0 && result.MissingMetadata == 0 && result.CorruptedEvents == 0

	if result.IsConsistent {
		t.Error("Result should not be consistent when missing metadata exists")
	}

	if result.MissingMetadata != 5 {
		t.Errorf("Expected 5 missing metadata, got %d", result.MissingMetadata)
	}
}

// TestConsistencyCheckResultCorruptedEvents tests corrupted events detection
func TestConsistencyCheckResultCorruptedEvents(t *testing.T) {
	t.Parallel()
	result := &ConsistencyCheckResult{
		EventCount:        10,
		MetadataCount:     10,
		OrphanedMetadata:  0,
		MissingMetadata:   0,
		CorruptedEvents:   2,
		CorruptedMetadata: 0,
		Issues:            []string{},
	}

	result.IsConsistent = len(result.Issues) == 0 && result.OrphanedMetadata == 0 && result.MissingMetadata == 0 && result.CorruptedEvents == 0

	if result.IsConsistent {
		t.Error("Result should not be consistent when corrupted events exist")
	}

	if result.CorruptedEvents != 2 {
		t.Errorf("Expected 2 corrupted events, got %d", result.CorruptedEvents)
	}
}

// TestConsistencyCheckResultIssues tests issues tracking
func TestConsistencyCheckResultIssues(t *testing.T) {
	t.Parallel()
	result := &ConsistencyCheckResult{
		EventCount:        10,
		MetadataCount:     10,
		OrphanedMetadata:  0,
		MissingMetadata:   0,
		CorruptedEvents:   0,
		CorruptedMetadata: 0,
		Issues:            []string{},
	}

	if len(result.Issues) != 0 {
		t.Errorf("Expected 0 issues, got %d", len(result.Issues))
	}

	result.Issues = append(result.Issues, "Issue 1")
	result.Issues = append(result.Issues, "Issue 2")

	if len(result.Issues) != 2 {
		t.Errorf("Expected 2 issues, got %d", len(result.Issues))
	}
}

// TestConsistencyCheckerMultipleChecks tests multiple consistency checks
func TestConsistencyCheckerMultipleChecks(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	eventStore := &mockEventStore{}
	metadataStore := &mockMetadataStore{}

	checker := NewConsistencyChecker(eventStore, metadataStore, logger, metrics)
	ctx := context.Background()

	_ = checker.Initialize(ctx)

	// Run multiple checks
	for i := 0; i < 3; i++ {
		result, err := checker.CheckConsistency(ctx)
		if err != nil {
			t.Errorf("CheckConsistency should succeed on iteration %d: %v", i, err)
		}

		if result == nil {
			t.Errorf("Result should not be nil on iteration %d", i)
		}
	}
}

// TestConsistencyCheckerWithFailingStores tests with failing stores
func TestConsistencyCheckerWithFailingStores(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	eventStore := &mockEventStore{
		shouldFail: true,
	}
	metadataStore := &mockMetadataStore{}

	checker := NewConsistencyChecker(eventStore, metadataStore, logger, metrics)
	ctx := context.Background()

	_ = checker.Initialize(ctx)

	// Should still return a result even if stores fail
	result, err := checker.CheckConsistency(ctx)
	if err != nil {
		t.Errorf("CheckConsistency should not fail: %v", err)
	}

	if result == nil {
		t.Fatalf("Result should not be nil")
	}

	// Should have issues
	if len(result.Issues) == 0 {
		t.Error("Result should have issues when stores fail")
	}
}
