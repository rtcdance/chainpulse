package config

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"

	_ "github.com/lib/pq"
)

// PostgresConfig holds PostgreSQL configuration
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password core.SecretString
	Database string
	SSLMode  string
}

// PostgresCluster manages PostgreSQL cluster operations
type PostgresCluster struct {
	config *PostgresConfig
	DB     *sql.DB
}

// NewPostgresCluster creates a new PostgreSQL cluster manager
func NewPostgresCluster(cfg *PostgresConfig) (*PostgresCluster, error) {
	if cfg == nil {
		cfg = &PostgresConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Database: "postgres",
			SSLMode:  "require",
		}
	}

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password.Value(), cfg.Database, cfg.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresCluster{
		config: cfg,
		DB:     db,
	}, nil
}

// Health checks PostgreSQL cluster health
func (p *PostgresCluster) Health(ctx context.Context) error {
	return p.DB.PingContext(ctx)
}

// Close closes the PostgreSQL connection
func (p *PostgresCluster) Close() error {
	if p.DB != nil {
		return p.DB.Close()
	}
	return nil
}

// WaitForPostgres waits for PostgreSQL to be available
func WaitForPostgres(ctx context.Context, cfg *PostgresConfig, timeout time.Duration) error {
	cluster, err := NewPostgresCluster(cfg)
	if err != nil {
		return fmt.Errorf("failed to create postgres cluster: %w", err)
	}
	defer func() {
		if err := cluster.Close(); err != nil {
			_ = err // Log but continue
		}
	}()

	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for PostgreSQL")
		}

		healthCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := cluster.Health(healthCtx)
		cancel()

		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}
