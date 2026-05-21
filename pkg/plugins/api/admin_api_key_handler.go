package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// AdminAPIKeyHandler handles admin API key management endpoints
type AdminAPIKeyHandler struct {
	store  *APIKeyStore
	logger core.Logger
}

// NewAdminAPIKeyHandler creates a new admin API key handler
func NewAdminAPIKeyHandler(store *APIKeyStore, logger core.Logger) *AdminAPIKeyHandler {
	return &AdminAPIKeyHandler{
		store:  store,
		logger: logger,
	}
}

// HandleCreateAPIKey creates a new API key
// POST /admin/api-keys
func (h *AdminAPIKeyHandler) HandleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.store.db == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	var req struct {
		ClientID    string   `json:"clientId"`
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
		ExpiresIn   string   `json:"expiresIn,omitempty"` // e.g. "720h" (30 days)
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ClientID == "" {
		writeJSONError(w, http.StatusBadRequest, "clientId is required")
		return
	}
	if req.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Permissions == nil {
		req.Permissions = []string{"read"}
	}

	var expiresAt *time.Time
	if req.ExpiresIn != "" {
		d, err := time.ParseDuration(req.ExpiresIn)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid expiresIn: %v", err))
			return
		}
		t := time.Now().Add(d)
		expiresAt = &t
	}

	record, plainKey, err := h.store.CreateAPIKey(r.Context(), req.ClientID, req.Name, req.Permissions, expiresAt)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create key: %v", err))
		return
	}

	// Return the plain key only on creation — it cannot be retrieved later
	writeJSONResponse(w, http.StatusCreated, map[string]any{
		"key":    plainKey,
		"record": record,
	})
}

// HandleListAPIKeys lists API keys for a client
// GET /admin/api-keys?clientId=xxx&limit=20&offset=0
func (h *AdminAPIKeyHandler) HandleListAPIKeys(w http.ResponseWriter, r *http.Request) {
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

	records, total, err := h.store.ListAPIKeys(r.Context(), clientID, limit, offset)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list keys: %v", err))
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

// HandleDeleteAPIKey deletes an API key
// DELETE /admin/api-keys/{id}
func (h *AdminAPIKeyHandler) HandleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.store.db == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		// Fallback for older Go versions
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) > 0 {
			id = parts[len(parts)-1]
		}
	}
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "key id is required")
		return
	}

	if err := h.store.DeleteAPIKey(r.Context(), id); err != nil {
		if err.Error() == "api key not found" {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete key: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"message":   "api key deleted",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// HandleToggleAPIKey enables or disables an API key
// PUT /admin/api-keys/{id}/toggle
func (h *AdminAPIKeyHandler) HandleToggleAPIKey(w http.ResponseWriter, r *http.Request) {
	if h.store == nil || h.store.db == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(r.URL.Path, "/")
		for i, p := range parts {
			if p == "api-keys" && i+1 < len(parts) {
				id = parts[i+1]
				break
			}
		}
	}
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "key id is required")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.store.ToggleAPIKey(r.Context(), id, req.Enabled); err != nil {
		if err.Error() == "api key not found" {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to toggle key: %v", err))
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"message":   fmt.Sprintf("api key %s", map[bool]string{true: "enabled", false: "disabled"}[req.Enabled]),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// EnsureAdminTables ensures the api_keys table exists (for non-migration environments)
func EnsureAdminTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS api_keys (
			id VARCHAR(255) PRIMARY KEY,
			client_id VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			key_hash VARCHAR(128) NOT NULL,
			key_prefix VARCHAR(16) NOT NULL,
			permissions JSONB DEFAULT '[]',
			enabled BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP,
			last_used_at TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_api_keys_client_id ON api_keys(client_id);
		CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
		CREATE INDEX IF NOT EXISTS idx_api_keys_enabled ON api_keys(enabled);
	`)
	return err
}
