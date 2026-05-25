package api

import (
	"database/sql"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/services/reorg"
)

func TestNewBlockConfirmationTracker(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	hub := NewSubscriptionHub()

	tracker := NewBlockConfirmationTracker(nil, logger, metrics, hub, nil)
	if tracker == nil {
		t.Fatal("Expected non-nil tracker")
	}
	if tracker.checkInterval != 30*time.Second {
		t.Fatalf("Expected default checkInterval 30s, got %v", tracker.checkInterval)
	}
	if len(tracker.confirmationMap) == 0 {
		t.Fatal("Expected non-empty confirmationMap defaults")
	}
	if tracker.reorgHandlers == nil {
		t.Fatal("Expected non-nil reorgHandlers map")
	}
}

func TestNewBlockConfirmationTracker_CustomMap(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	hub := NewSubscriptionHub()
	customMap := map[uint64]int{1: 6, 56: 10}

	tracker := NewBlockConfirmationTracker(nil, logger, metrics, hub, customMap)
	if tracker.confirmationMap[1] != 6 {
		t.Fatalf("Expected custom confirmation 6 for chain 1, got %d", tracker.confirmationMap[1])
	}
}

func TestBlockConfirmationTracker_RegisterReorgHandler(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	hub := NewSubscriptionHub()
	tracker := NewBlockConfirmationTracker(nil, logger, metrics, hub, nil)

	handler := &reorg.ReorgHandler{}
	tracker.RegisterReorgHandler("ethereum", handler)

	tracker.mu.RLock()
	stored := tracker.reorgHandlers["ethereum"]
	tracker.mu.RUnlock()
	if stored != handler {
		t.Fatal("Expected stored handler to match registered handler")
	}
}

func TestBlockConfirmationTracker_SetCheckInterval(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	hub := NewSubscriptionHub()
	tracker := NewBlockConfirmationTracker(nil, logger, metrics, hub, nil)

	tracker.SetCheckInterval(60 * time.Second)
	if tracker.checkInterval != 60*time.Second {
		t.Fatalf("Expected checkInterval 60s, got %v", tracker.checkInterval)
	}
}

func TestBlockConfirmationTracker_SetFinalityChecker(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	hub := NewSubscriptionHub()
	tracker := NewBlockConfirmationTracker(nil, logger, metrics, hub, nil)

	tracker.SetFinalityChecker(nil)
	if tracker.finalityChecker != nil {
		t.Fatal("Expected nil finalityChecker after setting nil")
	}
}

func TestBlockConfirmationTracker_StartNoDB(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	hub := NewSubscriptionHub()
	tracker := NewBlockConfirmationTracker(nil, logger, metrics, hub, nil)

	err := tracker.Start(t.Context())
	if err == nil {
		t.Fatal("Expected error starting with nil DB")
	}
}

func TestBlockConfirmationTracker_Stop(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	hub := NewSubscriptionHub()
	tracker := NewBlockConfirmationTracker(nil, logger, metrics, hub, nil)

	tracker.Stop()
	if tracker.running {
		t.Fatal("Expected not running after Stop")
	}
}

func TestBlockConfirmationTracker_GetConfirmationStats_NoDB(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	hub := NewSubscriptionHub()
	tracker := NewBlockConfirmationTracker(nil, logger, metrics, hub, nil)

	_, err := tracker.GetConfirmationStats(t.Context())
	if err == nil {
		t.Fatal("Expected error for GetConfirmationStats with nil DB")
	}
}

func TestBlockConfirmationTracker_GetConfirmationBlocks_Numeric(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	hub := NewSubscriptionHub()
	tracker := NewBlockConfirmationTracker(nil, logger, metrics, hub, nil)

	n := tracker.getConfirmationBlocks("1")
	if n != 12 {
		t.Fatalf("Expected 12 confirmations for chain 1, got %d", n)
	}
}

func TestBlockConfirmationTracker_GetConfirmationBlocks_Named(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	hub := NewSubscriptionHub()
	tracker := NewBlockConfirmationTracker(nil, logger, metrics, hub, nil)

	n := tracker.getConfirmationBlocks("arbitrum")
	if n != 12 {
		t.Fatalf("Expected 12 confirmations for arbitrum, got %d", n)
	}
}

func TestBlockConfirmationTracker_GetConfirmationBlocks_Unknown(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	hub := NewSubscriptionHub()
	tracker := NewBlockConfirmationTracker(nil, logger, metrics, hub, nil)

	n := tracker.getConfirmationBlocks("unknown_chain_99999")
	if n != 12 {
		t.Fatalf("Expected default 12 confirmations for unknown chain, got %d", n)
	}
}

func TestBlockConfirmationTracker_DetectReorgs_NoHandler(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	hub := NewSubscriptionHub()
	tracker := NewBlockConfirmationTracker(&sql.DB{}, logger, metrics, hub, nil)

	tracker.detectReorgs(t.Context(), "ethereum")
}
