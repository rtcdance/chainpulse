package database

import (
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

	if err := p.Initialize(&core.Config{ServiceName: "test"}); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}
	if err := p.Initialize(&core.Config{}); err == nil {
		t.Error("expected error for double Initialize")
	}

	if err := p.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if err := p.Start(); err == nil {
		t.Error("expected error for double Start")
	}

	if err := p.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
}

func TestBaseDatabasePlugin_InitializeNilConfig(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	if err := p.Initialize(nil); err == nil {
		t.Error("expected error for nil config")
	}
}

func TestBaseDatabasePlugin_StopNotStarted(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	if err := p.Stop(); err == nil {
		t.Error("expected error when stopping not started")
	}
}

func TestBaseDatabasePlugin_Health(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	defer p.Stop()

	health := p.Health()
	if health == nil {
		t.Fatal("Health() returned nil")
	}
	if health.Status != "healthy" {
		t.Errorf("Health().Status = %q", health.Status)
	}
}

func TestBaseDatabasePlugin_HealthBeforeInit(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	if h := p.Health(); h.Status != "unhealthy" {
		t.Errorf("Health() before Init = %q, want unhealthy", h.Status)
	}
}

func TestBaseDatabasePlugin_HealthAfterInitNotStarted(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	_ = p.Initialize(&core.Config{})
	if h := p.Health(); h.Status != "unhealthy" {
		t.Errorf("Health() after Init = %q, want unhealthy", h.Status)
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
	_ = p.Initialize(&core.Config{ServiceName: "test"})
	_ = p.Start()
	defer p.Stop()

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
	if err := p.Start(); err == nil {
		t.Error("expected error starting without init")
	}
}

func TestBaseDatabasePlugin_StopNotRunning(t *testing.T) {
	t.Parallel()
	p := newBasePlugin()
	_ = p.Initialize(&core.Config{})
	if err := p.Stop(); err == nil {
		t.Error("expected error stopping without start")
	}
}
