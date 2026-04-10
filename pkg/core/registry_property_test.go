package core

import (
	"fmt"
	"testing"
)

// Property 1: Plugin Registry Consistency
// For any sequence of plugin registrations and unregistrations,
// the registry count should accurately reflect the number of registered plugins

// TestPropertyRegistryConsistency verifies registry consistency
func TestPropertyRegistryConsistency(t *testing.T) {
	tests := []struct {
		name       string
		operations []string // "register" or "unregister"
		expected   int
	}{
		{
			name:       "empty registry",
			operations: []string{},
			expected:   0,
		},
		{
			name:       "single registration",
			operations: []string{"register"},
			expected:   1,
		},
		{
			name:       "multiple registrations",
			operations: []string{"register", "register", "register"},
			expected:   3,
		},
		{
			name:       "register and unregister",
			operations: []string{"register", "unregister"},
			expected:   0,
		},
		{
			name:       "multiple register and unregister",
			operations: []string{"register", "register", "unregister", "register"},
			expected:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewPluginRegistry(nil)
			pluginIndex := 0

			for _, op := range tt.operations {
				if op == "register" {
					plugin := &mockPlugin{
						name:    fmt.Sprintf("plugin-%d", pluginIndex),
						version: "1.0.0",
					}
					if err := registry.Register(plugin); err != nil {
						t.Logf("failed to register plugin: %v", err)
					}
					pluginIndex++
				} else if op == "unregister" && registry.Count() > 0 {
					plugins := registry.List()
					if len(plugins) > 0 {
						if err := registry.Unregister(plugins[0].Name()); err != nil {
							t.Logf("failed to unregister plugin: %v", err)
						}
					}
				}
			}

			if registry.Count() != tt.expected {
				t.Errorf("expected %d plugins, got %d", tt.expected, registry.Count())
			}
		})
	}
}

// TestPropertyRegistryRetrievalAfterRegistration verifies that registered plugins can be retrieved
func TestPropertyRegistryRetrievalAfterRegistration(t *testing.T) {
	registry := NewPluginRegistry(nil)

	// For any plugin that is registered, it should be retrievable
	for i := 0; i < 5; i++ {
		plugin := &mockPlugin{
			name:    fmt.Sprintf("plugin-%d", i),
			version: "1.0.0",
		}

		err := registry.Register(plugin)
		if err != nil {
			t.Fatalf("failed to register plugin: %v", err)
		}

		retrieved, err := registry.Get(plugin.Name())
		if err != nil {
			t.Errorf("expected to retrieve plugin %s, got error %v", plugin.Name(), err)
		}

		if retrieved.Name() != plugin.Name() {
			t.Errorf("expected plugin name %s, got %s", plugin.Name(), retrieved.Name())
		}
	}
}

// TestPropertyRegistryNotRetrievableAfterUnregistration verifies that unregistered plugins cannot be retrieved
func TestPropertyRegistryNotRetrievableAfterUnregistration(t *testing.T) {
	registry := NewPluginRegistry(nil)

	// For any plugin that is unregistered, it should not be retrievable
	for i := 0; i < 5; i++ {
		plugin := &mockPlugin{
			name:    fmt.Sprintf("plugin-%d", i),
			version: "1.0.0",
		}

		err := registry.Register(plugin)
		if err != nil {
			t.Fatalf("failed to register plugin: %v", err)
		}
		if err := registry.Unregister(plugin.Name()); err != nil {
			t.Fatalf("failed to unregister plugin: %v", err)
		}

		_, err2 := registry.Get(plugin.Name())
		if err2 == nil {
			t.Errorf("expected error when retrieving unregistered plugin %s", plugin.Name())
		}
	}
}

// TestPropertyRegistryListConsistency verifies that List() returns all registered plugins
func TestPropertyRegistryListConsistency(t *testing.T) {
	registry := NewPluginRegistry(nil)

	// For any set of registered plugins, List() should return all of them
	pluginNames := []string{"plugin-1", "plugin-2", "plugin-3"}
	for _, name := range pluginNames {
		plugin := &mockPlugin{name: name, version: "1.0.0"}
		err := registry.Register(plugin)
		if err != nil {
			t.Fatalf("failed to register plugin: %v", err)
		}
	}

	plugins := registry.List()
	if len(plugins) != len(pluginNames) {
		t.Errorf("expected %d plugins in list, got %d", len(pluginNames), len(plugins))
	}

	// Verify all registered plugins are in the list
	for _, name := range pluginNames {
		found := false
		for _, plugin := range plugins {
			if plugin.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected plugin %s in list", name)
		}
	}
}

// TestPropertyRegistryStartStopConsistency verifies that Start/Stop work correctly
func TestPropertyRegistryStartStopConsistency(t *testing.T) {
	registry := NewPluginRegistry(nil)

	// For any set of registered plugins, Start() and Stop() should succeed
	for i := 0; i < 3; i++ {
		plugin := &mockPlugin{
			name:    fmt.Sprintf("plugin-%d", i),
			version: "1.0.0",
		}
		if err := registry.Register(plugin); err != nil {
			t.Fatalf("failed to register plugin %d: %v", i, err)
		}
	}

	err := registry.Start()
	if err != nil {
		t.Errorf("expected Start() to succeed, got error %v", err)
	}

	err = registry.Stop()
	if err != nil {
		t.Errorf("expected Stop() to succeed, got error %v", err)
	}
}

// TestPropertyRegistryClearConsistency verifies that Clear() removes all plugins
func TestPropertyRegistryClearConsistency(t *testing.T) {
	registry := NewPluginRegistry(nil)

	// For any set of registered plugins, Clear() should remove all of them
	for i := 0; i < 5; i++ {
		plugin := &mockPlugin{
			name:    fmt.Sprintf("plugin-%d", i),
			version: "1.0.0",
		}
		if err := registry.Register(plugin); err != nil {
			t.Fatalf("failed to register plugin %d: %v", i, err)
		}
	}

	if registry.Count() != 5 {
		t.Errorf("expected 5 plugins before clear")
	}

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("expected 0 plugins after clear, got %d", registry.Count())
	}

	// Verify no plugins can be retrieved
	_, err := registry.Get("plugin-0")
	if err == nil {
		t.Errorf("expected error when retrieving plugin after clear")
	}
}

// TestPropertyRegistryIdempotentUnregister verifies that unregistering a non-existent plugin fails consistently
func TestPropertyRegistryIdempotentUnregister(t *testing.T) {
	registry := NewPluginRegistry(nil)

	// For any non-existent plugin, unregistering should always fail
	for i := 0; i < 3; i++ {
		err := registry.Unregister("nonexistent-plugin")
		if err == nil {
			t.Errorf("expected error when unregistering nonexistent plugin")
		}
	}
}

// TestPropertyRegistryDuplicateRegistrationFails verifies that duplicate registration always fails
func TestPropertyRegistryDuplicateRegistrationFails(t *testing.T) {
	registry := NewPluginRegistry(nil)
	plugin := &mockPlugin{name: "test-plugin", version: "1.0.0"}

	// First registration should succeed
	err := registry.Register(plugin)
	if err != nil {
		t.Errorf("expected first registration to succeed, got error %v", err)
	}

	// Subsequent registrations with same name should fail
	for i := 0; i < 3; i++ {
		plugin2 := &mockPlugin{name: "test-plugin", version: "2.0.0"}
		err := registry.Register(plugin2)
		if err == nil {
			t.Errorf("expected duplicate registration to fail")
		}
	}
}
