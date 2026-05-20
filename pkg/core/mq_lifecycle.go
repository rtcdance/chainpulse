package core

import (
	"fmt"
	"time"
)

// Initialize initializes the plugin
func (p *BaseMQPlugin) Initialize() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isInitialized {
		return fmt.Errorf("plugin already initialized")
	}

	p.isInitialized = true
	p.logger.Info("message queue plugin initialized", "name", p.name, "version", p.version)

	return nil
}

// Start starts the plugin
func (p *BaseMQPlugin) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isInitialized {
		return fmt.Errorf("plugin not initialized")
	}

	if p.isRunning {
		return fmt.Errorf("plugin already running")
	}

	p.isRunning = true
	p.logger.Info("message queue plugin started", "name", p.name)

	return nil
}

// Stop stops the plugin
func (p *BaseMQPlugin) Stop() error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return nil
	}

	p.isRunning = false
	p.logger.Info("message queue plugin stopping", "name", p.name)
	p.mu.Unlock()

	// Wait for all in-flight operations to complete.
	// Must be done outside the lock to avoid deadlock: if an in-flight
	// operation tries to acquire p.mu (e.g., in PublishMessage), it would
	// deadlock if we held the lock during Wait().
	p.inFlightWaitGroup.Wait()
	p.logger.Info("message queue plugin stopped", "name", p.name)

	return nil
}

// Health returns the health status of the plugin
func (p *BaseMQPlugin) Health() *HealthStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	messageCount := p.messageCount.Load()
	errorCount := p.errorCount.Load()
	dlqSize := p.deadLetterQueueSize.Load()

	status := "healthy"
	if errorCount > 0 {
		status = "degraded"
	}

	return &HealthStatus{
		Status:    status,
		Timestamp: time.Now().UTC(),
		Details: map[string]any{
			"name":                   p.name,
			"version":                p.version,
			"is_running":             p.isRunning,
			"message_count":          messageCount,
			"error_count":            errorCount,
			"dead_letter_queue_size": dlqSize,
		},
	}
}
