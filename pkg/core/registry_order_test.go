package core

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

type depPlugin struct {
	name         string
	version      string
	dependencies []string
	startCalled  int32
	stopCalled   int32
	startErr     error
	stopErr      error
}

func (p *depPlugin) Name() string            { return p.name }
func (p *depPlugin) Version() string         { return p.version }
func (p *depPlugin) Dependencies() []string  { return p.dependencies }
func (p *depPlugin) Initialize(_ context.Context, _ Config) error { return nil }
func (p *depPlugin) Start(_ context.Context) error {
	atomic.AddInt32(&p.startCalled, 1)
	return p.startErr
}
func (p *depPlugin) Stop(_ context.Context) error {
	atomic.AddInt32(&p.stopCalled, 1)
	return p.stopErr
}
func (p *depPlugin) Health(_ context.Context) error { return nil }

func TestSortTopologicalNoDeps(t *testing.T) {
	t.Parallel()
	plugins := []Plugin{
		&depPlugin{name: "a"},
		&depPlugin{name: "b"},
		&depPlugin{name: "c"},
	}
	sorted, err := sortTopological(plugins)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(sorted))
	}
}

func TestSortTopologicalLinearDeps(t *testing.T) {
	t.Parallel()
	a := &depPlugin{name: "a"}
	b := &depPlugin{name: "b", dependencies: []string{"a"}}
	c := &depPlugin{name: "c", dependencies: []string{"b"}}
	plugins := []Plugin{c, a, b} // reverse order to test sorting

	sorted, err := sortTopological(plugins)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(sorted))
	}
	index := make(map[string]int)
	for i, p := range sorted {
		index[p.Name()] = i
	}
	if index["a"] >= index["b"] {
		t.Fatalf("expected a before b, got a=%d b=%d", index["a"], index["b"])
	}
	if index["b"] >= index["c"] {
		t.Fatalf("expected b before c, got b=%d c=%d", index["b"], index["c"])
	}
}

func TestSortTopologicalDiamondDeps(t *testing.T) {
	t.Parallel()
	// a -> b -> d  and  a -> c -> d
	a := &depPlugin{name: "a"}
	b := &depPlugin{name: "b", dependencies: []string{"a"}}
	c := &depPlugin{name: "c", dependencies: []string{"a"}}
	d := &depPlugin{name: "d", dependencies: []string{"b", "c"}}
	plugins := []Plugin{d, c, b, a}

	sorted, err := sortTopological(plugins)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 4 {
		t.Fatalf("expected 4 plugins, got %d", len(sorted))
	}
	index := make(map[string]int)
	for i, p := range sorted {
		index[p.Name()] = i
	}
	if index["a"] >= index["b"] || index["a"] >= index["c"] {
		t.Fatal("expected a before b and c")
	}
	if index["b"] >= index["d"] || index["c"] >= index["d"] {
		t.Fatal("expected b and c before d")
	}
}

func TestSortTopologicalMissingDep(t *testing.T) {
	t.Parallel()
	plugins := []Plugin{
		&depPlugin{name: "a", dependencies: []string{"nonexistent"}},
	}
	_, err := sortTopological(plugins)
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
}

func TestSortTopologicalCycle(t *testing.T) {
	t.Parallel()
	plugins := []Plugin{
		&depPlugin{name: "a", dependencies: []string{"b"}},
		&depPlugin{name: "b", dependencies: []string{"a"}},
	}
	_, err := sortTopological(plugins)
	if err == nil {
		t.Fatal("expected error for cycle")
	}
}

func TestSortTopologicalSelfDep(t *testing.T) {
	t.Parallel()
	plugins := []Plugin{
		&depPlugin{name: "a", dependencies: []string{"a"}},
	}
	_, err := sortTopological(plugins)
	if err == nil {
		t.Fatal("expected error for self-dependency cycle")
	}
}

func TestStartWithDepsOrder(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	a := &depPlugin{name: "a"}
	b := &depPlugin{name: "b", dependencies: []string{"a"}}

	_ = registry.Register(b)
	_ = registry.Register(a)

	err := registry.Start(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&a.startCalled) != 1 {
		t.Fatal("expected a to be started")
	}
	if atomic.LoadInt32(&b.startCalled) != 1 {
		t.Fatal("expected b to be started")
	}
}

func TestStopReverseDepsOrder(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	a := &depPlugin{name: "a"}
	b := &depPlugin{name: "b", dependencies: []string{"a"}}

	_ = registry.Register(a)
	_ = registry.Register(b)

	_ = registry.Start(context.Background())

	err := registry.Stop(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&a.stopCalled) != 1 {
		t.Fatal("expected a to be stopped")
	}
	if atomic.LoadInt32(&b.stopCalled) != 1 {
		t.Fatal("expected b to be stopped")
	}
}

func TestStartCycleError(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	_ = registry.Register(&depPlugin{name: "a", dependencies: []string{"b"}})
	_ = registry.Register(&depPlugin{name: "b", dependencies: []string{"a"}})

	err := registry.Start(context.Background())
	if err == nil {
		t.Fatal("expected error for dependency cycle")
	}
}

func TestStopFallbackOnCycle(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	a := &depPlugin{name: "a", dependencies: []string{"b"}}
	b := &depPlugin{name: "b", dependencies: []string{"a"}}

	_ = registry.Register(a)
	_ = registry.Register(b)

	// Stop should not fail even with cycle — it falls back to unsorted
	err := registry.Stop(context.Background())
	if err != nil {
		t.Fatalf("Stop should tolerate cycle, got: %v", err)
	}
}

func TestStartMissingDepError(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	_ = registry.Register(&depPlugin{name: "a", dependencies: []string{"missing"}})

	err := registry.Start(context.Background())
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
}

func TestStartWithDepsStartErrorStillReportsStartCalled(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	a := &depPlugin{name: "a"}
	b := &depPlugin{name: "b", dependencies: []string{"a"}, startErr: errors.New("boom")}

	_ = registry.Register(a)
	_ = registry.Register(b)

	err := registry.Start(context.Background())
	if err == nil {
		t.Fatal("expected start error")
	}
	if atomic.LoadInt32(&a.startCalled) != 1 {
		t.Fatal("expected a to have been started before b failed")
	}
}
