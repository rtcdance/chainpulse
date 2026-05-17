package core

import (
	"fmt"
	"strings"
	"time"
)

// GetBlockchainConfig returns configuration for a specific blockchain
func (cm *DefaultConfigManager) GetBlockchainConfig(chainName string) (BlockchainConfig, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cfg, exists := cm.config.Blockchains[chainName]; exists {
		return cfg, nil
	}
	return BlockchainConfig{}, NewSystemError(
		ErrorTypePermanent,
		ErrorCodeNotFound,
		fmt.Sprintf("blockchain %s not configured", chainName),
		nil,
	)
}

// IsMultiChain returns true if multiple blockchains are configured
func (cm *DefaultConfigManager) IsMultiChain() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return len(cm.config.Blockchains) > 1
}

// GetActiveChains returns the list of active blockchain chains
func (cm *DefaultConfigManager) GetActiveChains() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	chains := make([]string, len(cm.config.ActiveChains))
	copy(chains, cm.config.ActiveChains)
	return chains
}

// GetAllBlockchainConfigs returns all blockchain configurations
func (cm *DefaultConfigManager) GetAllBlockchainConfigs() map[string]BlockchainConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	configs := make(map[string]BlockchainConfig)
	for k, v := range cm.config.Blockchains {
		configs[k] = v
	}
	return configs
}

// SetFeatureFlag sets a feature flag value
func (cm *DefaultConfigManager) SetFeatureFlag(flag string, enabled bool) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if flag == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			"feature flag name cannot be empty",
			nil,
		)
	}

	cm.config.FeatureFlags[flag] = enabled

	if cm.logger != nil {
		cm.logger.Info(
			"feature flag updated",
			"flag", flag,
			"enabled", enabled,
		)
	}

	return nil
}

// IsFeatureFlagEnabled checks if a feature flag is enabled
func (cm *DefaultConfigManager) IsFeatureFlagEnabled(flag string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.config.FeatureFlags[flag]
}

// GetFeatureFlags returns all feature flags
func (cm *DefaultConfigManager) GetFeatureFlags() map[string]bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	flags := make(map[string]bool)
	for k, v := range cm.config.FeatureFlags {
		flags[k] = v
	}
	return flags
}

// SetHotReloadEnabled enables or disables hot reload
func (cm *DefaultConfigManager) SetHotReloadEnabled(enabled bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.hotReloadEnabled = enabled

	if cm.logger != nil {
		cm.logger.Info(
			"hot reload configuration",
			"enabled", enabled,
		)
	}
}

// GetLastLoadTime returns the last time configuration was loaded
func (cm *DefaultConfigManager) GetLastLoadTime() time.Time {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.lastLoadTime
}

func parseFeatureFlags(flagsStr string) map[string]bool {
	flags := make(map[string]bool)
	if flagsStr == "" {
		return flags
	}

	parts := strings.Split(flagsStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.Split(part, "=")
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			flags[key] = value == "true" || value == "1"
		}
	}

	return flags
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
