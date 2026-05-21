package config

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ConfigurationService provides advanced configuration management
type ConfigurationService struct {
	configManager  *ConfigManager
	versionedCM    *VersionedConfigManager
	validators     map[string]ConfigValidator
	validatorMutex sync.RWMutex
	updateHooks    map[string][]ConfigUpdateHook
	hookMutex      sync.RWMutex
}

// ConfigValidator validates configuration values.
//
// Renaming would break many external uses.
type ConfigValidator func(key, value string) error

// ConfigUpdateHook is called when configuration is updated.
//
// Renaming would break many external uses.
type ConfigUpdateHook func(key, oldValue, newValue string) error

// NewConfigurationService creates a new configuration service
func NewConfigurationService(cm *ConfigManager) *ConfigurationService {
	return &ConfigurationService{
		configManager: cm,
		versionedCM:   NewVersionedConfigManager(cm),
		validators:    make(map[string]ConfigValidator),
		updateHooks:   make(map[string][]ConfigUpdateHook),
	}
}

// RegisterValidator registers a validator for a configuration key
func (cs *ConfigurationService) RegisterValidator(key string, validator ConfigValidator) {
	cs.validatorMutex.Lock()
	defer cs.validatorMutex.Unlock()
	cs.validators[key] = validator
}

// RegisterUpdateHook registers an update hook for a configuration key
func (cs *ConfigurationService) RegisterUpdateHook(key string, hook ConfigUpdateHook) {
	cs.hookMutex.Lock()
	defer cs.hookMutex.Unlock()
	cs.updateHooks[key] = append(cs.updateHooks[key], hook)
}

// GetConfig retrieves a configuration value with validation
func (cs *ConfigurationService) GetConfig(ctx context.Context, key string) (string, error) {
	return cs.configManager.GetConfig(ctx, key)
}

// SetConfig sets a configuration value with validation and hooks
func (cs *ConfigurationService) SetConfig(ctx context.Context, key, value, author string) error {
	// Get old value
	oldValue, _ := cs.configManager.GetConfig(ctx, key)

	// Validate new value
	cs.validatorMutex.RLock()
	validator := cs.validators[key]
	cs.validatorMutex.RUnlock()

	if validator != nil {
		if err := validator(key, value); err != nil {
			return fmt.Errorf("validation failed for key %s: %w", key, err)
		}
	}

	// Set configuration with versioning
	if err := cs.versionedCM.SetConfigWithVersion(ctx, key, value, author); err != nil {
		return err
	}

	// Call update hooks
	cs.hookMutex.RLock()
	hooks := cs.updateHooks[key]
	cs.hookMutex.RUnlock()

	for _, hook := range hooks {
		if err := hook(key, oldValue, value); err != nil {
			return fmt.Errorf("update hook failed for key %s: %w", key, err)
		}
	}

	return nil
}

// GetConfigWithDefault retrieves a configuration value with a default
func (cs *ConfigurationService) GetConfigWithDefault(ctx context.Context, key, defaultValue string) string {
	val, err := cs.configManager.GetConfig(ctx, key)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetConfigInt retrieves an integer configuration value
func (cs *ConfigurationService) GetConfigInt(ctx context.Context, key string) (int, error) {
	val, err := cs.configManager.GetConfig(ctx, key)
	if err != nil {
		return 0, err
	}

	var intVal int
	if _, err := fmt.Sscanf(val, "%d", &intVal); err != nil {
		return 0, fmt.Errorf("failed to parse integer config %s: %w", key, err)
	}

	return intVal, nil
}

// GetConfigDuration retrieves a duration configuration value
func (cs *ConfigurationService) GetConfigDuration(ctx context.Context, key string) (time.Duration, error) {
	val, err := cs.configManager.GetConfig(ctx, key)
	if err != nil {
		return 0, err
	}

	duration, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration config %s: %w", key, err)
	}

	return duration, nil
}

// GetConfigBool retrieves a boolean configuration value
func (cs *ConfigurationService) GetConfigBool(ctx context.Context, key string) (bool, error) {
	val, err := cs.configManager.GetConfig(ctx, key)
	if err != nil {
		return false, err
	}

	switch val {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value for config %s: %s", key, val)
	}
}

// WatchConfig watches for configuration changes
func (cs *ConfigurationService) WatchConfig(ctx context.Context, key string, handler func(string)) error {
	return cs.configManager.WatchConfig(ctx, key, handler)
}

// GetConfigHistory retrieves the history of a configuration
func (cs *ConfigurationService) GetConfigHistory(ctx context.Context, key string) ([]*ConfigVersion, error) {
	return cs.versionedCM.GetConfigHistory(key)
}

// RollbackConfig rolls back to a previous version
func (cs *ConfigurationService) RollbackConfig(ctx context.Context, key string, version int, author string) error {
	return cs.versionedCM.RollbackConfig(ctx, key, version, author)
}

// ConfigurationBuilder provides a fluent interface for configuration
type ConfigurationBuilder struct {
	service *ConfigurationService
	ctx     context.Context
	configs map[string]string
	author  string
}

// NewConfigurationBuilder creates a new configuration builder
func NewConfigurationBuilder(service *ConfigurationService, ctx context.Context, author string) *ConfigurationBuilder { //nolint:revive // ctx cannot be first; service is the primary receiver-like param
	return &ConfigurationBuilder{
		service: service,
		ctx:     ctx,
		configs: make(map[string]string),
		author:  author,
	}
}

// Set sets a configuration value
func (cb *ConfigurationBuilder) Set(key, value string) *ConfigurationBuilder {
	cb.configs[key] = value
	return cb
}

// SetInt sets an integer configuration value
func (cb *ConfigurationBuilder) SetInt(key string, value int) *ConfigurationBuilder {
	cb.configs[key] = fmt.Sprintf("%d", value)
	return cb
}

// SetDuration sets a duration configuration value
func (cb *ConfigurationBuilder) SetDuration(key string, value time.Duration) *ConfigurationBuilder {
	cb.configs[key] = value.String()
	return cb
}

// SetBool sets a boolean configuration value
func (cb *ConfigurationBuilder) SetBool(key string, value bool) *ConfigurationBuilder {
	if value {
		cb.configs[key] = "true"
	} else {
		cb.configs[key] = "false"
	}
	return cb
}

// Apply applies all configuration changes
func (cb *ConfigurationBuilder) Apply() error {
	for key, value := range cb.configs {
		if err := cb.service.SetConfig(cb.ctx, key, value, cb.author); err != nil {
			return err
		}
	}
	return nil
}

// ConfigurationSnapshot represents a snapshot of configuration at a point in time
type ConfigurationSnapshot struct {
	Timestamp time.Time
	Configs   map[string]string
}

// ConfigurationSnapshotManager manages configuration snapshots
type ConfigurationSnapshotManager struct {
	service   *ConfigurationService
	snapshots map[string]*ConfigurationSnapshot
	mutex     sync.RWMutex
}

// NewConfigurationSnapshotManager creates a new snapshot manager
func NewConfigurationSnapshotManager(service *ConfigurationService) *ConfigurationSnapshotManager {
	return &ConfigurationSnapshotManager{
		service:   service,
		snapshots: make(map[string]*ConfigurationSnapshot),
	}
}

// CreateSnapshot creates a configuration snapshot
func (csm *ConfigurationSnapshotManager) CreateSnapshot(ctx context.Context, name string, keys []string) error {
	snapshot := &ConfigurationSnapshot{
		Timestamp: time.Now(),
		Configs:   make(map[string]string),
	}

	for _, key := range keys {
		val, err := csm.service.GetConfig(ctx, key)
		if err != nil {
			return fmt.Errorf("failed to get config %s: %w", key, err)
		}
		snapshot.Configs[key] = val
	}

	csm.mutex.Lock()
	defer csm.mutex.Unlock()
	csm.snapshots[name] = snapshot

	return nil
}

// GetSnapshot retrieves a configuration snapshot
func (csm *ConfigurationSnapshotManager) GetSnapshot(name string) (*ConfigurationSnapshot, error) {
	csm.mutex.RLock()
	defer csm.mutex.RUnlock()

	snapshot, exists := csm.snapshots[name]
	if !exists {
		return nil, fmt.Errorf("snapshot not found: %s", name)
	}

	return snapshot, nil
}

// RestoreSnapshot restores a configuration snapshot
func (csm *ConfigurationSnapshotManager) RestoreSnapshot(ctx context.Context, name, author string) error {
	snapshot, err := csm.GetSnapshot(name)
	if err != nil {
		return err
	}

	for key, value := range snapshot.Configs {
		if err := csm.service.SetConfig(ctx, key, value, author); err != nil {
			return fmt.Errorf("failed to restore config %s: %w", key, err)
		}
	}

	return nil
}

// ListSnapshots lists all configuration snapshots
func (csm *ConfigurationSnapshotManager) ListSnapshots() []string {
	csm.mutex.RLock()
	defer csm.mutex.RUnlock()

	names := make([]string, 0, len(csm.snapshots))
	for name := range csm.snapshots {
		names = append(names, name)
	}

	return names
}

// DeleteSnapshot deletes a configuration snapshot
func (csm *ConfigurationSnapshotManager) DeleteSnapshot(name string) error {
	csm.mutex.Lock()
	defer csm.mutex.Unlock()

	if _, exists := csm.snapshots[name]; !exists {
		return fmt.Errorf("snapshot not found: %s", name)
	}

	delete(csm.snapshots, name)
	return nil
}
