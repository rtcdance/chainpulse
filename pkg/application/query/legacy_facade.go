package query

import (
	"context"

	"chainpulse/pkg/core"
	domainquery "chainpulse/pkg/domain/query"
	legacyquery "chainpulse/pkg/services/query"
)

// LegacyFacade adapts the legacy query service to the new domain contract.
type LegacyFacade struct {
	legacy legacyquery.QueryService
}

var _ domainquery.Service = (*LegacyFacade)(nil)

// NewLegacyFacade creates a domain-compatible facade from the legacy service.
func NewLegacyFacade(legacy legacyquery.QueryService) *LegacyFacade {
	return &LegacyFacade{legacy: legacy}
}

// Query routes domain request to legacy service with explicit field mapping.
func (f *LegacyFacade) Query(ctx context.Context, req *domainquery.Request) (*domainquery.Result, error) {
	if req == nil {
		legacyResult, err := f.legacy.Query(ctx, nil)
		if err != nil || legacyResult == nil {
			return nil, err
		}
		return &domainquery.Result{
			Events:       legacyResult.Events,
			Total:        legacyResult.Total,
			CacheHit:     legacyResult.CacheHit,
			ResponseTime: legacyResult.ResponseTime,
			Source:       legacyResult.Source,
		}, nil
	}

	legacyReq := &legacyquery.QueryRequest{
		QueryType:  req.QueryType,
		Collection: req.Collection,
		Filter:     req.Filter,
		Limit:      req.Limit,
		Offset:     req.Offset,
		CacheKey:   req.CacheKey,
		CacheTTL:   req.CacheTTL,
		Sort:       req.Sort,
	}

	legacyResult, err := f.legacy.Query(ctx, legacyReq)
	if err != nil || legacyResult == nil {
		return nil, err
	}

	return &domainquery.Result{
		Events:       legacyResult.Events,
		Total:        legacyResult.Total,
		CacheHit:     legacyResult.CacheHit,
		ResponseTime: legacyResult.ResponseTime,
		Source:       legacyResult.Source,
	}, nil
}

// QueryByHash delegates directly to legacy service.
func (f *LegacyFacade) QueryByHash(ctx context.Context, hash string) (*core.BlockchainEvent, error) {
	return f.legacy.QueryByHash(ctx, hash)
}

// InvalidateCache delegates directly to legacy service.
func (f *LegacyFacade) InvalidateCache(ctx context.Context, key string) error {
	return f.legacy.InvalidateCache(ctx, key)
}

// Health delegates directly to legacy service.
func (f *LegacyFacade) Health(ctx context.Context) *core.HealthStatus {
	return f.legacy.Health(ctx)
}
