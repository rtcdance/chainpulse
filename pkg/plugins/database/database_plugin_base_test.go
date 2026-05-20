package database

import (
	"context"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func newBasePlugin() *BaseDatabasePlugin {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	return NewBaseDatabasePlugin(logger, metrics)
}

func TestBaseDatabasePlugin_Lifecycle(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	ctx := context.Background()

	if err := p.Initialize(ctx, core.Config{ServiceName: "test"}); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}
	if err := p.Initialize(ctx, core.Config{ServiceName: "test"}); err == nil {
		t.Error("expected error for double Initialize")
	}

	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if err := p.Start(ctx); err == nil {
		t.Error("expected error for double Start")
	}

	if err := p.Stop(ctx); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
}

func TestBaseDatabasePlugin_InitializeNilConfig(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	ctx := context.Background()
	if err := p.Initialize(ctx, core.Config{}); err == nil {
		t.Error("expected error for nil config")
	}
}

func TestBaseDatabasePlugin_StopNotStarted(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	ctx := context.Background()
	if err := p.Stop(ctx); err == nil {
		t.Error("expected error when stopping not started")
	}
}

func TestBaseDatabasePlugin_Health(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	ctx := context.Background()
	_ = p.Initialize(ctx, core.Config{ServiceName: "test"})
	_ = p.Start(ctx)
	defer func() { _ = p.Stop(ctx) }()

	if err := p.Health(ctx); err != nil {
		t.Errorf("Health() = %v, want nil", err)
	}
}

func TestBaseDatabasePlugin_HealthBeforeInit(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	ctx := context.Background()
	if err := p.Health(ctx); err == nil {
		t.Error("Health() before Init = nil, want error")
	}
}

func TestBaseDatabasePlugin_HealthAfterInitNotStarted(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	ctx := context.Background()
	_ = p.Initialize(ctx, core.Config{})
	if err := p.Health(ctx); err == nil {
		t.Error("Health() after Init = nil, want error")
	}
}

func TestBaseDatabasePlugin_RecordWrite(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	p.RecordWrite(100)
	p.RecordWrite(200)
	if p.GetWriteCount() != 2 {
		t.Errorf("GetWriteCount() = %d, want 2", p.GetWriteCount())
	}
}

func TestBaseDatabasePlugin_RecordRead(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	p.RecordRead(50)
	if p.GetReadCount() != 1 {
		t.Errorf("GetReadCount() = %d, want 1", p.GetReadCount())
	}
}

func TestBaseDatabasePlugin_RecordDelete(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	p.RecordDelete()
	if p.GetDeleteCount() != 1 {
		t.Errorf("GetDeleteCount() = %d, want 1", p.GetDeleteCount())
	}
}

func TestBaseDatabasePlugin_RecordError(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	p.RecordError()
	p.RecordError()
	if p.GetErrorCount() != 2 {
		t.Errorf("GetErrorCount() = %d, want 2", p.GetErrorCount())
	}
}

func TestBaseDatabasePlugin_UpdateEventCount(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	p.UpdateEventCount(42)
	if p.GetEventCount() != 42 {
		t.Errorf("GetEventCount() = %d, want 42", p.GetEventCount())
	}
}

func TestBaseDatabasePlugin_UpdateTotalSize(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	p.UpdateTotalSize(2048)
	if p.GetTotalSize() != 2048 {
		t.Errorf("GetTotalSize() = %d, want 2048", p.GetTotalSize())
	}
}

func TestBaseDatabasePlugin_GetterMethods(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	ctx := context.Background()
	_ = p.Initialize(ctx, core.Config{ServiceName: "test"})
	_ = p.Start(ctx)
	defer func() { _ = p.Stop(ctx) }()

	p.RecordWrite(100)
	p.RecordWrite(200)
	p.RecordRead(50)
	p.RecordDelete()
	p.RecordError()
	p.RecordError()
	p.UpdateEventCount(7)
	p.UpdateTotalSize(4096)

	if p.GetWriteCount() != 2 {
		t.Errorf("GetWriteCount() = %d", p.GetWriteCount())
	}
	if p.GetReadCount() != 1 {
		t.Errorf("GetReadCount() = %d", p.GetReadCount())
	}
	if p.GetDeleteCount() != 1 {
		t.Errorf("GetDeleteCount() = %d", p.GetDeleteCount())
	}
	if p.GetErrorCount() != 2 {
		t.Errorf("GetErrorCount() = %d", p.GetErrorCount())
	}
	if p.GetEventCount() != 7 {
		t.Errorf("GetEventCount() = %d", p.GetEventCount())
	}
	if p.GetTotalSize() != 4096 {
		t.Errorf("GetTotalSize() = %d", p.GetTotalSize())
	}
}

func TestBaseDatabasePlugin_StartErrorHandling(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	ctx := context.Background()
	if err := p.Start(ctx); err == nil {
		t.Error("expected error starting without init")
	}
}

func TestBaseDatabasePlugin_StopNotRunning(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	ctx := context.Background()
	_ = p.Initialize(ctx, core.Config{})
	if err := p.Stop(ctx); err == nil {
		t.Error("expected error stopping without start")
	}
}
