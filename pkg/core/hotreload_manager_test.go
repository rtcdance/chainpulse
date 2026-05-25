package core

import (
	"context"
	"io"
	"testing"
)

type testHotReloadPlugin struct {
	name        string
	reloadable  bool
	reloadOk    bool
	reloadErr   error
	reloadCalls int
}

func (p *testHotReloadPlugin) Name() string       { return p.name }
func (p *testHotReloadPlugin) IsReloadable() bool { return p.reloadable }
func (p *testHotReloadPlugin) Reload(ctx context.Context, cfg Config) (bool, error) {
	p.reloadCalls++
	return p.reloadOk, p.reloadErr
}

func TestNewDefaultHotReloadManager(t *testing.T) {
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, io.Discard)
	mgr := NewDefaultHotReloadManager(logger)
	if mgr == nil {
		t.Fatal("NewDefaultHotReloadManager returned nil")
	}
	if mgr.plugins == nil {
		t.Error("plugins map should be initialized")
	}
	names := mgr.ListPlugins()
	if len(names) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(names))
	}
}

func TestHotReloadManager_RegisterPlugin(t *testing.T) {
	mgr := NewDefaultHotReloadManager(nil)
	p := &testHotReloadPlugin{name: "test-plugin", reloadable: true}

	err := mgr.RegisterPlugin("test-plugin", p)
	if err != nil {
		t.Fatalf("RegisterPlugin error: %v", err)
	}

	names := mgr.ListPlugins()
	if len(names) != 1 || names[0] != "test-plugin" {
		t.Errorf("expected [test-plugin], got %v", names)
	}
}

func TestHotReloadManager_RegisterPluginDuplicate(t *testing.T) {
	mgr := NewDefaultHotReloadManager(nil)
	p := &testHotReloadPlugin{name: "test-plugin"}

	_ = mgr.RegisterPlugin("test-plugin", p)
	err := mgr.RegisterPlugin("test-plugin", p)
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestHotReloadManager_GetPlugin(t *testing.T) {
	mgr := NewDefaultHotReloadManager(nil)
	p := &testHotReloadPlugin{name: "test-plugin"}
	_ = mgr.RegisterPlugin("test-plugin", p)

	got, err := mgr.GetPlugin("test-plugin")
	if err != nil {
		t.Fatalf("GetPlugin error: %v", err)
	}
	if got != p {
		t.Error("GetPlugin returned wrong plugin")
	}
}

func TestHotReloadManager_GetPluginNotFound(t *testing.T) {
	mgr := NewDefaultHotReloadManager(nil)
	_, err := mgr.GetPlugin("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plugin")
	}
}

func TestHotReloadManager_ListPlugins(t *testing.T) {
	mgr := NewDefaultHotReloadManager(nil)
	_ = mgr.RegisterPlugin("a", &testHotReloadPlugin{name: "a"})
	_ = mgr.RegisterPlugin("b", &testHotReloadPlugin{name: "b"})

	names := mgr.ListPlugins()
	if len(names) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(names))
	}
}

func TestHotReloadManager_ReloadPluginNotFound(t *testing.T) {
	mgr := NewDefaultHotReloadManager(nil)
	ctx := context.Background()
	err := mgr.ReloadPlugin(ctx, "nonexistent", Config{})
	if err == nil {
		t.Error("expected error for nonexistent plugin")
	}
}

func TestHotReloadManager_ReloadPluginNotReloadable(t *testing.T) {
	mgr := NewDefaultHotReloadManager(nil)
	p := &testHotReloadPlugin{name: "p", reloadable: false}
	_ = mgr.RegisterPlugin("p", p)

	ctx := context.Background()
	err := mgr.ReloadPlugin(ctx, "p", Config{})
	if err == nil {
		t.Error("expected error for non-reloadable plugin")
	}
}

func TestHotReloadManager_ReloadPluginSuccess(t *testing.T) {
	mgr := NewDefaultHotReloadManager(nil)
	p := &testHotReloadPlugin{name: "p", reloadable: true, reloadOk: true}
	_ = mgr.RegisterPlugin("p", p)

	ctx := context.Background()
	err := mgr.ReloadPlugin(ctx, "p", Config{})
	if err != nil {
		t.Fatalf("ReloadPlugin error: %v", err)
	}
	if p.reloadCalls != 1 {
		t.Errorf("expected 1 reload call, got %d", p.reloadCalls)
	}
}

func TestHotReloadManager_ReloadPluginError(t *testing.T) {
	mgr := NewDefaultHotReloadManager(nil)
	p := &testHotReloadPlugin{name: "p", reloadable: true, reloadOk: false, reloadErr: context.DeadlineExceeded}
	_ = mgr.RegisterPlugin("p", p)

	ctx := context.Background()
	err := mgr.ReloadPlugin(ctx, "p", Config{})
	if err == nil {
		t.Error("expected error from reload")
	}
}

func TestHotReloadManager_ReloadPluginReturnFalse(t *testing.T) {
	mgr := NewDefaultHotReloadManager(nil)
	p := &testHotReloadPlugin{name: "p", reloadable: true, reloadOk: false, reloadErr: nil}
	_ = mgr.RegisterPlugin("p", p)

	ctx := context.Background()
	err := mgr.ReloadPlugin(ctx, "p", Config{})
	if err == nil {
		t.Error("expected error when reload returns false")
	}
}

func TestHotReloadManager_ReloadAll(t *testing.T) {
	mgr := NewDefaultHotReloadManager(nil)
	p1 := &testHotReloadPlugin{name: "p1", reloadable: true, reloadOk: true}
	p2 := &testHotReloadPlugin{name: "p2", reloadable: true, reloadOk: true}
	_ = mgr.RegisterPlugin("p1", p1)
	_ = mgr.RegisterPlugin("p2", p2)

	ctx := context.Background()
	err := mgr.ReloadAll(ctx, Config{})
	if err != nil {
		t.Fatalf("ReloadAll error: %v", err)
	}
	if p1.reloadCalls != 1 || p2.reloadCalls != 1 {
		t.Errorf("expected 1 reload call each, got p1=%d p2=%d", p1.reloadCalls, p2.reloadCalls)
	}
}

func TestHotReloadManager_ReloadAllWithFailure(t *testing.T) {
	mgr := NewDefaultHotReloadManager(nil)
	p1 := &testHotReloadPlugin{name: "p1", reloadable: true, reloadOk: false, reloadErr: context.DeadlineExceeded}
	p2 := &testHotReloadPlugin{name: "p2", reloadable: true, reloadOk: true}
	_ = mgr.RegisterPlugin("p1", p1)
	_ = mgr.RegisterPlugin("p2", p2)

	ctx := context.Background()
	err := mgr.ReloadAll(ctx, Config{})
	if err == nil {
		t.Error("expected error when one plugin fails")
	}
}

func TestHotReloadManager_RegisterPluginWithLogger(t *testing.T) {
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, io.Discard)
	mgr := NewDefaultHotReloadManager(logger)
	p := &testHotReloadPlugin{name: "test", reloadable: true}

	err := mgr.RegisterPlugin("test", p)
	if err != nil {
		t.Fatalf("RegisterPlugin error: %v", err)
	}
}

func TestHotReloadManager_ReloadPluginWithLogger(t *testing.T) {
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, io.Discard)
	mgr := NewDefaultHotReloadManager(logger)
	p := &testHotReloadPlugin{name: "test", reloadable: true, reloadOk: true}
	_ = mgr.RegisterPlugin("test", p)

	ctx := context.Background()
	err := mgr.ReloadPlugin(ctx, "test", Config{})
	if err != nil {
		t.Fatalf("ReloadPlugin error: %v", err)
	}
}

func TestHotReloadManager_ReloadAllWithLoggerAndFailure(t *testing.T) {
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, io.Discard)
	mgr := NewDefaultHotReloadManager(logger)
	p1 := &testHotReloadPlugin{name: "p1", reloadable: true, reloadOk: false, reloadErr: context.DeadlineExceeded}
	_ = mgr.RegisterPlugin("p1", p1)

	ctx := context.Background()
	err := mgr.ReloadAll(ctx, Config{})
	if err == nil {
		t.Error("expected error from failed reload")
	}
}
