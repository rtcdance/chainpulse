package ports

import (
	"context"
	"time"
)

// HealthChecker checks system health
type HealthChecker interface {
	Check(ctx context.Context) (HealthStatus, error)
}

// HealthStatus represents system health
type HealthStatus struct {
	Status    string         `json:"status"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details"`
	Timestamp time.Time      `json:"timestamp"`
}
