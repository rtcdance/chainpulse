package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// APIKeyRecord represents a persisted API key
type APIKeyRecord struct {
	ID           string   `json:"id"`
	ClientID     string   `json:"clientId"`
	Name         string   `json:"name"`
	KeyHash      string   `json:"-"`
	KeyPrefix    string   `json:"keyPrefix"`
	Permissions  []string `json:"permissions"`
	Enabled      bool     `json:"enabled"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
	ExpiresAt    *string  `json:"expiresAt,omitempty"`
	LastUsedAt   *string  `json:"lastUsedAt,omitempty"`
	RequestCount int64    `json:"requestCount"`
}

// APIKeyStore manages persistent API key storage
type APIKeyStore struct {
	db       *sql.DB
	logger   core.Logger
	metrics  core.MetricsCollector
	wg       sync.WaitGroup
	done     chan struct{}
	stopOnce sync.Once
}

func NewAPIKeyStore(db *sql.DB, logger core.Logger, metrics core.MetricsCollector) *APIKeyStore {
	return &APIKeyStore{
		db:      db,
		logger:  logger,
		metrics: metrics,
		done:    make(chan struct{}),
	}
}

func (s *APIKeyStore) Stop() {
	s.stopOnce.Do(func() {
		close(s.done)
	})
	s.wg.Wait()
}

func (s *APIKeyStore) CreateAPIKey(ctx context.Context, clientID, name string, permissions []string, expiresAt *time.Time) (*APIKeyRecord, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", fmt.Errorf("failed to generate key: %w", err)
	}
	plainKey := "cp_" + hex.EncodeToString(raw)
	prefix := plainKey[:8]

	hash := sha256.Sum256([]byte(plainKey))
	keyHash := hex.EncodeToString(hash[:])

	id := fmt.Sprintf("ak_%s", hex.EncodeToString(raw[:16]))
	permsJSON, _ := json.Marshal(permissions)

	var expiresAtVal any
	if expiresAt != nil {
		expiresAtVal = expiresAt.UTC()
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO api_keys (id, client_id, name, key_hash, key_prefix, permissions, enabled, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, true, $7)`,
		id, clientID, name, keyHash, prefix, string(permsJSON), expiresAtVal,
	)
	if err != nil {
		s.metrics.RecordCounter("api_keys.create_failed", 1, nil)
		return nil, "", fmt.Errorf("failed to store key: %w", err)
	}

	s.metrics.RecordCounter("api_keys.created", 1, nil)

	record := &APIKeyRecord{
		ID:          id,
		ClientID:    clientID,
		Name:        name,
		KeyPrefix:   prefix,
		Permissions: permissions,
		Enabled:     true,
	}
	return record, plainKey, nil
}

func (s *APIKeyStore) ValidateAPIKey(ctx context.Context, plainKey string) (*APIKeyRecord, error) {
	hash := sha256.Sum256([]byte(plainKey))
	keyHash := hex.EncodeToString(hash[:])

	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, client_id, name, key_prefix, permissions, enabled, expires_at
		 FROM api_keys WHERE key_hash = $1`,
		keyHash,
	)

	var id, clientID, name, prefix string
	var permsJSON string
	var enabled bool
	var expiresAt *time.Time

	if err := row.Scan(&id, &clientID, &name, &prefix, &permsJSON, &enabled, &expiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.metrics.RecordCounter("api_keys.validation_failed", 1, nil)
			return nil, fmt.Errorf("api key not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if !enabled {
		s.metrics.RecordCounter("api_keys.validation_disabled", 1, nil)
		return nil, fmt.Errorf("api key is disabled")
	}

	if expiresAt != nil && expiresAt.Before(time.Now()) {
		s.metrics.RecordCounter("api_keys.validation_expired", 1, nil)
		return nil, fmt.Errorf("api key has expired")
	}

	var permissions []string
	if err := json.Unmarshal([]byte(permsJSON), &permissions); err != nil {
		s.metrics.RecordCounter("api_keys.permissions_parse_error", 1, nil)
		return nil, fmt.Errorf("invalid permissions data: %w", err)
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("goroutine panic recovered", "panic", r)
			}
		}()
		select {
		case <-s.done:
			return
		default:
			updateCtx, updateCancel := context.WithTimeout(ctx, 5*time.Second)
			_, err := s.db.ExecContext(updateCtx,
				`UPDATE api_keys SET last_used_at = NOW(), request_count = request_count + 1 WHERE id = $1`, id)
			updateCancel()
			if err != nil {
				s.logger.Warn("failed to update last_used_at", "key_id", id, "error", err)
			}
		}
	}()

	s.metrics.RecordCounter("api_keys.validation_success", 1, nil)
	return &APIKeyRecord{
		ID:          id,
		ClientID:    clientID,
		Name:        name,
		KeyPrefix:   prefix,
		Permissions: permissions,
		Enabled:     enabled,
	}, nil
}

func (s *APIKeyStore) ListAPIKeys(ctx context.Context, clientID string, limit, offset int) ([]*APIKeyRecord, int, error) {
	var total int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM api_keys WHERE client_id = $1`, clientID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, client_id, name, key_prefix, permissions, enabled, created_at, updated_at, expires_at, last_used_at, request_count
		 FROM api_keys WHERE client_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		clientID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var records []*APIKeyRecord
	for rows.Next() {
		var r APIKeyRecord
		var permsJSON string
		var createdAt, updatedAt time.Time
		var expiresAt, lastUsedAt *time.Time
		var requestCount int64

		if err := rows.Scan(
			&r.ID, &r.ClientID, &r.Name, &r.KeyPrefix, &permsJSON, &r.Enabled,
			&createdAt, &updatedAt, &expiresAt, &lastUsedAt, &requestCount,
		); err != nil {
			return nil, 0, err
		}
		r.RequestCount = requestCount
		if err := json.Unmarshal([]byte(permsJSON), &r.Permissions); err != nil {
			s.logger.Warn("failed to parse api key permissions", "key_id", r.ID, "error", err)
			r.Permissions = []string{}
		}
		r.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		r.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		if expiresAt != nil {
			s := expiresAt.UTC().Format(time.RFC3339)
			r.ExpiresAt = &s
		}
		if lastUsedAt != nil {
			s := lastUsedAt.UTC().Format(time.RFC3339)
			r.LastUsedAt = &s
		}
		records = append(records, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func (s *APIKeyStore) DeleteAPIKey(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("api key not found")
	}
	s.metrics.RecordCounter("api_keys.deleted", 1, nil)
	return nil
}

func (s *APIKeyStore) ToggleAPIKey(ctx context.Context, id string, enabled bool) error {
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE api_keys SET enabled = $1, updated_at = NOW() WHERE id = $2`,
		enabled, id,
	)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}

func (s *APIKeyStore) RotateKey(ctx context.Context, id string, overlapWindow time.Duration) (*APIKeyRecord, string, error) {
	oldKey, err := s.getKeyByID(ctx, id)
	if err != nil {
		return nil, "", fmt.Errorf("get key %s: %w", id, err)
	}
	extendedExpiry := time.Now().Add(overlapWindow)
	if err := s.extendKeyExpiry(ctx, id, extendedExpiry); err != nil {
		return nil, "", fmt.Errorf("extend old key: %w", err)
	}
	return s.CreateAPIKey(ctx, oldKey.ClientID, oldKey.Name, oldKey.Permissions, &extendedExpiry)
}

func (s *APIKeyStore) ListExpiringKeys(ctx context.Context, before time.Time) ([]*APIKeyRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, client_id, name, key_prefix, enabled, expires_at
		 FROM api_keys WHERE enabled = true AND expires_at IS NOT NULL AND expires_at <= $1
		 ORDER BY expires_at ASC`, before.UTC())
	if err != nil {
		return nil, fmt.Errorf("query expiring keys: %w", err)
	}
	defer rows.Close()
	var keys []*APIKeyRecord
	for rows.Next() {
		var k APIKeyRecord
		var expiresAt sql.NullTime
		if err := rows.Scan(&k.ID, &k.ClientID, &k.Name, &k.KeyPrefix, &k.Enabled, &expiresAt); err != nil {
			return nil, fmt.Errorf("scan key: %w", err)
		}
		if expiresAt.Valid {
			formatted := expiresAt.Time.Format(time.RFC3339)
			k.ExpiresAt = &formatted
		}
		keys = append(keys, &k)
	}
	return keys, rows.Err()
}

func (s *APIKeyStore) getKeyByID(ctx context.Context, id string) (*APIKeyRecord, error) {
	var k APIKeyRecord
	var permsJSON string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, client_id, name, key_prefix, permissions, enabled FROM api_keys WHERE id = $1`,
		id,
	).Scan(&k.ID, &k.ClientID, &k.Name, &k.KeyPrefix, &permsJSON, &k.Enabled)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(permsJSON), &k.Permissions); err != nil {
		k.Permissions = []string{}
	}
	return &k, nil
}

func (s *APIKeyStore) extendKeyExpiry(ctx context.Context, id string, newExpiry time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE api_keys SET expires_at = $1, updated_at = NOW() WHERE id = $2`,
		newExpiry.UTC(), id)
	return err
}

func (s *APIKeyStore) LoadAllKeys(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT key_hash, client_id FROM api_keys WHERE enabled = true AND (expires_at IS NULL OR expires_at > NOW())`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	whitelist := make(map[string]string)
	for rows.Next() {
		var keyHash, clientID string
		if err := rows.Scan(&keyHash, &clientID); err != nil {
			return nil, err
		}
		whitelist[keyHash] = clientID
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return whitelist, nil
}

func KeyHash(plainKey string) string {
	if !strings.HasPrefix(plainKey, "cp_") {
		return ""
	}
	hash := sha256.Sum256([]byte(plainKey))
	return hex.EncodeToString(hash[:])
}
