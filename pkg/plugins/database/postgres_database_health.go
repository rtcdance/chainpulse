package database

import (
	"context"
	"sync"
	"time"
)

const recoveryTimeout = 30 * time.Second

// HealthChecker manages database health checks
type HealthChecker struct {
	db                      *PostgreSQLDatabase
	ticker                  *time.Ticker
	stopChan                chan bool
	mu                      sync.RWMutex
	lastHealthCheck         time.Time
	isHealthy               bool
	consecutiveErrors       int
	maxConsecutiveErrors    int
	recoveryAttempts        int
	maxRecoveryAttempts     int
	lastRecoveryAttemptTime time.Time
	recoveryAttemptInterval time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *PostgreSQLDatabase) *HealthChecker {
	return &HealthChecker{
		db:                      db,
		stopChan:                make(chan bool),
		isHealthy:               true,
		consecutiveErrors:       0,
		maxConsecutiveErrors:    3,
		recoveryAttempts:        0,
		maxRecoveryAttempts:     5,
		recoveryAttemptInterval: 5 * time.Second,
	}
}

// Start starts the health checker
func (hc *HealthChecker) Start(interval time.Duration) {
	hc.ticker = time.NewTicker(interval)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				hc.db.logger.Error("goroutine panic recovered", "panic", r)
			}
		}()
		for {
			select {
			case <-hc.stopChan:
				hc.ticker.Stop()
				return
			case <-hc.ticker.C:
				hc.check()
			}
		}
	}()
}

// Stop stops the health checker
func (hc *HealthChecker) Stop() {
	select {
	case hc.stopChan <- true:
	default:
	}
}

// check performs a health check
func (hc *HealthChecker) check() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := hc.db.db.PingContext(ctx)

	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.lastHealthCheck = time.Now()

	if err != nil {
		hc.consecutiveErrors++
		hc.db.logger.Warn("Health check failed", "error", err.Error(), "consecutive_errors", hc.consecutiveErrors)

		if hc.consecutiveErrors >= hc.maxConsecutiveErrors {
			hc.isHealthy = false
			hc.db.logger.Error("Database marked unhealthy", "consecutive_errors", hc.consecutiveErrors)

			// Attempt recovery if not already recovering
			if time.Since(hc.lastRecoveryAttemptTime) > hc.recoveryAttemptInterval {
				go hc.attemptRecovery()
			}
		}
	} else {
		hc.consecutiveErrors = 0
		if !hc.isHealthy {
			hc.isHealthy = true
			hc.recoveryAttempts = 0
			hc.db.logger.Info("Database recovered")
		}
	}
}

// attemptRecovery attempts to recover the database connection
func (hc *HealthChecker) attemptRecovery() {
	hc.mu.Lock()
	hc.lastRecoveryAttemptTime = time.Now()
	hc.recoveryAttempts++
	attempts := hc.recoveryAttempts
	hc.mu.Unlock()

	hc.db.logger.Info("Attempting database recovery", "attempt", attempts)

	// Close existing connection
	if hc.db.db != nil {
		_ = hc.db.db.Close()
	}

	// Reinitialize
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := hc.db.Initialize(ctx, *hc.db.config)
	if err != nil {
		hc.db.logger.Error("Recovery failed", "error", err.Error(), "attempt", attempts)

		if attempts >= hc.maxRecoveryAttempts {
			hc.mu.Lock()
			hc.isHealthy = false
			hc.mu.Unlock()
			hc.db.logger.Error("Max recovery attempts exceeded", "attempts", attempts)
		}
		return
	}

	hc.db.logger.Info("Database recovery successful", "attempt", attempts)

	hc.mu.Lock()
	hc.isHealthy = true
	hc.consecutiveErrors = 0
	hc.recoveryAttempts = 0
	hc.mu.Unlock()
}

// IsHealthy returns whether the database is healthy
func (hc *HealthChecker) IsHealthy() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.isHealthy
}

// GetStatus returns the health status
func (hc *HealthChecker) GetStatus() map[string]any {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	return map[string]any{
		"is_healthy":             hc.isHealthy,
		"last_health_check":      hc.lastHealthCheck,
		"consecutive_errors":     hc.consecutiveErrors,
		"max_consecutive_errors": hc.maxConsecutiveErrors,
		"recovery_attempts":      hc.recoveryAttempts,
		"max_recovery_attempts":  hc.maxRecoveryAttempts,
	}
}

// SetMaxConsecutiveErrors sets the max consecutive errors before marking unhealthy
func (hc *HealthChecker) SetMaxConsecutiveErrors(max int) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.maxConsecutiveErrors = max
}

// SetMaxRecoveryAttempts sets the max recovery attempts
func (hc *HealthChecker) SetMaxRecoveryAttempts(max int) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.maxRecoveryAttempts = max
}

// SetRecoveryAttemptInterval sets the interval between recovery attempts
func (hc *HealthChecker) SetRecoveryAttemptInterval(interval time.Duration) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.recoveryAttemptInterval = interval
}
