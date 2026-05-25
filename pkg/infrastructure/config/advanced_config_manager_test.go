package config

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newTestConfigManager() *ConfigManager {
	return NewConfigManager(NewMockConsulClient(), generateTestEncryptionKey())
}

func TestNewConfigurationService(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)

	assert.NotNil(t, service)
	assert.Equal(t, cm, service.configManager)
	assert.NotNil(t, service.versionedCM)
	assert.NotNil(t, service.validators)
	assert.NotNil(t, service.updateHooks)
}

func TestConfigurationService_RegisterValidator(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)

	validator := func(key, value string) error { return nil }
	service.RegisterValidator("test-key", validator)

	assert.NotNil(t, service.validators["test-key"])
}

func TestConfigurationService_RegisterUpdateHook(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)

	hook := func(key, oldValue, newValue string) error { return nil }
	service.RegisterUpdateHook("test-key", hook)

	assert.Equal(t, 1, len(service.updateHooks["test-key"]))
}

func TestConfigurationService_GetConfig(t *testing.T) {
	t.Parallel()
	consul := NewMockConsulClient()
	_ = consul.SetConfig(context.Background(), "test-key", "test-value")
	cm := NewConfigManager(consul, generateTestEncryptionKey())
	service := NewConfigurationService(cm)

	val, err := service.GetConfig(context.Background(), "test-key")
	assert.NoError(t, err)
	assert.Equal(t, "test-value", val)
}

func TestConfigurationService_SetConfig(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)

	err := service.SetConfig(context.Background(), "cfg-key", "cfg-value", "author1")
	assert.NoError(t, err)

	val, err := service.GetConfig(context.Background(), "cfg-key")
	assert.NoError(t, err)
	assert.Equal(t, "cfg-value", val)
}

func TestConfigurationService_SetConfigWithValidator(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)

	service.RegisterValidator("validated-key", func(key, value string) error {
		if value == "bad" {
			return assert.AnError
		}
		return nil
	})

	err := service.SetConfig(context.Background(), "validated-key", "good", "author")
	assert.NoError(t, err)

	err = service.SetConfig(context.Background(), "validated-key", "bad", "author")
	assert.Error(t, err)
}

func TestConfigurationService_SetConfigWithHook(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)

	var hookCalled bool
	service.RegisterUpdateHook("hook-key", func(key, oldValue, newValue string) error {
		hookCalled = true
		assert.Equal(t, "", oldValue)
		assert.Equal(t, "new-val", newValue)
		return nil
	})

	err := service.SetConfig(context.Background(), "hook-key", "new-val", "author")
	assert.NoError(t, err)
	assert.True(t, hookCalled)
}

func TestConfigurationService_SetConfigHookError(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)

	service.RegisterUpdateHook("hook-key", func(key, oldValue, newValue string) error {
		return assert.AnError
	})

	err := service.SetConfig(context.Background(), "hook-key", "value", "author")
	assert.Error(t, err)
}

func TestConfigurationService_GetConfigWithDefault(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)

	val := service.GetConfigWithDefault(context.Background(), "nonexistent", "default-val")
	assert.Equal(t, "default-val", val)

	_ = service.SetConfig(context.Background(), "real-key", "real-val", "author")
	val = service.GetConfigWithDefault(context.Background(), "real-key", "default-val")
	assert.Equal(t, "real-val", val)
}

func TestConfigurationService_GetConfigInt(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)

	_ = service.SetConfig(context.Background(), "int-key", "42", "author")
	val, err := service.GetConfigInt(context.Background(), "int-key")
	assert.NoError(t, err)
	assert.Equal(t, 42, val)

	_, err = service.GetConfigInt(context.Background(), "nonexistent")
	assert.Error(t, err)

	_ = service.SetConfig(context.Background(), "bad-int", "not-a-number", "author")
	_, err = service.GetConfigInt(context.Background(), "bad-int")
	assert.Error(t, err)
}

func TestConfigurationService_GetConfigDuration(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)

	_ = service.SetConfig(context.Background(), "dur-key", "5s", "author")
	val, err := service.GetConfigDuration(context.Background(), "dur-key")
	assert.NoError(t, err)
	assert.Equal(t, 5*time.Second, val)

	_, err = service.GetConfigDuration(context.Background(), "nonexistent")
	assert.Error(t, err)

	_ = service.SetConfig(context.Background(), "bad-dur", "xyz", "author")
	_, err = service.GetConfigDuration(context.Background(), "bad-dur")
	assert.Error(t, err)
}

func TestConfigurationService_GetConfigBool(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)

	tests := []struct {
		value    string
		expected bool
		err      bool
	}{
		{"true", true, false},
		{"1", true, false},
		{"yes", true, false},
		{"on", true, false},
		{"false", false, false},
		{"0", false, false},
		{"no", false, false},
		{"off", false, false},
		{"invalid", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			svc := NewConfigurationService(newTestConfigManager())
			_ = svc.SetConfig(context.Background(), "bool-key", tt.value, "author")
			val, err := svc.GetConfigBool(context.Background(), "bool-key")
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, val)
			}
		})
	}

	_, err := service.GetConfigBool(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestConfigurationService_WatchConfig(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)

	err := service.WatchConfig(context.Background(), "watch-key", func(value string) {})
	assert.NoError(t, err)
}

func TestConfigurationService_GetConfigHistory(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)

	_ = service.SetConfig(context.Background(), "hist-key", "v1", "author1")
	_ = service.SetConfig(context.Background(), "hist-key", "v2", "author2")

	history, err := service.GetConfigHistory(context.Background(), "hist-key")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(history))

	_, err = service.GetConfigHistory(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestConfigurationService_RollbackConfig(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)

	_ = service.SetConfig(context.Background(), "rb-key", "v1", "author1")
	_ = service.SetConfig(context.Background(), "rb-key", "v2", "author2")

	err := service.RollbackConfig(context.Background(), "rb-key", 1, "rollback-author")
	assert.NoError(t, err)

	val, _ := service.GetConfig(context.Background(), "rb-key")
	assert.Equal(t, "v1", val)
}

func TestNewConfigurationBuilder(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)
	ctx := context.Background()

	builder := NewConfigurationBuilder(service, ctx, "test-author")

	assert.NotNil(t, builder)
	assert.Equal(t, service, builder.service)
	assert.Equal(t, ctx, builder.ctx)
	assert.Equal(t, "test-author", builder.author)
}

func TestConfigurationBuilder_Set(t *testing.T) {
	t.Parallel()
	builder := NewConfigurationBuilder(nil, context.Background(), "author")
	result := builder.Set("key1", "value1")

	assert.Equal(t, builder, result)
	assert.Equal(t, "value1", builder.configs["key1"])
}

func TestConfigurationBuilder_SetInt(t *testing.T) {
	t.Parallel()
	builder := NewConfigurationBuilder(nil, context.Background(), "author")
	result := builder.SetInt("int-key", 42)

	assert.Equal(t, builder, result)
	assert.Equal(t, "42", builder.configs["int-key"])
}

func TestConfigurationBuilder_SetDuration(t *testing.T) {
	t.Parallel()
	builder := NewConfigurationBuilder(nil, context.Background(), "author")
	result := builder.SetDuration("dur-key", 5*time.Second)

	assert.Equal(t, builder, result)
	assert.Equal(t, "5s", builder.configs["dur-key"])
}

func TestConfigurationBuilder_SetBool(t *testing.T) {
	t.Parallel()
	builder := NewConfigurationBuilder(nil, context.Background(), "author")
	result := builder.SetBool("bool-key", true)
	assert.Equal(t, "true", builder.configs["bool-key"])

	result = builder.SetBool("bool-key2", false)
	assert.Equal(t, "false", builder.configs["bool-key2"])
	_ = result
}

func TestConfigurationBuilder_Apply(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)
	ctx := context.Background()

	builder := NewConfigurationBuilder(service, ctx, "test-author").
		Set("k1", "v1").
		Set("k2", "v2")

	err := builder.Apply()
	assert.NoError(t, err)

	val1, _ := service.GetConfig(ctx, "k1")
	val2, _ := service.GetConfig(ctx, "k2")
	assert.Equal(t, "v1", val1)
	assert.Equal(t, "v2", val2)
}

func TestNewConfigurationSnapshotManager(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)
	csm := NewConfigurationSnapshotManager(service)

	assert.NotNil(t, csm)
	assert.Equal(t, service, csm.service)
	assert.NotNil(t, csm.snapshots)
}

func TestConfigurationSnapshotManager_CreateAndGetSnapshot(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)
	_ = service.SetConfig(context.Background(), "snap-k1", "snap-v1", "author")
	_ = service.SetConfig(context.Background(), "snap-k2", "snap-v2", "author")

	csm := NewConfigurationSnapshotManager(service)
	err := csm.CreateSnapshot(context.Background(), "my-snap", []string{"snap-k1", "snap-k2"})
	assert.NoError(t, err)

	snapshot, err := csm.GetSnapshot("my-snap")
	assert.NoError(t, err)
	assert.Equal(t, "snap-v1", snapshot.Configs["snap-k1"])
	assert.Equal(t, "snap-v2", snapshot.Configs["snap-k2"])

	_, err = csm.GetSnapshot("nonexistent")
	assert.Error(t, err)
}

func TestConfigurationSnapshotManager_RestoreSnapshot(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)
	_ = service.SetConfig(context.Background(), "restore-key", "original", "author")

	csm := NewConfigurationSnapshotManager(service)
	_ = csm.CreateSnapshot(context.Background(), "restore-snap", []string{"restore-key"})

	_ = service.SetConfig(context.Background(), "restore-key", "modified", "author")
	val, _ := service.GetConfig(context.Background(), "restore-key")
	assert.Equal(t, "modified", val)

	err := csm.RestoreSnapshot(context.Background(), "restore-snap", "restore-author")
	assert.NoError(t, err)

	val, _ = service.GetConfig(context.Background(), "restore-key")
	assert.Equal(t, "original", val)
}

func TestConfigurationSnapshotManager_ListSnapshots(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)
	_ = service.SetConfig(context.Background(), "k1", "v1", "author")

	csm := NewConfigurationSnapshotManager(service)
	_ = csm.CreateSnapshot(context.Background(), "snap1", []string{"k1"})
	_ = csm.CreateSnapshot(context.Background(), "snap2", []string{"k1"})

	names := csm.ListSnapshots()
	assert.Equal(t, 2, len(names))
}

func TestConfigurationSnapshotManager_DeleteSnapshot(t *testing.T) {
	t.Parallel()
	cm := newTestConfigManager()
	service := NewConfigurationService(cm)
	_ = service.SetConfig(context.Background(), "k1", "v1", "author")

	csm := NewConfigurationSnapshotManager(service)
	_ = csm.CreateSnapshot(context.Background(), "del-snap", []string{"k1"})

	csm.DeleteSnapshot("del-snap")
	_, err := csm.GetSnapshot("del-snap")
	assert.Error(t, err)
}
