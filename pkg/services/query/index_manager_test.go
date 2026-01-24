package query

import (
	"context"
	"testing"
	"time"
)

func TestNewIndexManager(t *testing.T) {
	im := NewIndexManager()
	if im == nil {
		t.Fatal("expected non-nil IndexManager")
	}
}

func TestCreateIndex(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	index, err := im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE")
	if err != nil {
		t.Fatalf("CreateIndex() error = %v", err)
	}

	if index == nil {
		t.Fatal("expected non-nil index")
	}

	if index.Name != "idx_events_id" {
		t.Errorf("Name = %s, want idx_events_id", index.Name)
	}

	if index.TableName != "events" {
		t.Errorf("TableName = %s, want events", index.TableName)
	}
}

func TestCreateIndexDuplicate(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	_, _ = im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE")

	_, err := im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE")
	if err == nil {
		t.Error("expected error for duplicate index")
	}
}

func TestCreateIndexInvalidParams(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	tests := []struct {
		name      string
		indexName string
		tableName string
		columns   []string
		wantErr   bool
	}{
		{
			name:      "empty index name",
			indexName: "",
			tableName: "events",
			columns:   []string{"id"},
			wantErr:   true,
		},
		{
			name:      "empty table name",
			indexName: "idx_events_id",
			tableName: "",
			columns:   []string{"id"},
			wantErr:   true,
		},
		{
			name:      "empty columns",
			indexName: "idx_events_id",
			tableName: "events",
			columns:   []string{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := im.CreateIndex(ctx, tt.indexName, tt.tableName, tt.columns, "BTREE")
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateIndex() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteIndex(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	_, _ = im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE")

	err := im.DeleteIndex("idx_events_id")
	if err != nil {
		t.Fatalf("DeleteIndex() error = %v", err)
	}

	index := im.GetIndex("idx_events_id")
	if index != nil {
		t.Error("expected index to be deleted")
	}
}

func TestDeleteIndexNotFound(t *testing.T) {
	im := NewIndexManager()

	err := im.DeleteIndex("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent index")
	}
}

func TestGetIndex(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	_, _ = im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE")

	index := im.GetIndex("idx_events_id")
	if index == nil {
		t.Fatal("expected non-nil index")
	}

	if index.Name != "idx_events_id" {
		t.Errorf("Name = %s, want idx_events_id", index.Name)
	}
}

func TestGetIndexesByTable(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	_, _ = im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE")
	_, _ = im.CreateIndex(ctx, "idx_events_user", "events", []string{"user_id"}, "BTREE")
	_, _ = im.CreateIndex(ctx, "idx_users_id", "users", []string{"id"}, "BTREE")

	indexes := im.GetIndexesByTable("events")
	if len(indexes) != 2 {
		t.Errorf("len(indexes) = %d, want 2", len(indexes))
	}

	for _, idx := range indexes {
		if idx.TableName != "events" {
			t.Errorf("TableName = %s, want events", idx.TableName)
		}
	}
}

func TestGetAllIndexes(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	if _, err := im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE"); err != nil {
		t.Logf("Failed to create index: %v", err)
	}
	if _, err := im.CreateIndex(ctx, "idx_users_id", "users", []string{"id"}, "BTREE"); err != nil {
		t.Logf("Failed to create index: %v", err)
	}

	indexes := im.GetAllIndexes()
	if len(indexes) != 2 {
		t.Errorf("len(indexes) = %d, want 2", len(indexes))
	}
}

func TestRecordIndexUsage(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	if _, err := im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE"); err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	im.RecordIndexUsage("idx_events_id", "select", 10*time.Millisecond)

	stats := im.GetIndexStatistics("idx_events_id")
	if stats == nil {
		t.Fatal("expected non-nil statistics")
	}

	if stats.UsageCount != 1 {
		t.Errorf("UsageCount = %d, want 1", stats.UsageCount)
	}

	if stats.SelectsUsed != 1 {
		t.Errorf("SelectsUsed = %d, want 1", stats.SelectsUsed)
	}
}

func TestRecordIndexUsageMultipleOperations(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	if _, err := im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE"); err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	im.RecordIndexUsage("idx_events_id", "select", 10*time.Millisecond)
	im.RecordIndexUsage("idx_events_id", "insert", 5*time.Millisecond)
	im.RecordIndexUsage("idx_events_id", "update", 8*time.Millisecond)
	im.RecordIndexUsage("idx_events_id", "delete", 6*time.Millisecond)

	stats := im.GetIndexStatistics("idx_events_id")
	if stats.UsageCount != 4 {
		t.Errorf("UsageCount = %d, want 4", stats.UsageCount)
	}

	if stats.SelectsUsed != 1 {
		t.Errorf("SelectsUsed = %d, want 1", stats.SelectsUsed)
	}

	if stats.InsertsUsed != 1 {
		t.Errorf("InsertsUsed = %d, want 1", stats.InsertsUsed)
	}

	if stats.UpdatesUsed != 1 {
		t.Errorf("UpdatesUsed = %d, want 1", stats.UpdatesUsed)
	}

	if stats.DeletesUsed != 1 {
		t.Errorf("DeletesUsed = %d, want 1", stats.DeletesUsed)
	}
}

func TestGetAllIndexStatistics(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	_, _ = im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE")
	_, _ = im.CreateIndex(ctx, "idx_users_id", "users", []string{"id"}, "BTREE")

	im.RecordIndexUsage("idx_events_id", "select", 10*time.Millisecond)
	im.RecordIndexUsage("idx_users_id", "select", 15*time.Millisecond)

	stats := im.GetAllIndexStatistics()
	if len(stats) != 2 {
		t.Errorf("len(stats) = %d, want 2", len(stats))
	}
}

func TestAnalyzeIndexFragmentation(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	_, _ = im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE")

	frag, err := im.AnalyzeIndexFragmentation("idx_events_id")
	if err != nil {
		t.Fatalf("AnalyzeIndexFragmentation() error = %v", err)
	}

	if frag < 0 || frag > 100 {
		t.Errorf("fragmentation = %f, want 0-100", frag)
	}
}

func TestAnalyzeIndexFragmentationNotFound(t *testing.T) {
	im := NewIndexManager()

	_, err := im.AnalyzeIndexFragmentation("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent index")
	}
}

func TestRebuildIndex(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	_, _ = im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE")

	err := im.RebuildIndex(ctx, "idx_events_id")
	if err != nil {
		t.Fatalf("RebuildIndex() error = %v", err)
	}

	index := im.GetIndex("idx_events_id")
	if index.Fragmentation != 0 {
		t.Errorf("Fragmentation = %f, want 0", index.Fragmentation)
	}
}

func TestGetUnusedIndexes(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	_, _ = im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE")
	_, _ = im.CreateIndex(ctx, "idx_events_user", "events", []string{"user_id"}, "BTREE")

	im.RecordIndexUsage("idx_events_id", "select", 10*time.Millisecond)

	unused := im.GetUnusedIndexes(1 * time.Millisecond)
	if len(unused) != 1 {
		t.Errorf("len(unused) = %d, want 1", len(unused))
	}

	if unused[0].Name != "idx_events_user" {
		t.Errorf("Name = %s, want idx_events_user", unused[0].Name)
	}
}

func TestGetIneffectiveIndexes(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	_, _ = im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE")

	// Record mostly non-select operations
	im.RecordIndexUsage("idx_events_id", "insert", 5*time.Millisecond)
	im.RecordIndexUsage("idx_events_id", "insert", 5*time.Millisecond)
	im.RecordIndexUsage("idx_events_id", "insert", 5*time.Millisecond)
	im.RecordIndexUsage("idx_events_id", "select", 10*time.Millisecond)

	ineffective := im.GetIneffectiveIndexes(50)
	if len(ineffective) != 1 {
		t.Errorf("len(ineffective) = %d, want 1", len(ineffective))
	}
}

func TestValidateIndex(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	_, _ = im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE")

	valid, err := im.ValidateIndex(ctx, "idx_events_id")
	if err != nil {
		t.Fatalf("ValidateIndex() error = %v", err)
	}

	if !valid {
		t.Error("expected index to be valid")
	}
}

func TestUpdateIndexStatistics(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	_, _ = im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE")

	err := im.UpdateIndexStatistics("idx_events_id", 1000, 50000)
	if err != nil {
		t.Fatalf("UpdateIndexStatistics() error = %v", err)
	}

	index := im.GetIndex("idx_events_id")
	if index.RowCount != 1000 {
		t.Errorf("RowCount = %d, want 1000", index.RowCount)
	}

	if index.SizeBytes != 50000 {
		t.Errorf("SizeBytes = %d, want 50000", index.SizeBytes)
	}
}

func TestCreatePendingIndex(t *testing.T) {
	im := NewIndexManager()

	pending := im.CreatePendingIndex("idx_events_id", "events", []string{"id"}, "BTREE", 5)
	if pending == nil {
		t.Fatal("expected non-nil pending index")
	}

	if pending.Name != "idx_events_id" {
		t.Errorf("Name = %s, want idx_events_id", pending.Name)
	}

	if pending.Status != "pending" {
		t.Errorf("Status = %s, want pending", pending.Status)
	}
}

func TestGetPendingIndexes(t *testing.T) {
	im := NewIndexManager()

	im.CreatePendingIndex("idx_events_id", "events", []string{"id"}, "BTREE", 5)
	im.CreatePendingIndex("idx_events_user", "events", []string{"user_id"}, "BTREE", 3)

	pending := im.GetPendingIndexes()
	if len(pending) != 2 {
		t.Errorf("len(pending) = %d, want 2", len(pending))
	}
}

func TestUpdatePendingIndexStatus(t *testing.T) {
	im := NewIndexManager()

	im.CreatePendingIndex("idx_events_id", "events", []string{"id"}, "BTREE", 5)

	err := im.UpdatePendingIndexStatus("idx_events_id", "created", "")
	if err != nil {
		t.Fatalf("UpdatePendingIndexStatus() error = %v", err)
	}

	pending := im.GetPendingIndexes()
	if len(pending) != 0 {
		t.Errorf("len(pending) = %d, want 0", len(pending))
	}
}

func TestGetIndexSize(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	_, _ = im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE")
	_ = im.UpdateIndexStatistics("idx_events_id", 1000, 50000)

	size, err := im.GetIndexSize("idx_events_id")
	if err != nil {
		t.Fatalf("GetIndexSize() error = %v", err)
	}

	if size != 50000 {
		t.Errorf("size = %d, want 50000", size)
	}
}

func TestGetTotalIndexSize(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	_, _ = im.CreateIndex(ctx, "idx_events_id", "events", []string{"id"}, "BTREE")
	_, _ = im.CreateIndex(ctx, "idx_users_id", "users", []string{"id"}, "BTREE")

	_ = im.UpdateIndexStatistics("idx_events_id", 1000, 50000)
	_ = im.UpdateIndexStatistics("idx_users_id", 500, 25000)

	total := im.GetTotalIndexSize()
	if total != 75000 {
		t.Errorf("total = %d, want 75000", total)
	}
}

func TestConcurrentIndexOperations(t *testing.T) {
	im := NewIndexManager()
	ctx := context.Background()

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			indexName := "idx_" + string(rune(id))
			_, _ = im.CreateIndex(ctx, indexName, "events", []string{"id"}, "BTREE")
			im.RecordIndexUsage(indexName, "select", 10*time.Millisecond)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	indexes := im.GetAllIndexes()
	if len(indexes) != 10 {
		t.Errorf("len(indexes) = %d, want 10", len(indexes))
	}
}
