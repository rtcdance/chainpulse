package cache

import (
	"testing"
)

func TestNewDistributedCacheConfig(t *testing.T) {
	cfg := NewDistributedCacheConfig()
	if cfg == nil {
		t.Fatal("NewDistributedCacheConfig returned nil")
	}
	if cfg.RedisAddr == "" {
		t.Error("RedisAddr should have default value")
	}
	if cfg.PoolSize < 1 {
		t.Error("PoolSize should have default value")
	}
	if cfg.DefaultTTL == 0 {
		t.Error("DefaultTTL should have default value")
	}
}

func TestDistributedCacheConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *DistributedCacheConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &DistributedCacheConfig{
				RedisAddr:         "localhost:6379",
				PoolSize:          10,
				MaxLocalCacheSize: 1000,
			},
			wantErr: false,
		},
		{
			name: "empty RedisAddr",
			cfg: &DistributedCacheConfig{
				RedisAddr:         "",
				PoolSize:          10,
				MaxLocalCacheSize: 1000,
			},
			wantErr: true,
		},
		{
			name: "PoolSize zero",
			cfg: &DistributedCacheConfig{
				RedisAddr:         "localhost:6379",
				PoolSize:          0,
				MaxLocalCacheSize: 1000,
			},
			wantErr: true,
		},
		{
			name: "MaxLocalCacheSize zero",
			cfg: &DistributedCacheConfig{
				RedisAddr:         "localhost:6379",
				PoolSize:          10,
				MaxLocalCacheSize: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
