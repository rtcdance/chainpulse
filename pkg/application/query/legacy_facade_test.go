package query

import (
	"context"
	"testing"

	"chainpulse/pkg/core"
	domainquery "chainpulse/pkg/domain/query"
	legacyquery "chainpulse/pkg/services/query"
)

// mockLegacyQueryService implements legacyquery.QueryService for testing
type mockLegacyQueryService struct {
	queryResult      *legacyquery.QueryResult
	queryError       error
	queryByHashEvent *core.BlockchainEvent
	queryByHashError error
	invalidateError  error
	healthStatus     *core.HealthStatus
}

func (m *mockLegacyQueryService) Query(ctx context.Context, req *legacyquery.QueryRequest) (*legacyquery.QueryResult, error) {
	return m.queryResult, m.queryError
}

func (m *mockLegacyQueryService) QueryByHash(ctx context.Context, hash string) (*core.BlockchainEvent, error) {
	return m.queryByHashEvent, m.queryByHashError
}

func (m *mockLegacyQueryService) InvalidateCache(ctx context.Context, key string) error {
	return m.invalidateError
}

func (m *mockLegacyQueryService) Health(ctx context.Context) *core.HealthStatus {
	return m.healthStatus
}

func TestLegacyFacade_Query_NilRequest(t *testing.T) {
	legacy := &mockLegacyQueryService{
		queryResult: &legacyquery.QueryResult{
			Events:       []core.BlockchainEvent{{ID: "evt-1"}},
			Total:        1,
			CacheHit:     true,
			ResponseTime: 10,
			Source:       "cache",
		},
	}

	facade := NewLegacyFacade(legacy)
	result, err := facade.Query(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Total)
	}
	if !result.CacheHit {
		t.Error("expected cache hit")
	}
	if result.Source != "cache" {
		t.Errorf("expected source 'cache', got %s", result.Source)
	}
}

func TestLegacyFacade_Query_WithRequest(t *testing.T) {
	legacy := &mockLegacyQueryService{
		queryResult: &legacyquery.QueryResult{
			Events:       []core.BlockchainEvent{{ID: "evt-2"}},
			Total:        1,
			CacheHit:     false,
			ResponseTime: 50,
			Source:       "database",
		},
	}

	facade := NewLegacyFacade(legacy)
	req := &domainquery.Request{
		QueryType:  "by_block_range",
		Collection: "events",
		Limit:      10,
		Offset:     0,
	}
	result, err := facade.Query(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Total)
	}
	if result.Source != "database" {
		t.Errorf("expected source 'database', got %s", result.Source)
	}
}

func TestLegacyFacade_Query_LegacyError(t *testing.T) {
	legacy := &mockLegacyQueryService{
		queryError: core.NewSystemError(core.ErrorTypePermanent, core.ErrorCodeDatabaseError, "db down", nil),
	}

	facade := NewLegacyFacade(legacy)
	_, err := facade.Query(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error from legacy service")
	}
}

func TestLegacyFacade_QueryByHash(t *testing.T) {
	expectedEvent := &core.BlockchainEvent{ID: "evt-hash"}
	legacy := &mockLegacyQueryService{
		queryByHashEvent: expectedEvent,
	}

	facade := NewLegacyFacade(legacy)
	event, err := facade.QueryByHash(context.Background(), "0xabc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.ID != "evt-hash" {
		t.Errorf("expected event ID 'evt-hash', got %s", event.ID)
	}
}

func TestLegacyFacade_InvalidateCache(t *testing.T) {
	legacy := &mockLegacyQueryService{}

	facade := NewLegacyFacade(legacy)
	if err := facade.InvalidateCache(context.Background(), "test-key"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLegacyFacade_Health(t *testing.T) {
	legacy := &mockLegacyQueryService{
		healthStatus: &core.HealthStatus{Status: "healthy"},
	}

	facade := NewLegacyFacade(legacy)
	status := facade.Health(context.Background())
	if status.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %s", status.Status)
	}
}

func TestLegacyFacade_ImplementsDomainService(t *testing.T) {
	// Compile-time check — this will fail to compile if LegacyFacade
	// does not implement domainquery.Service
	var _ domainquery.Service = (*LegacyFacade)(nil)
}
