package query

import (
	"testing"
	"time"
)

func TestNewQueryStatisticsCollector(t *testing.T) {
	qsc := NewQueryStatisticsCollector(1 * time.Minute)
	if qsc == nil {
		t.Fatal("expected non-nil QueryStatisticsCollector")
	}
}

func TestRecordExecution(t *testing.T) {
	qsc := NewQueryStatisticsCollector(1 * time.Minute)

	record := ExecutionRecord{
		QueryHash:    "query_hash_1",
		Duration:     100 * time.Millisecond,
		RowsReturned: 50,
		RowsScanned:  100,
		CacheHit:     false,
		Error:        nil,
		IndexesUsed:  []string{"idx_1"},
		Timestamp:    time.Now(),
	}

	qsc.RecordExecution(record)

	metrics := qsc.GetQueryMetrics("query_hash_1")
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}

	if metrics.ExecutionCount != 1 {
		t.Errorf("ExecutionCount = %d, want 1", metrics.ExecutionCount)
	}

	if metrics.TotalDuration != 100*time.Millisecond {
		t.Errorf("TotalDuration = %v, want 100ms", metrics.TotalDuration)
	}

	if metrics.RowsReturned != 50 {
		t.Errorf("RowsReturned = %d, want 50", metrics.RowsReturned)
	}

	if metrics.RowsScanned != 100 {
		t.Errorf("RowsScanned = %d, want 100", metrics.RowsScanned)
	}

	if metrics.CacheMisses != 1 {
		t.Errorf("CacheMisses = %d, want 1", metrics.CacheMisses)
	}
}

func TestRecordExecutionWithError(t *testing.T) {
	qsc := NewQueryStatisticsCollector(1 * time.Minute)

	record := ExecutionRecord{
		QueryHash:    "query_hash_1",
		Duration:     100 * time.Millisecond,
		RowsReturned: 0,
		RowsScanned:  0,
		CacheHit:     false,
		Error:        ErrQueryOptimizationFailed,
		IndexesUsed:  []string{},
		Timestamp:    time.Now(),
	}

	qsc.RecordExecution(record)

	metrics := qsc.GetQueryMetrics("query_hash_1")
	if metrics.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", metrics.ErrorCount)
	}

	if metrics.SuccessCount != 0 {
		t.Errorf("SuccessCount = %d, want 0", metrics.SuccessCount)
	}
}

func TestRecordExecutionWithCacheHit(t *testing.T) {
	qsc := NewQueryStatisticsCollector(1 * time.Minute)

	record := ExecutionRecord{
		QueryHash:    "query_hash_1",
		Duration:     10 * time.Millisecond,
		RowsReturned: 50,
		RowsScanned:  0,
		CacheHit:     true,
		Error:        nil,
		IndexesUsed:  []string{},
		Timestamp:    time.Now(),
	}

	qsc.RecordExecution(record)

	metrics := qsc.GetQueryMetrics("query_hash_1")
	if metrics.CacheHits != 1 {
		t.Errorf("CacheHits = %d, want 1", metrics.CacheHits)
	}

	if metrics.CacheMisses != 0 {
		t.Errorf("CacheMisses = %d, want 0", metrics.CacheMisses)
	}
}

func TestGetAllQueryMetrics(t *testing.T) {
	qsc := NewQueryStatisticsCollector(1 * time.Minute)

	for i := 0; i < 5; i++ {
		record := ExecutionRecord{
			QueryHash:    "query_hash_" + string(rune(i)),
			Duration:     100 * time.Millisecond,
			RowsReturned: 50,
			RowsScanned:  100,
			CacheHit:     false,
			Error:        nil,
			IndexesUsed:  []string{},
			Timestamp:    time.Now(),
		}
		qsc.RecordExecution(record)
	}

	metrics := qsc.GetAllQueryMetrics()
	if len(metrics) != 5 {
		t.Errorf("len(metrics) = %d, want 5", len(metrics))
	}
}

func TestAggregateMetrics(t *testing.T) {
	qsc := NewQueryStatisticsCollector(1 * time.Millisecond)

	for i := 0; i < 3; i++ {
		record := ExecutionRecord{
			QueryHash:    "query_hash_1",
			Duration:     100 * time.Millisecond,
			RowsReturned: 50,
			RowsScanned:  100,
			CacheHit:     i%2 == 0,
			Error:        nil,
			IndexesUsed:  []string{},
			Timestamp:    time.Now(),
		}
		qsc.RecordExecution(record)
	}

	agg := qsc.AggregateMetrics()
	if agg == nil {
		t.Fatal("expected non-nil aggregated metrics")
	}

	if agg.TotalQueries != 1 {
		t.Errorf("TotalQueries = %d, want 1", agg.TotalQueries)
	}

	if agg.TotalExecutions != 3 {
		t.Errorf("TotalExecutions = %d, want 3", agg.TotalExecutions)
	}

	if agg.SuccessRate != 100 {
		t.Errorf("SuccessRate = %f, want 100", agg.SuccessRate)
	}

	if agg.ErrorRate != 0 {
		t.Errorf("ErrorRate = %f, want 0", agg.ErrorRate)
	}
}

func TestAggregateMetricsWithErrors(t *testing.T) {
	qsc := NewQueryStatisticsCollector(1 * time.Millisecond)

	// Record 2 successful executions
	for i := 0; i < 2; i++ {
		record := ExecutionRecord{
			QueryHash:    "query_hash_1",
			Duration:     100 * time.Millisecond,
			RowsReturned: 50,
			RowsScanned:  100,
			CacheHit:     false,
			Error:        nil,
			IndexesUsed:  []string{},
			Timestamp:    time.Now(),
		}
		qsc.RecordExecution(record)
	}

	// Record 1 failed execution
	record := ExecutionRecord{
		QueryHash:    "query_hash_1",
		Duration:     100 * time.Millisecond,
		RowsReturned: 0,
		RowsScanned:  0,
		CacheHit:     false,
		Error:        ErrQueryOptimizationFailed,
		IndexesUsed:  []string{},
		Timestamp:    time.Now(),
	}
	qsc.RecordExecution(record)

	agg := qsc.AggregateMetrics()
	if agg.TotalExecutions != 3 {
		t.Errorf("TotalExecutions = %d, want 3", agg.TotalExecutions)
	}

	expectedSuccessRate := float64(2) / 3 * 100
	if agg.SuccessRate < expectedSuccessRate-1 || agg.SuccessRate > expectedSuccessRate+1 {
		t.Errorf("SuccessRate = %f, want ~%f", agg.SuccessRate, expectedSuccessRate)
	}

	expectedErrorRate := float64(1) / 3 * 100
	if agg.ErrorRate < expectedErrorRate-1 || agg.ErrorRate > expectedErrorRate+1 {
		t.Errorf("ErrorRate = %f, want ~%f", agg.ErrorRate, expectedErrorRate)
	}
}

func TestGetAggregatedMetrics(t *testing.T) {
	qsc := NewQueryStatisticsCollector(1 * time.Millisecond)

	record := ExecutionRecord{
		QueryHash:    "query_hash_1",
		Duration:     100 * time.Millisecond,
		RowsReturned: 50,
		RowsScanned:  100,
		CacheHit:     false,
		Error:        nil,
		IndexesUsed:  []string{},
		Timestamp:    time.Now(),
	}
	qsc.RecordExecution(record)

	agg1 := qsc.AggregateMetrics()
	agg2 := qsc.GetAggregatedMetrics()

	if agg1 != agg2 {
		t.Error("expected same aggregated metrics")
	}
}

func TestResetStatistics(t *testing.T) {
	qsc := NewQueryStatisticsCollector(1 * time.Minute)

	record := ExecutionRecord{
		QueryHash:    "query_hash_1",
		Duration:     100 * time.Millisecond,
		RowsReturned: 50,
		RowsScanned:  100,
		CacheHit:     false,
		Error:        nil,
		IndexesUsed:  []string{},
		Timestamp:    time.Now(),
	}
	qsc.RecordExecution(record)

	qsc.ResetStatistics()

	metrics := qsc.GetAllQueryMetrics()
	if len(metrics) != 0 {
		t.Errorf("len(metrics) = %d, want 0", len(metrics))
	}
}

func TestGetQueryCount(t *testing.T) {
	qsc := NewQueryStatisticsCollector(1 * time.Minute)

	for i := 0; i < 5; i++ {
		record := ExecutionRecord{
			QueryHash:    "query_hash_" + string(rune(i)),
			Duration:     100 * time.Millisecond,
			RowsReturned: 50,
			RowsScanned:  100,
			CacheHit:     false,
			Error:        nil,
			IndexesUsed:  []string{},
			Timestamp:    time.Now(),
		}
		qsc.RecordExecution(record)
	}

	count := qsc.GetQueryCount()
	if count != 5 {
		t.Errorf("GetQueryCount() = %d, want 5", count)
	}
}

func TestCalculatePercentiles(t *testing.T) {
	qsc := NewQueryStatisticsCollector(1 * time.Minute)

	record := ExecutionRecord{
		QueryHash:    "query_hash_1",
		Duration:     100 * time.Millisecond,
		RowsReturned: 50,
		RowsScanned:  100,
		CacheHit:     false,
		Error:        nil,
		IndexesUsed:  []string{},
		Timestamp:    time.Now(),
	}
	qsc.RecordExecution(record)

	percentiles := qsc.CalculatePercentiles("query_hash_1")
	if percentiles == nil {
		t.Fatal("expected non-nil percentiles")
	}

	if _, ok := percentiles["p50"]; !ok {
		t.Error("expected p50 percentile")
	}

	if _, ok := percentiles["p95"]; !ok {
		t.Error("expected p95 percentile")
	}

	if _, ok := percentiles["p99"]; !ok {
		t.Error("expected p99 percentile")
	}
}

func TestGetCacheHitRate(t *testing.T) {
	qsc := NewQueryStatisticsCollector(1 * time.Minute)

	// Record 3 cache hits and 1 cache miss
	for i := 0; i < 3; i++ {
		record := ExecutionRecord{
			QueryHash:    "query_hash_1",
			Duration:     10 * time.Millisecond,
			RowsReturned: 50,
			RowsScanned:  0,
			CacheHit:     true,
			Error:        nil,
			IndexesUsed:  []string{},
			Timestamp:    time.Now(),
		}
		qsc.RecordExecution(record)
	}

	record := ExecutionRecord{
		QueryHash:    "query_hash_1",
		Duration:     100 * time.Millisecond,
		RowsReturned: 50,
		RowsScanned:  100,
		CacheHit:     false,
		Error:        nil,
		IndexesUsed:  []string{},
		Timestamp:    time.Now(),
	}
	qsc.RecordExecution(record)

	hitRate := qsc.GetCacheHitRate("query_hash_1")
	expectedRate := 75.0
	if hitRate < expectedRate-1 || hitRate > expectedRate+1 {
		t.Errorf("GetCacheHitRate() = %f, want ~%f", hitRate, expectedRate)
	}
}

func TestGetErrorRate(t *testing.T) {
	qsc := NewQueryStatisticsCollector(1 * time.Minute)

	// Record 2 successful executions
	for i := 0; i < 2; i++ {
		record := ExecutionRecord{
			QueryHash:    "query_hash_1",
			Duration:     100 * time.Millisecond,
			RowsReturned: 50,
			RowsScanned:  100,
			CacheHit:     false,
			Error:        nil,
			IndexesUsed:  []string{},
			Timestamp:    time.Now(),
		}
		qsc.RecordExecution(record)
	}

	// Record 1 failed execution
	record := ExecutionRecord{
		QueryHash:    "query_hash_1",
		Duration:     100 * time.Millisecond,
		RowsReturned: 0,
		RowsScanned:  0,
		CacheHit:     false,
		Error:        ErrQueryOptimizationFailed,
		IndexesUsed:  []string{},
		Timestamp:    time.Now(),
	}
	qsc.RecordExecution(record)

	errorRate := qsc.GetErrorRate("query_hash_1")
	expectedRate := float64(1) / 3 * 100
	if errorRate < expectedRate-1 || errorRate > expectedRate+1 {
		t.Errorf("GetErrorRate() = %f, want ~%f", errorRate, expectedRate)
	}
}

func TestMultipleQueries(t *testing.T) {
	qsc := NewQueryStatisticsCollector(1 * time.Minute)

	// Record executions for multiple queries
	for q := 0; q < 3; q++ {
		for i := 0; i < 5; i++ {
			record := ExecutionRecord{
				QueryHash:    "query_hash_" + string(rune(q)),
				Duration:     time.Duration(100+q*10) * time.Millisecond,
				RowsReturned: int64(50 + q*10),
				RowsScanned:  int64(100 + q*20),
				CacheHit:     i%2 == 0,
				Error:        nil,
				IndexesUsed:  []string{},
				Timestamp:    time.Now(),
			}
			qsc.RecordExecution(record)
		}
	}

	metrics := qsc.GetAllQueryMetrics()
	if len(metrics) != 3 {
		t.Errorf("len(metrics) = %d, want 3", len(metrics))
	}

	agg := qsc.AggregateMetrics()
	if agg.TotalQueries != 3 {
		t.Errorf("TotalQueries = %d, want 3", agg.TotalQueries)
	}

	if agg.TotalExecutions != 15 {
		t.Errorf("TotalExecutions = %d, want 15", agg.TotalExecutions)
	}
}

func TestConcurrentRecording(t *testing.T) {
	qsc := NewQueryStatisticsCollector(1 * time.Minute)

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 5; j++ {
				record := ExecutionRecord{
					QueryHash:    "query_hash_" + string(rune(id)),
					Duration:     100 * time.Millisecond,
					RowsReturned: 50,
					RowsScanned:  100,
					CacheHit:     j%2 == 0,
					Error:        nil,
					IndexesUsed:  []string{},
					Timestamp:    time.Now(),
				}
				qsc.RecordExecution(record)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	metrics := qsc.GetAllQueryMetrics()
	if len(metrics) != 10 {
		t.Errorf("len(metrics) = %d, want 10", len(metrics))
	}

	agg := qsc.AggregateMetrics()
	if agg.TotalExecutions != 50 {
		t.Errorf("TotalExecutions = %d, want 50", agg.TotalExecutions)
	}
}

func TestAverageCalculations(t *testing.T) {
	qsc := NewQueryStatisticsCollector(1 * time.Minute)

	durations := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		150 * time.Millisecond,
	}

	for _, d := range durations {
		record := ExecutionRecord{
			QueryHash:    "query_hash_1",
			Duration:     d,
			RowsReturned: 50,
			RowsScanned:  100,
			CacheHit:     false,
			Error:        nil,
			IndexesUsed:  []string{},
			Timestamp:    time.Now(),
		}
		qsc.RecordExecution(record)
	}

	metrics := qsc.GetQueryMetrics("query_hash_1")
	expectedAvg := (50 + 100 + 150) / 3 * time.Millisecond
	if metrics.AverageDuration != expectedAvg {
		t.Errorf("AverageDuration = %v, want %v", metrics.AverageDuration, expectedAvg)
	}

	if metrics.MinDuration != 50*time.Millisecond {
		t.Errorf("MinDuration = %v, want 50ms", metrics.MinDuration)
	}

	if metrics.MaxDuration != 150*time.Millisecond {
		t.Errorf("MaxDuration = %v, want 150ms", metrics.MaxDuration)
	}
}

func TestTopQueriesSorting(t *testing.T) {
	qsc := NewQueryStatisticsCollector(1 * time.Millisecond)

	// Record different execution counts for different queries
	executionCounts := []int{5, 10, 3, 15, 8}
	for i, count := range executionCounts {
		for j := 0; j < count; j++ {
			record := ExecutionRecord{
				QueryHash:    "query_hash_" + string(rune(i)),
				Duration:     100 * time.Millisecond,
				RowsReturned: 50,
				RowsScanned:  100,
				CacheHit:     false,
				Error:        nil,
				IndexesUsed:  []string{},
				Timestamp:    time.Now(),
			}
			qsc.RecordExecution(record)
		}
	}

	agg := qsc.AggregateMetrics()
	if len(agg.TopQueries) == 0 {
		t.Fatal("expected top queries")
	}

	// Top query should have highest execution count
	if agg.TopQueries[0].ExecutionCount != 15 {
		t.Errorf("TopQueries[0].ExecutionCount = %d, want 15", agg.TopQueries[0].ExecutionCount)
	}
}
