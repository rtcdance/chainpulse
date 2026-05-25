package qindex

import (
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	t.Parallel()

	m := NewManager()
	if m == nil {
		t.Fatal("expected non-nil Manager")
	}
	if m.indexes == nil {
		t.Error("expected non-nil indexes map")
	}
	if m.indexStats == nil {
		t.Error("expected non-nil indexStats map")
	}
	if m.pendingIndexes == nil {
		t.Error("expected non-nil pendingIndexes map")
	}
	if len(m.indexes) != 0 {
		t.Errorf("expected empty indexes, got %d", len(m.indexes))
	}
	if len(m.indexStats) != 0 {
		t.Errorf("expected empty indexStats, got %d", len(m.indexStats))
	}
	if len(m.pendingIndexes) != 0 {
		t.Errorf("expected empty pendingIndexes, got %d", len(m.pendingIndexes))
	}
}

func TestNewStatsCollector_DefaultWindow(t *testing.T) {
	t.Parallel()

	sc := NewStatsCollector(0)
	if sc == nil {
		t.Fatal("expected non-nil StatsCollector")
	}
	if sc.aggregationWindow != 5*time.Minute {
		t.Errorf("expected default 5m window, got %v", sc.aggregationWindow)
	}
	if sc.stats == nil {
		t.Error("expected non-nil stats map")
	}
	if sc.aggregatedStats == nil {
		t.Error("expected non-nil aggregatedStats")
	}
}

func TestNewStatsCollector_NegativeWindow(t *testing.T) {
	t.Parallel()

	sc := NewStatsCollector(-1 * time.Minute)
	if sc.aggregationWindow != 5*time.Minute {
		t.Errorf("expected default 5m for negative input, got %v", sc.aggregationWindow)
	}
}

func TestNewStatsCollector_CustomWindow(t *testing.T) {
	t.Parallel()

	sc := NewStatsCollector(10 * time.Minute)
	if sc.aggregationWindow != 10*time.Minute {
		t.Errorf("expected 10m, got %v", sc.aggregationWindow)
	}
}

func TestNewOptimizer_DefaultCacheSize(t *testing.T) {
	t.Parallel()

	o := NewOptimizer(0)
	if o == nil {
		t.Fatal("expected non-nil Optimizer")
	}
	if o.maxCacheSize != 1000 {
		t.Errorf("expected default 1000 cache size, got %d", o.maxCacheSize)
	}
	if o.queryStats == nil {
		t.Error("expected non-nil queryStats map")
	}
	if o.indexRecommendations == nil {
		t.Error("expected non-nil indexRecommendations map")
	}
	if o.cache == nil {
		t.Error("expected non-nil cache map")
	}
}

func TestNewOptimizer_NegativeCacheSize(t *testing.T) {
	t.Parallel()

	o := NewOptimizer(-5)
	if o.maxCacheSize != 1000 {
		t.Errorf("expected default 1000 for negative input, got %d", o.maxCacheSize)
	}
}

func TestNewOptimizer_CustomCacheSize(t *testing.T) {
	t.Parallel()

	o := NewOptimizer(500)
	if o.maxCacheSize != 500 {
		t.Errorf("expected 500, got %d", o.maxCacheSize)
	}
}

func TestIndexInfo_Fields(t *testing.T) {
	t.Parallel()

	now := time.Now()
	info := IndexInfo{
		Name:       "idx_test",
		TableName:  "events",
		Columns:    []string{"block_number", "tx_index"},
		Type:       "btree",
		Unique:     true,
		IsValid:    true,
		Properties: map[string]string{"tablespace": "fast_ssd"},
	}

	if info.Name != "idx_test" {
		t.Errorf("expected idx_test, got %s", info.Name)
	}
	if info.IsValid != true {
		t.Error("expected valid index")
	}
	if len(info.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(info.Columns))
	}

	info.LastModified = now
	if !info.LastModified.Equal(now) {
		t.Error("LastModified not set correctly")
	}
}

func TestStatistics_Fields(t *testing.T) {
	t.Parallel()

	stats := Statistics{
		IndexName:          "idx_events",
		UsageCount:         100,
		SelectsUsed:        80,
		EffectivenessScore: 0.95,
	}

	if stats.IndexName != "idx_events" {
		t.Errorf("expected idx_events, got %s", stats.IndexName)
	}
	if stats.UsageCount != 100 {
		t.Errorf("expected 100, got %d", stats.UsageCount)
	}
}

func TestPendingIndex_Fields(t *testing.T) {
	t.Parallel()

	pi := PendingIndex{
		Name:       "idx_pending",
		TableName:  "blocks",
		Columns:    []string{"number"},
		Priority:   1,
		MaxRetries: 3,
	}

	if pi.MaxRetries != 3 {
		t.Errorf("expected 3 max retries, got %d", pi.MaxRetries)
	}
	if pi.RetryCount != 0 {
		t.Errorf("expected 0 retries, got %d", pi.RetryCount)
	}
}

func TestQueryMetrics_Fields(t *testing.T) {
	t.Parallel()

	qm := QueryMetrics{
		QueryHash:      "abc123",
		ExecutionCount: 10,
		SuccessCount:   9,
		ErrorCount:     1,
		CacheHits:      5,
		CacheMisses:    5,
	}

	if qm.QueryHash != "abc123" {
		t.Errorf("expected abc123, got %s", qm.QueryHash)
	}
	if qm.CacheHits+qm.CacheMisses != 10 {
		t.Error("cache hits + misses should equal total")
	}
}

func TestAggregatedMetrics_Fields(t *testing.T) {
	t.Parallel()

	am := AggregatedMetrics{
		TotalQueries: 100,
		CacheHitRate: 0.75,
	}

	if am.TotalQueries != 100 {
		t.Errorf("expected 100, got %d", am.TotalQueries)
	}
	if am.CacheHitRate != 0.75 {
		t.Errorf("expected 0.75, got %f", am.CacheHitRate)
	}
}

func TestIndexRecommendation_Fields(t *testing.T) {
	t.Parallel()

	rec := IndexRecommendation{
		TableName:     "events",
		Columns:       []string{"block_number"},
		Type:          "hash",
		Priority:      1,
		EstimatedGain: 0.8,
	}

	if rec.TableName != "events" {
		t.Errorf("expected events, got %s", rec.TableName)
	}
	if rec.EstimatedGain != 0.8 {
		t.Errorf("expected 0.8, got %f", rec.EstimatedGain)
	}
}

func TestOptimizedQuery_Fields(t *testing.T) {
	t.Parallel()

	oq := OptimizedQuery{
		OriginalQuery:  "SELECT * FROM events",
		OptimizedQuery: "SELECT * FROM events WHERE block_number > $1",
		EstimatedGain:  0.6,
		ExecutionCount: 42,
	}

	if oq.OriginalQuery != "SELECT * FROM events" {
		t.Errorf("unexpected original query: %s", oq.OriginalQuery)
	}
	if oq.ExecutionCount != 42 {
		t.Errorf("expected 42, got %d", oq.ExecutionCount)
	}
}
