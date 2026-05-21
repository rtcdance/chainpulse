package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// WebhookRecord represents a persisted webhook configuration
type WebhookRecord struct {
	ID                 string   `json:"id"`
	ClientID           string   `json:"clientId"`
	Name               string   `json:"name"`
	URL                string   `json:"url"`
	Secret             string   `json:"-"`
	Events             []string `json:"events"`
	Enabled            bool     `json:"enabled"`
	CreatedAt          string   `json:"createdAt"`
	UpdatedAt          string   `json:"updatedAt"`
	LastDeliveryAt     *string  `json:"lastDeliveryAt,omitempty"`
	LastDeliveryStatus *string  `json:"lastDeliveryStatus,omitempty"`
	FailureCount       int      `json:"failureCount"`
}

// WebhookDelivery represents a single delivery attempt
type WebhookDelivery struct {
	WebhookID  string         `json:"webhookId"`
	Event      string         `json:"event"`
	Payload    map[string]any `json:"payload"`
	Attempt    int            `json:"attempt"`
	MaxRetries int            `json:"maxRetries"`
}

// WebhookStore manages persistent webhook storage and delivery
type WebhookStore struct {
	db         *sql.DB
	logger     core.Logger
	metrics    core.MetricsCollector
	httpClient *http.Client
	maxRetries int
	retryDelay time.Duration

	deliverSem chan struct{} // bounded semaphore for concurrent deliveries
	wg         sync.WaitGroup
}

// NewWebhookStore creates a new webhook store
func NewWebhookStore(db *sql.DB, logger core.Logger, metrics core.MetricsCollector) *WebhookStore {
	const maxConcurrentDeliveries = 16
	return &WebhookStore{
		db:         db,
		logger:     logger,
		metrics:    metrics,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		maxRetries: 3,
		retryDelay: 5 * time.Second,
		deliverSem: make(chan struct{}, maxConcurrentDeliveries),
	}
}

// Stop waits for all in-flight webhook deliveries to complete.
func (s *WebhookStore) Stop() {
	s.wg.Wait()
}

// CreateWebhook creates a new webhook configuration
func (s *WebhookStore) CreateWebhook(ctx context.Context, clientID, name, url, secret string, events []string) (*WebhookRecord, error) {
	id := fmt.Sprintf("wh_%d", time.Now().UnixNano())
	eventsJSON, _ := json.Marshal(events)
	if len(events) == 0 {
		eventsJSON = []byte(`["event:created","event:confirmed","event:failed"]`)
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO webhooks (id, client_id, name, url, secret, events, enabled)
		 VALUES ($1, $2, $3, $4, $5, $6, true)`,
		id, clientID, name, url, secret, string(eventsJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook: %w", err)
	}

	return &WebhookRecord{
		ID:       id,
		ClientID: clientID,
		Name:     name,
		URL:      url,
		Events:   events,
		Enabled:  true,
	}, nil
}

// GetWebhooksByEvent returns all enabled webhooks subscribed to a given event type
func (s *WebhookStore) GetWebhooksByEvent(ctx context.Context, eventType string) ([]*WebhookRecord, error) {
	// Use JSONB contains operator
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, client_id, name, url, secret, events, enabled, failure_count
		 FROM webhooks 
		 WHERE enabled = true AND events @> $1`,
		fmt.Sprintf(`["%s"]`, eventType),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck // defer close

	var webhooks []*WebhookRecord
	for rows.Next() {
		var w WebhookRecord
		var eventsJSON string
		if err := rows.Scan(&w.ID, &w.ClientID, &w.Name, &w.URL, &w.Secret, &eventsJSON, &w.Enabled, &w.FailureCount); err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(eventsJSON), &w.Events)
		webhooks = append(webhooks, &w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return webhooks, nil
}

// ListWebhooks returns webhooks for a client
func (s *WebhookStore) ListWebhooks(ctx context.Context, clientID string, limit, offset int) ([]*WebhookRecord, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM webhooks WHERE client_id = $1`, clientID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, client_id, name, url, events, enabled, created_at, updated_at, 
		        last_delivery_at, last_delivery_status, failure_count
		 FROM webhooks WHERE client_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		clientID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close() //nolint:errcheck // defer close

	var records []*WebhookRecord
	for rows.Next() {
		var r WebhookRecord
		var eventsJSON string
		var createdAt, updatedAt time.Time
		var lastDeliveryAt *time.Time
		var lastDeliveryStatus *string

		if err := rows.Scan(&r.ID, &r.ClientID, &r.Name, &r.URL, &eventsJSON, &r.Enabled,
			&createdAt, &updatedAt, &lastDeliveryAt, &lastDeliveryStatus, &r.FailureCount); err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(eventsJSON), &r.Events)
		r.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		r.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		if lastDeliveryAt != nil {
			s := lastDeliveryAt.UTC().Format(time.RFC3339)
			r.LastDeliveryAt = &s
		}
		r.LastDeliveryStatus = lastDeliveryStatus
		records = append(records, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

// DeleteWebhook removes a webhook
func (s *WebhookStore) DeleteWebhook(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM webhooks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("webhook not found")
	}
	return nil
}

// Deliver sends a webhook payload with HMAC signature and retry logic
func (s *WebhookStore) Deliver(ctx context.Context, webhook *WebhookRecord, eventType string, payload map[string]any) error {
	body, err := json.Marshal(map[string]any{
		"event":     eventType,
		"timestamp": time.Now().Unix(),
		"data":      payload,
	})
	if err != nil {
		return err
	}

	// Compute HMAC-SHA256 signature
	mac := hmac.New(sha256.New, []byte(webhook.Secret))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	var lastErr error
	for attempt := 1; attempt <= s.maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Webhook-Signature", signature)
		req.Header.Set("X-Webhook-Event", eventType)
		req.Header.Set("X-Webhook-ID", webhook.ID)
		req.Header.Set("X-Webhook-Attempt", strconv.Itoa(attempt))

		resp, err := s.httpClient.Do(req)
		if err != nil {
			lastErr = err
			s.metrics.RecordCounter("webhook.delivery_error", 1, nil)
			if attempt < s.maxRetries {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(s.retryDelay):
				}
			}
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close() //nolint:errcheck

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			s.recordDelivery(ctx, webhook.ID, "delivered")
			s.metrics.RecordCounter("webhook.delivery_success", 1, nil)
			return nil
		}

		lastErr = fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(respBody))
		s.metrics.RecordCounter("webhook.delivery_http_error", 1, nil)
		if attempt < s.maxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.retryDelay):
			}
		}
	}

	s.recordDelivery(ctx, webhook.ID, "failed")
	s.metrics.RecordCounter("webhook.delivery_failed", 1, nil)
	return lastErr
}

// NotifyEvent delivers webhook notifications to all webhooks subscribed to an event type
func (s *WebhookStore) NotifyEvent(ctx context.Context, eventType string, payload map[string]any) {
	if s.db == nil {
		return
	}
	webhooks, err := s.GetWebhooksByEvent(ctx, eventType)
	if err != nil {
		s.logger.Error("failed to get webhooks for event", "event", eventType, "error", err)
		return
	}

	for _, w := range webhooks {
		s.deliverSem <- struct{}{} // acquire semaphore slot (blocks if at capacity)
		s.wg.Add(1)
		go func(webhook *WebhookRecord) {
			defer func() {
				<-s.deliverSem // release semaphore slot
				s.wg.Done()
			}()
			if err := s.Deliver(ctx, webhook, eventType, payload); err != nil {
				s.logger.Error("webhook delivery failed",
					"webhookId", webhook.ID, "event", eventType, "error", err)
			}
		}(w)
	}
}

func (s *WebhookStore) recordDelivery(ctx context.Context, webhookID, status string) {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE webhooks SET last_delivery_at = NOW(), last_delivery_status = $1,
		 failure_count = CASE WHEN $1 = 'failed' THEN failure_count + 1 ELSE 0 END,
		 updated_at = NOW()
		 WHERE id = $2`,
		status, webhookID,
	); err != nil {
		s.logger.Error("failed to record webhook delivery",
			"webhookId", webhookID, "status", status, "error", err)
	}
}

// EnsureWebhookTables ensures the webhooks table exists
func EnsureWebhookTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS webhooks (
			id VARCHAR(255) PRIMARY KEY,
			client_id VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			url VARCHAR(2048) NOT NULL,
			secret VARCHAR(255) NOT NULL,
			events JSONB DEFAULT '["event:created","event:confirmed","event:failed"]',
			enabled BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_delivery_at TIMESTAMP,
			last_delivery_status VARCHAR(32),
			failure_count INTEGER DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_webhooks_client_id ON webhooks(client_id);
		CREATE INDEX IF NOT EXISTS idx_webhooks_enabled ON webhooks(enabled);
	`)
	return err
}

// AdminWebhookHandler handles admin webhook management endpoints
type AdminWebhookHandler struct {
	store  *WebhookStore
	logger core.Logger
}

// NewAdminWebhookHandler creates a new admin webhook handler
func NewAdminWebhookHandler(store *WebhookStore, logger core.Logger) *AdminWebhookHandler {
	return &AdminWebhookHandler{store: store, logger: logger}
}

// HandleCreateWebhook creates a new webhook
// POST /admin/webhooks
func (h *AdminWebhookHandler) HandleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.store.db == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	var req struct {
		ClientID string   `json:"clientId"`
		Name     string   `json:"name"`
		URL      string   `json:"url"`
		Secret   string   `json:"secret"`
		Events   []string `json:"events"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ClientID == "" || req.Name == "" || req.URL == "" {
		writeJSONError(w, http.StatusBadRequest, "clientId, name, and url are required")
		return
	}
	if req.Secret == "" {
		// Generate a cryptographically random secret
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to generate secret")
			return
		}
		req.Secret = "whsec_" + hex.EncodeToString(buf)
	}
	if req.Events == nil {
		req.Events = []string{"event:created", "event:confirmed", "event:failed"}
	}

	record, err := h.store.CreateWebhook(r.Context(), req.ClientID, req.Name, req.URL, req.Secret, req.Events)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create webhook: %v", err))
		return
	}

	// Return the secret only on creation
	writeJSONResponse(w, http.StatusCreated, map[string]any{
		"secret": req.Secret,
		"record": record,
	})
}

// HandleListWebhooks lists webhooks for a client
// GET /admin/webhooks?clientId=xxx&limit=20&offset=0
func (h *AdminWebhookHandler) HandleListWebhooks(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.store.db == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	clientID := r.URL.Query().Get("clientId")
	if clientID == "" {
		writeJSONError(w, http.StatusBadRequest, "clientId query parameter is required")
		return
	}

	limit := 20
	offset := 0
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}

	records, total, err := h.store.ListWebhooks(r.Context(), clientID, limit, offset)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list webhooks: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"data":      records,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// HandleDeleteWebhook deletes a webhook
// DELETE /admin/webhooks/{id}
func (h *AdminWebhookHandler) HandleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.store.db == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(r.URL.Path, "/")
		for i, p := range parts {
			if p == "webhooks" && i+1 < len(parts) {
				id = parts[i+1]
				break
			}
		}
	}
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "webhook id is required")
		return
	}

	if err := h.store.DeleteWebhook(r.Context(), id); err != nil {
		if err.Error() == "webhook not found" {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete webhook: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"message":   "webhook deleted",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
