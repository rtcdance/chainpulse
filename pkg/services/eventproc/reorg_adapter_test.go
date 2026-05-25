package eventproc

import (
	"context"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestNewReorgDBAdapter(t *testing.T) {
	t.Parallel()
	a := NewReorgDBAdapter(nil, nil, nil)
	if a == nil {
		t.Fatal("expected non-nil adapter")
	}
	if a.Name() != "reorg-adapter" {
		t.Errorf("expected reorg-adapter, got %s", a.Name())
	}
	if a.Version() != "1.0.0" {
		t.Errorf("expected 1.0.0, got %s", a.Version())
	}
}

func TestReorgDBAdapter_Lifecycle(t *testing.T) {
	t.Parallel()
	a := NewReorgDBAdapter(nil, nil, nil)

	if err := a.Initialize(core.Config{}); err != nil {
		t.Errorf("Initialize: %v", err)
	}
	if err := a.Start(); err != nil {
		t.Errorf("Start: %v", err)
	}
	if err := a.Stop(); err != nil {
		t.Errorf("Stop: %v", err)
	}
	if err := a.Health(); err != nil {
		t.Errorf("Health: %v", err)
	}
}

func TestReorgDBAdapter_NotImplemented(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	a := NewReorgDBAdapter(nil, nil, nil)

	if _, err := a.GetEvent(ctx, "id"); err == nil {
		t.Error("GetEvent should return error")
	}
	if _, err := a.QueryEvents(ctx, nil); err == nil {
		t.Error("QueryEvents should return error")
	}
	if _, err := a.GetAllEvents(ctx); err == nil {
		t.Error("GetAllEvents should return error")
	}
	if err := a.StoreEvent(ctx, nil); err == nil {
		t.Error("StoreEvent should return error")
	}
	if err := a.BatchStoreEvents(ctx, nil); err == nil {
		t.Error("BatchStoreEvents should return error")
	}
	if err := a.DeleteEvent(ctx, "id"); err == nil {
		t.Error("DeleteEvent should return error")
	}
	if _, err := a.GetAllBlocks(ctx); err == nil {
		t.Error("GetAllBlocks should return error")
	}
}

func TestReorgDBAdapter_GetBlock_NilDB(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	a := NewReorgDBAdapter(nil, nil, nil)

	_, err := a.GetBlock(ctx, 1)
	if err == nil {
		t.Error("GetBlock with nil DB should return error")
	}
}

func TestReorgDBAdapter_GetLatestBlock_NilDB(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	a := NewReorgDBAdapter(nil, nil, nil)

	_, err := a.GetLatestBlock(ctx)
	if err == nil {
		t.Error("GetLatestBlock with nil DB should return error")
	}
}

func TestReorgDBAdapter_GetReorgStats(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	a := NewReorgDBAdapter(nil, nil, nil)

	stats, err := a.GetReorgStats(ctx)
	if err != nil {
		t.Errorf("GetReorgStats: %v", err)
	}
	if stats == nil {
		t.Fatal("expected non-nil ReorgStats")
	}
}
