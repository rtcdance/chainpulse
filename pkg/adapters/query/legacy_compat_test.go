package query

import (
	"context"
	"testing"

	"chainpulse/pkg/core"
	domainquery "chainpulse/pkg/domain/query"
	legacyquery "chainpulse/pkg/services/query"
)

func TestNewDomainServiceFromLegacy_ReturnsDomainService(t *testing.T) {
	legacy := &facadeTestLegacyService{
		queryResult: &legacyquery.QueryResult{
			Events: []core.BlockchainEvent{{ID: "test"}},
			Total:  1,
		},
	}

	service := NewDomainServiceFromLegacy(legacy)

	// Verify it implements the domain Service interface
	var _ domainquery.Service = service

	result, err := service.Query(context.Background(), &domainquery.Request{
		QueryType: "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Total)
	}
}

// facadeTestLegacyService is a minimal mock for adapter tests
type facadeTestLegacyService struct {
	queryResult *legacyquery.QueryResult
	queryError  error
}

func (m *facadeTestLegacyService) Query(ctx context.Context, req *legacyquery.QueryRequest) (*legacyquery.QueryResult, error) {
	return m.queryResult, m.queryError
}

func (m *facadeTestLegacyService) QueryByHash(ctx context.Context, hash string) (*core.BlockchainEvent, error) {
	return nil, nil
}

func (m *facadeTestLegacyService) InvalidateCache(ctx context.Context, key string) error {
	return nil
}

func (m *facadeTestLegacyService) Health(ctx context.Context) *core.HealthStatus {
	return &core.HealthStatus{Status: "healthy"}
}
