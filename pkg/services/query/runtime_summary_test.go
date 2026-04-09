package query

import (
	"context"
	"strings"
	"testing"

	"chainpulse/pkg/core"
)

func TestQueryServiceRuntimeSummaryReady(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	cacheService := NewCacheService(logger, metrics)

	service := NewQueryService(
		&mockDatabaseManager{},
		&MockMongoDBAdapter{healthy: true},
		&MockPostgreSQLAdapter{healthy: true},
		cacheService,
		logger,
		metrics,
	)
	if err := service.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize query service: %v", err)
	}
	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("start query service: %v", err)
	}

	summary := service.RuntimeSummary(context.Background())
	if summary.QueryPosture != "query-runtime-ready" {
		t.Fatalf("expected query-runtime-ready, got %q", summary.QueryPosture)
	}
	if summary.CachePosture != "cache-ready" {
		t.Fatalf("expected cache-ready, got %q", summary.CachePosture)
	}
	if summary.CircuitBreakerPosture != "circuit-not-wired" {
		t.Fatalf("expected circuit-not-wired, got %q", summary.CircuitBreakerPosture)
	}
	if summary.ConsistencyPosture != "consistency-not-wired" {
		t.Fatalf("expected consistency-not-wired, got %q", summary.ConsistencyPosture)
	}
}

func TestQueryServiceRuntimeSummaryHealthyCacheAfterStart(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	cacheService := NewCacheService(logger, metrics)

	service := NewQueryService(
		&mockDatabaseManager{},
		&MockMongoDBAdapter{healthy: true},
		&MockPostgreSQLAdapter{healthy: true},
		cacheService,
		logger,
		metrics,
	)
	if err := service.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize query service: %v", err)
	}
	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("start query service: %v", err)
	}

	summary := service.RuntimeSummary(context.Background())
	if summary.Status != "healthy" {
		t.Fatalf("expected healthy status, got %q", summary.Status)
	}
	if summary.CachePosture != "cache-ready" {
		t.Fatalf("expected cache-ready, got %q", summary.CachePosture)
	}
}

func TestQueryServiceRuntimeSummaryDegradedWhenPostgresUnhealthy(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	cacheService := NewCacheService(logger, metrics)

	service := NewQueryService(
		&mockDatabaseManager{},
		&MockMongoDBAdapter{healthy: true},
		&MockPostgreSQLAdapter{healthy: false},
		cacheService,
		logger,
		metrics,
	)
	if err := service.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize query service: %v", err)
	}
	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("start query service: %v", err)
	}

	health := service.Health(context.Background())
	if health.Status != "degraded" {
		t.Fatalf("expected degraded status, got %q", health.Status)
	}
	if !strings.Contains(health.Message, "PostgreSQL") {
		t.Fatalf("expected PostgreSQL in health message, got %q", health.Message)
	}

	summary := service.RuntimeSummary(context.Background())
	if summary.Status != "degraded" {
		t.Fatalf("expected degraded runtime summary status, got %q", summary.Status)
	}
	if summary.QueryPosture != "query-runtime-degraded" {
		t.Fatalf("expected query-runtime-degraded, got %q", summary.QueryPosture)
	}
}
