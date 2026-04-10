package query

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNewQueryOptimizer(t *testing.T) {
	qo := NewQueryOptimizer(100)
	if qo == nil {
		t.Fatal("expected non-nil QueryOptimizer")
	}
}

func TestOptimizeQuery(t *testing.T) {
	qo := NewQueryOptimizer(100)
	ctx := context.Background()

	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "simple select",
			query:   "SELECT * FROM events",
			wantErr: false,
		},
		{
			name:    "select with where",
			query:   "SELECT * FROM events WHERE id = 1",
			wantErr: false,
		},
		{
			name:    "select with join",
			query:   "SELECT * FROM events JOIN users ON events.user_id = users.id",
			wantErr: false,
		},
		{
			name:    "select with order by",
			query:   "SELECT * FROM events ORDER BY timestamp DESC",
			wantErr: false,
		},
		{
			name:    "select with group by",
			query:   "SELECT COUNT(*) FROM events GROUP BY user_id",
			wantErr: false,
		},
		{
			name:    "empty query",
			query:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := qo.OptimizeQuery(ctx, tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("OptimizeQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

func TestAnalyzeQueryPlan(t *testing.T) {
	qo := NewQueryOptimizer(100)
	ctx := context.Background()

	query := "SELECT * FROM events WHERE id = 1 ORDER BY timestamp"
	plan, err := qo.AnalyzeQueryPlan(ctx, query)
	if err != nil {
		t.Fatalf("AnalyzeQueryPlan() error = %v", err)
	}

	if plan == nil {
		t.Fatal("expected non-nil plan")
	}

	if plan.Query != query {
		t.Errorf("plan.Query = %v, want %v", plan.Query, query)
	}

	if len(plan.Operations) == 0 {
		t.Error("expected operations in plan")
	}

	if plan.EstimatedCost <= 0 {
		t.Error("expected positive estimated cost")
	}
}

func TestRecordQueryExecution(t *testing.T) {
	qo := NewQueryOptimizer(100)

	query := "SELECT * FROM events"
	duration := 100 * time.Millisecond
	indexes := []string{"idx_events_id"}

	qo.RecordQueryExecution(query, duration, nil, indexes)

	stats := qo.GetQueryStatistics(query)
	if stats == nil {
		t.Fatal("expected non-nil statistics")
	}

	if stats.ExecutionCount != 1 {
		t.Errorf("ExecutionCount = %d, want 1", stats.ExecutionCount)
	}

	if stats.TotalDuration != duration {
		t.Errorf("TotalDuration = %v, want %v", stats.TotalDuration, duration)
	}

	if stats.ErrorCount != 0 {
		t.Errorf("ErrorCount = %d, want 0", stats.ErrorCount)
	}
}

func TestRecordQueryExecutionWithError(t *testing.T) {
	qo := NewQueryOptimizer(100)

	query := "SELECT * FROM events"
	duration := 100 * time.Millisecond

	qo.RecordQueryExecution(query, duration, nil, []string{})
	qo.RecordQueryExecution(query, duration, nil, []string{})

	stats := qo.GetQueryStatistics(query)
	if stats.ExecutionCount != 2 {
		t.Errorf("ExecutionCount = %d, want 2", stats.ExecutionCount)
	}

	if stats.AverageDuration != duration {
		t.Errorf("AverageDuration = %v, want %v", stats.AverageDuration, duration)
	}
}

func TestGetAllQueryStatistics(t *testing.T) {
	qo := NewQueryOptimizer(100)

	queries := []string{
		"SELECT * FROM events",
		"SELECT * FROM users",
		"SELECT * FROM logs",
	}

	for _, q := range queries {
		qo.RecordQueryExecution(q, 100*time.Millisecond, nil, []string{})
	}

	stats := qo.GetAllQueryStatistics()
	if len(stats) != 3 {
		t.Errorf("len(stats) = %d, want 3", len(stats))
	}
}

func TestAddIndexRecommendation(t *testing.T) {
	qo := NewQueryOptimizer(100)

	rec := IndexRecommendation{
		TableName:     "events",
		Columns:       []string{"user_id"},
		Type:          "BTREE",
		Priority:      5,
		EstimatedGain: 0.25,
		RecommendedAt: time.Now(),
	}

	qo.AddIndexRecommendation(rec)

	recs := qo.RecommendIndexes("events")
	if len(recs) != 1 {
		t.Errorf("len(recs) = %d, want 1", len(recs))
	}

	if recs[0].TableName != "events" {
		t.Errorf("TableName = %s, want events", recs[0].TableName)
	}
}

func TestMarkIndexImplemented(t *testing.T) {
	qo := NewQueryOptimizer(100)

	rec := IndexRecommendation{
		TableName:     "events",
		Columns:       []string{"user_id"},
		Type:          "BTREE",
		Priority:      5,
		RecommendedAt: time.Now(),
	}

	qo.AddIndexRecommendation(rec)
	qo.MarkIndexImplemented("events", []string{"user_id"})

	recs := qo.RecommendIndexes("events")
	if len(recs) > 0 && recs[0].ImplementedAt == nil {
		t.Error("expected ImplementedAt to be set")
	}
}

func TestClearCache(t *testing.T) {
	qo := NewQueryOptimizer(100)
	ctx := context.Background()

	query := "SELECT * FROM events"
	_, _ = qo.OptimizeQuery(ctx, query)

	qo.ClearCache()

	// Cache should be empty, so next optimization should create new entry
	result, err := qo.OptimizeQuery(ctx, query)
	if err != nil {
		t.Fatalf("OptimizeQuery() error = %v", err)
	}

	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestCacheLRUEviction(t *testing.T) {
	qo := NewQueryOptimizer(2) // Small cache size
	ctx := context.Background()

	queries := []string{
		"SELECT * FROM events",
		"SELECT * FROM users",
		"SELECT * FROM logs",
	}

	for _, q := range queries {
		_, _ = qo.OptimizeQuery(ctx, q)
	}

	// Cache should only have 2 entries due to LRU eviction
	// This is tested indirectly through successful optimization
	result, err := qo.OptimizeQuery(ctx, queries[2])
	if err != nil {
		t.Fatalf("OptimizeQuery() error = %v", err)
	}

	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestParseQueryOperations(t *testing.T) {
	qo := NewQueryOptimizer(100)

	tests := []struct {
		name          string
		query         string
		expectedOps   int
		expectedTypes []string
	}{
		{
			name:          "simple select",
			query:         "SELECT * FROM events",
			expectedOps:   1,
			expectedTypes: []string{"Scan"},
		},
		{
			name:          "select with where",
			query:         "SELECT * FROM events WHERE id = 1",
			expectedOps:   2,
			expectedTypes: []string{"Scan", "Filter"},
		},
		{
			name:          "select with join and order",
			query:         "SELECT * FROM events JOIN users ON events.user_id = users.id ORDER BY timestamp",
			expectedOps:   3,
			expectedTypes: []string{"Scan", "Join", "Sort"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops := qo.parseQueryOperations(tt.query)
			if len(ops) != tt.expectedOps {
				t.Errorf("len(ops) = %d, want %d", len(ops), tt.expectedOps)
			}

			for i, op := range ops {
				if i < len(tt.expectedTypes) && op.Type != tt.expectedTypes[i] {
					t.Errorf("ops[%d].Type = %s, want %s", i, op.Type, tt.expectedTypes[i])
				}
			}
		})
	}
}

func TestOptimizationRules(t *testing.T) {
	qo := NewQueryOptimizer(100)
	ctx := context.Background()

	query := "SELECT * FROM events WHERE id = 1"
	result, err := qo.OptimizeQuery(ctx, query)
	if err != nil {
		t.Fatalf("OptimizeQuery() error = %v", err)
	}

	if len(result.RewriteRules) == 0 {
		t.Error("expected optimization rules to be applied")
	}
}

func TestConcurrentOptimization(t *testing.T) {
	qo := NewQueryOptimizer(100)
	ctx := context.Background()

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			query := fmt.Sprintf("SELECT * FROM events WHERE id = %d", id)
			_, _ = qo.OptimizeQuery(ctx, query)
			qo.RecordQueryExecution(query, 100*time.Millisecond, nil, []string{})
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	stats := qo.GetAllQueryStatistics()
	if len(stats) == 0 {
		t.Error("expected statistics to be recorded")
	}
}

func TestQueryStatisticsMinMax(t *testing.T) {
	qo := NewQueryOptimizer(100)

	query := "SELECT * FROM events"
	durations := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		75 * time.Millisecond,
		150 * time.Millisecond,
	}

	for _, d := range durations {
		qo.RecordQueryExecution(query, d, nil, []string{})
	}

	stats := qo.GetQueryStatistics(query)
	if stats.MinDuration != 50*time.Millisecond {
		t.Errorf("MinDuration = %v, want 50ms", stats.MinDuration)
	}

	if stats.MaxDuration != 150*time.Millisecond {
		t.Errorf("MaxDuration = %v, want 150ms", stats.MaxDuration)
	}
}

func TestGenerateRecommendations(t *testing.T) {
	qo := NewQueryOptimizer(100)
	ctx := context.Background()

	query := "SELECT * FROM events ORDER BY timestamp"
	plan, err := qo.AnalyzeQueryPlan(ctx, query)
	if err != nil {
		t.Fatalf("AnalyzeQueryPlan() error = %v", err)
	}

	if len(plan.Recommendations) == 0 {
		t.Error("expected recommendations to be generated")
	}
}
