package deployment

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
)

type testMQPlugin struct{}

func (t *testMQPlugin) Name() string { return "test-mq" }

func (t *testMQPlugin) Publish(ctx context.Context, topic string, message []byte) error { return nil }

func (t *testMQPlugin) Subscribe(ctx context.Context, topic string, handler func([]byte)) error {
	return nil
}

func (t *testMQPlugin) GetQueueDepth(ctx context.Context, topic string) (int64, error) { return 0, nil }

func (t *testMQPlugin) Initialize(ctx context.Context, config core.Config) error { return nil }

func (t *testMQPlugin) Start(ctx context.Context) error { return nil }

func (t *testMQPlugin) Stop(ctx context.Context) error { return nil }

func (t *testMQPlugin) Health(ctx context.Context) error { return nil }

type testCachePlugin struct{}

func (t *testCachePlugin) Name() string                                        { return "test-cache" }
func (t *testCachePlugin) Get(ctx context.Context, key string) ([]byte, error) { return nil, nil }
func (t *testCachePlugin) Set(ctx context.Context, key string, value []byte, ttl int) error {
	return nil
}
func (t *testCachePlugin) Delete(ctx context.Context, key string) error             { return nil }
func (t *testCachePlugin) GetStats() core.CacheStats                                { return core.CacheStats{} }
func (t *testCachePlugin) HealthCheck(ctx context.Context) error                    { return nil }
func (t *testCachePlugin) Initialize(ctx context.Context, config core.Config) error { return nil }
func (t *testCachePlugin) Start(ctx context.Context) error                          { return nil }
func (t *testCachePlugin) Stop(ctx context.Context) error                           { return nil }
func (t *testCachePlugin) Health(ctx context.Context) error                         { return nil }

type testDatabasePlugin struct{}

func (t *testDatabasePlugin) Name() string { return "test-db" }
func (t *testDatabasePlugin) GetEvent(ctx context.Context, id string) (*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (t *testDatabasePlugin) QueryEvents(ctx context.Context, filter any) ([]any, error) {
	return nil, nil
}

func (t *testDatabasePlugin) GetAllEvents(ctx context.Context) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (t *testDatabasePlugin) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (t *testDatabasePlugin) StoreEvent(ctx context.Context, event any) error { return nil }

func (t *testDatabasePlugin) BatchStoreEvents(ctx context.Context, events []any) error { return nil }

func (t *testDatabasePlugin) DeleteEvent(ctx context.Context, eventID string) error { return nil }

func (t *testDatabasePlugin) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return 0, nil
}

func (t *testDatabasePlugin) MarkEventsAsReorged(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return 0, nil
}

func (t *testDatabasePlugin) GetBlock(ctx context.Context, blockNumber uint64) (*blockchain.Block, error) {
	return nil, nil
}
func (t *testDatabasePlugin) GetLatestBlock(ctx context.Context) (uint64, error) { return 0, nil }
func (t *testDatabasePlugin) GetAllBlocks(ctx context.Context) ([]*blockchain.Block, error) {
	return nil, nil
}

func (t *testDatabasePlugin) GetReorgStats(ctx context.Context) (*core.ReorgStats, error) {
	return nil, nil
}

func (t *testDatabasePlugin) Initialize(ctx context.Context, config core.Config) error { return nil }

func (t *testDatabasePlugin) Start(ctx context.Context) error { return nil }

func (t *testDatabasePlugin) Stop(ctx context.Context) error { return nil }

func (t *testDatabasePlugin) Health(ctx context.Context) error { return nil }

func TestNewAdapterFactory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mode DeploymentMode
	}{
		{
			name: "MonolithicMode",
			mode: MonolithicMode,
		},
		{
			name: "MicroserviceMode",
			mode: MicroserviceMode,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := NewAdapterFactory(tt.mode)

			if f == nil {
				t.Fatal("expected non-nil AdapterFactory")
			}
			if f.mode != tt.mode {
				t.Errorf("mode = %s, want %s", f.mode, tt.mode)
			}
			if f.registry == nil {
				t.Error("registry should be initialized")
			}
			if len(f.registry) != 0 {
				t.Errorf("registry length = %d, want 0", len(f.registry))
			}
		})
	}
}

func TestAdapterFactory_RegisterFactory(t *testing.T) {
	t.Parallel()

	t.Run("register new factory", func(t *testing.T) {
		t.Parallel()

		f := NewAdapterFactory(MonolithicMode)
		factory := func(ctx context.Context) (any, error) { return "test_value", nil }

		f.RegisterFactory("mq:memory", factory)

		if len(f.registry) != 1 {
			t.Errorf("registry length = %d, want 1", len(f.registry))
		}
		if _, ok := f.registry["mq:memory"]; !ok {
			t.Error("expected factory registered under key mq:memory")
		}
	})

	t.Run("overwrite existing factory", func(t *testing.T) {
		t.Parallel()

		f := NewAdapterFactory(MonolithicMode)
		factory1 := func(ctx context.Context) (any, error) { return "value1", nil }
		factory2 := func(ctx context.Context) (any, error) { return "value2", nil }

		f.RegisterFactory("cache:inmemory", factory1)
		f.RegisterFactory("cache:inmemory", factory2)

		if len(f.registry) != 1 {
			t.Errorf("registry length = %d, want 1", len(f.registry))
		}

		val, err := f.registry["cache:inmemory"](context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "value2" {
			t.Errorf("value = %v, want value2", val)
		}
	})

	t.Run("register multiple factories", func(t *testing.T) {
		t.Parallel()

		f := NewAdapterFactory(MicroserviceMode)
		f.RegisterFactory("mq:memory", func(ctx context.Context) (any, error) { return nil, nil })
		f.RegisterFactory("cache:inmemory", func(ctx context.Context) (any, error) { return nil, nil })
		f.RegisterFactory("database:postgres", func(ctx context.Context) (any, error) { return nil, nil })

		if len(f.registry) != 3 {
			t.Errorf("registry length = %d, want 3", len(f.registry))
		}
	})
}

func TestGetDeploymentModeFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		setEnv   bool
		want     DeploymentMode
	}{
		{
			name:     "microservice env",
			envValue: "microservice",
			setEnv:   true,
			want:     MicroserviceMode,
		},
		{
			name:   "empty env",
			setEnv: false,
			want:   MonolithicMode,
		},
		{
			name:     "unknown env value",
			envValue: "something_else",
			setEnv:   true,
			want:     MonolithicMode,
		},
		{
			name:     "empty string env",
			envValue: "",
			setEnv:   true,
			want:     MonolithicMode,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("CHAINPULSE_DEPLOYMENT_MODE")
			if tt.setEnv {
				os.Setenv("CHAINPULSE_DEPLOYMENT_MODE", tt.envValue)
			}

			got := GetDeploymentModeFromEnv()

			if got != tt.want {
				t.Errorf("GetDeploymentModeFromEnv() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestAdapterFactory_CreateMQPlugin(t *testing.T) {
	t.Run("success with default type", func(t *testing.T) {
		f := NewAdapterFactory(MonolithicMode)
		f.RegisterFactory("mq:memory", func(ctx context.Context) (any, error) {
			return &testMQPlugin{}, nil
		})

		plugin, err := f.CreateMQPlugin(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plugin.Name() != "test-mq" {
			t.Errorf("Name() = %s, want test-mq", plugin.Name())
		}
	})

	t.Run("success with env type", func(t *testing.T) {
		os.Setenv("CHAINPULSE_MQ_TYPE", "kafka")
		defer os.Unsetenv("CHAINPULSE_MQ_TYPE")

		f := NewAdapterFactory(MonolithicMode)
		f.RegisterFactory("mq:kafka", func(ctx context.Context) (any, error) {
			return &testMQPlugin{}, nil
		})

		plugin, err := f.CreateMQPlugin(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plugin == nil {
			t.Fatal("expected non-nil plugin")
		}
	})

	t.Run("factory returns error", func(t *testing.T) {
		f := NewAdapterFactory(MonolithicMode)
		f.RegisterFactory("mq:memory", func(ctx context.Context) (any, error) {
			return nil, errors.New("mq connection failed")
		})

		plugin, err := f.CreateMQPlugin(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if plugin != nil {
			t.Errorf("plugin = %v, want nil", plugin)
		}
	})

	t.Run("type assertion failure", func(t *testing.T) {
		f := NewAdapterFactory(MonolithicMode)
		f.RegisterFactory("mq:memory", func(ctx context.Context) (any, error) {
			return "not_a_plugin", nil
		})

		plugin, err := f.CreateMQPlugin(context.Background())
		if err == nil {
			t.Fatal("expected error for type assertion failure")
		}
		if plugin != nil {
			t.Errorf("plugin = %v, want nil", plugin)
		}
	})
}

func TestAdapterFactory_CreateCachePlugin(t *testing.T) {
	t.Run("success with default type", func(t *testing.T) {
		f := NewAdapterFactory(MonolithicMode)
		f.RegisterFactory("cache:inmemory", func(ctx context.Context) (any, error) {
			return &testCachePlugin{}, nil
		})

		plugin, err := f.CreateCachePlugin(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plugin.Name() != "test-cache" {
			t.Errorf("Name() = %s, want test-cache", plugin.Name())
		}
	})

	t.Run("type assertion failure", func(t *testing.T) {
		f := NewAdapterFactory(MonolithicMode)
		f.RegisterFactory("cache:inmemory", func(ctx context.Context) (any, error) {
			return 42, nil
		})

		plugin, err := f.CreateCachePlugin(context.Background())
		if err == nil {
			t.Fatal("expected error for type assertion failure")
		}
		if plugin != nil {
			t.Errorf("plugin = %v, want nil", plugin)
		}
	})
}

func TestAdapterFactory_CreateDatabasePlugin(t *testing.T) {
	t.Run("success with default type", func(t *testing.T) {
		f := NewAdapterFactory(MonolithicMode)
		f.RegisterFactory("database:postgres", func(ctx context.Context) (any, error) {
			return &testDatabasePlugin{}, nil
		})

		plugin, err := f.CreateDatabasePlugin(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plugin.Name() != "test-db" {
			t.Errorf("Name() = %s, want test-db", plugin.Name())
		}
	})

	t.Run("type assertion failure", func(t *testing.T) {
		f := NewAdapterFactory(MonolithicMode)
		f.RegisterFactory("database:postgres", func(ctx context.Context) (any, error) {
			return struct{}{}, nil
		})

		plugin, err := f.CreateDatabasePlugin(context.Background())
		if err == nil {
			t.Fatal("expected error for type assertion failure")
		}
		if plugin != nil {
			t.Errorf("plugin = %v, want nil", plugin)
		}
	})
}

func TestRegisterDefaultFactories(t *testing.T) {
	f := NewAdapterFactory(MonolithicMode)
	RegisterDefaultFactories(f)

	t.Run("mq memory factory", func(t *testing.T) {
		f.mu.RLock()
		factory, ok := f.registry["mq:memory"]
		f.mu.RUnlock()
		if !ok {
			t.Fatal("expected mq:memory factory to be registered")
		}
		plugin, err := factory(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plugin == nil {
			t.Fatal("expected non-nil plugin")
		}
	})

	t.Run("cache inmemory factory", func(t *testing.T) {
		f.mu.RLock()
		factory, ok := f.registry["cache:inmemory"]
		f.mu.RUnlock()
		if !ok {
			t.Fatal("expected cache:inmemory factory to be registered")
		}
		plugin, err := factory(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plugin == nil {
			t.Fatal("expected non-nil plugin")
		}
	})

	t.Run("database mock factory", func(t *testing.T) {
		f.mu.RLock()
		factory, ok := f.registry["database:mock"]
		f.mu.RUnlock()
		if !ok {
			t.Fatal("expected database:mock factory to be registered")
		}
		plugin, err := factory(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plugin == nil {
			t.Fatal("expected non-nil plugin")
		}
	})
}

func TestAdapterFactory_createPlugin(t *testing.T) {
	t.Parallel()

	t.Run("factory exists", func(t *testing.T) {
		t.Parallel()

		f := NewAdapterFactory(MonolithicMode)
		f.RegisterFactory("mq:memory", func(ctx context.Context) (any, error) {
			return "mq_plugin", nil
		})

		plugin, err := f.createPlugin(context.Background(), "mq", "memory")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plugin != "mq_plugin" {
			t.Errorf("plugin = %v, want mq_plugin", plugin)
		}
	})

	t.Run("no factory registered for key", func(t *testing.T) {
		t.Parallel()

		f := NewAdapterFactory(MicroserviceMode)

		plugin, err := f.createPlugin(context.Background(), "cache", "redis")
		if err == nil {
			t.Error("expected error, got nil")
		}
		if plugin != nil {
			t.Errorf("plugin = %v, want nil", plugin)
		}
	})

	t.Run("factory returns error", func(t *testing.T) {
		t.Parallel()

		f := NewAdapterFactory(MonolithicMode)
		f.RegisterFactory("database:postgres", func(ctx context.Context) (any, error) {
			return nil, errors.New("connection refused")
		})

		plugin, err := f.createPlugin(context.Background(), "database", "postgres")
		if err == nil {
			t.Error("expected error, got nil")
		}
		if plugin != nil {
			t.Errorf("plugin = %v, want nil", plugin)
		}
	})
}
