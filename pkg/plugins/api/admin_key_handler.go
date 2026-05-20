package api

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

type AdminKeyHandler struct {
	db     *sql.DB
	logger core.Logger
}

func NewAdminKeyHandler(db *sql.DB, logger core.Logger) *AdminKeyHandler {
	return &AdminKeyHandler{
		db:     db,
		logger: logger,
	}
}

type apiKeyRow struct {
	ID          string `json:"id"`
	ClientID    string `json:"clientId"`
	Name        string `json:"name"`
	KeyHash     string `json:"-"`
	KeyPrefix   string `json:"keyPrefix"`
	Permissions string `json:"permissions"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	ExpiresAt   *string `json:"expiresAt"`
	LastUsedAt  *string `json:"lastUsedAt"`
}

type createKeyRequest struct {
	ClientID    string `json:"clientId"`
	Name        string `json:"name"`
	Permissions string `json:"permissions"`
	ExpiryDays  int    `json:"expiryDays"`
}

type createKeyResponse struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	KeyPrefix string `json:"keyPrefix"`
}

func (h *AdminKeyHandler) HandleListKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := h.db.QueryContext(ctx,
		`SELECT id, client_id, name, key_hash, key_prefix, permissions::text, enabled,
		 COALESCE(to_char(created_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
		 COALESCE(to_char(updated_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
		 COALESCE(to_char(expires_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
		 COALESCE(to_char(last_used_at, 'YYYY-MM-DD HH24:MI:SS'), '')
		 FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		h.logger.Error("Failed to list API keys", "error", err.Error())
		(&APIError{Code: "LIST_FAILED", Message: "Failed to list API keys", Status: http.StatusInternalServerError}).WriteHTTP(w)
		return
	}
	defer rows.Close()

	keys := make([]apiKeyRow, 0)
	for rows.Next() {
		var row apiKeyRow
		var expiresAt, lastUsedAt sql.NullString
		if scanErr := rows.Scan(&row.ID, &row.ClientID, &row.Name, &row.KeyHash,
			&row.KeyPrefix, &row.Permissions, &row.Enabled,
			&row.CreatedAt, &row.UpdatedAt, &expiresAt, &lastUsedAt); scanErr != nil {
			h.logger.Error("Failed to scan API key row", "error", scanErr.Error())
			continue
		}
		if expiresAt.Valid {
			row.ExpiresAt = &expiresAt.String
		}
		if lastUsedAt.Valid {
			row.LastUsedAt = &lastUsedAt.String
		}
		keys = append(keys, row)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"keys": keys})
}

func (h *AdminKeyHandler) HandleCreateKey(w http.ResponseWriter, r *http.Request) {
	var req createKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		(&APIError{Code: "INVALID_REQUEST", Message: "Invalid request body", Status: http.StatusBadRequest}).WriteHTTP(w)
		return
	}
	if req.Name == "" {
		(&APIError{Code: "INVALID_REQUEST", Message: "name is required", Status: http.StatusBadRequest}).WriteHTTP(w)
		return
	}

	rawKey, err := generateAPIKey(32)
	if err != nil {
		h.logger.Error("Failed to generate API key", "error", err.Error())
		(&APIError{Code: "KEY_GEN_FAILED", Message: "Failed to generate key", Status: http.StatusInternalServerError}).WriteHTTP(w)
		return
	}

	keyPrefix := rawKey[:8]
	keyHash := hashKey(rawKey)
	id := generateID()

	permissions := req.Permissions
	if permissions == "" {
		permissions = `["read","read_write"]`
	}

	var expiresAt interface{}
	if req.ExpiryDays > 0 {
		expiresAt = time.Now().Add(time.Duration(req.ExpiryDays) * 24 * time.Hour)
	} else {
		expiresAt = nil
	}

	ctx := r.Context()
	_, err = h.db.ExecContext(ctx,
		`INSERT INTO api_keys (id, client_id, name, key_hash, key_prefix, permissions, enabled, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, true, $7)`,
		id, req.ClientID, req.Name, keyHash, keyPrefix, permissions, expiresAt)
	if err != nil {
		h.logger.Error("Failed to create API key", "error", err.Error())
		(&APIError{Code: "CREATE_FAILED", Message: "Failed to create API key", Status: http.StatusInternalServerError}).WriteHTTP(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(createKeyResponse{
		ID:        id,
		Key:       rawKey,
		KeyPrefix: keyPrefix,
	})
}

func (h *AdminKeyHandler) HandleRevokeKey(w http.ResponseWriter, r *http.Request) {
	keyID := r.PathValue("id")
	if keyID == "" {
		(&APIError{Code: "INVALID_REQUEST", Message: "key id is required", Status: http.StatusBadRequest}).WriteHTTP(w)
		return
	}

	ctx := r.Context()
	result, err := h.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = $1`, keyID)
	if err != nil {
		h.logger.Error("Failed to revoke API key", "error", err.Error())
		(&APIError{Code: "REVOKE_FAILED", Message: "Failed to revoke API key", Status: http.StatusInternalServerError}).WriteHTTP(w)
		return
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		(&APIError{Code: "NOT_FOUND", Message: "API key not found", Status: http.StatusNotFound}).WriteHTTP(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"revoked": true})
}

func (h *AdminKeyHandler) HandleToggleKey(w http.ResponseWriter, r *http.Request) {
	keyID := r.PathValue("id")
	if keyID == "" {
		(&APIError{Code: "INVALID_REQUEST", Message: "key id is required", Status: http.StatusBadRequest}).WriteHTTP(w)
		return
	}

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		(&APIError{Code: "INVALID_REQUEST", Message: "Invalid request body", Status: http.StatusBadRequest}).WriteHTTP(w)
		return
	}

	ctx := r.Context()
	result, err := h.db.ExecContext(ctx, `UPDATE api_keys SET enabled = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		body.Enabled, keyID)
	if err != nil {
		h.logger.Error("Failed to toggle API key", "error", err.Error())
		(&APIError{Code: "TOGGLE_FAILED", Message: "Failed to update API key status", Status: http.StatusInternalServerError}).WriteHTTP(w)
		return
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		(&APIError{Code: "NOT_FOUND", Message: "API key not found", Status: http.StatusNotFound}).WriteHTTP(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"updated": true, "enabled": body.Enabled})
}

func (h *AdminKeyHandler) HandleGetKeyByID(w http.ResponseWriter, r *http.Request) {
	keyID := r.PathValue("id")
	if keyID == "" {
		(&APIError{Code: "INVALID_REQUEST", Message: "key id is required", Status: http.StatusBadRequest}).WriteHTTP(w)
		return
	}

	ctx := r.Context()
	var row apiKeyRow
	var expiresAt, lastUsedAt sql.NullString
	err := h.db.QueryRowContext(ctx,
		`SELECT id, client_id, name, key_hash, key_prefix, permissions::text, enabled,
		 COALESCE(to_char(created_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
		 COALESCE(to_char(updated_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
		 COALESCE(to_char(expires_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
		 COALESCE(to_char(last_used_at, 'YYYY-MM-DD HH24:MI:SS'), '')
		 FROM api_keys WHERE id = $1`, keyID).
		Scan(&row.ID, &row.ClientID, &row.Name, &row.KeyHash,
			&row.KeyPrefix, &row.Permissions, &row.Enabled,
			&row.CreatedAt, &row.UpdatedAt, &expiresAt, &lastUsedAt)
	if err == sql.ErrNoRows {
		(&APIError{Code: "NOT_FOUND", Message: "API key not found", Status: http.StatusNotFound}).WriteHTTP(w)
		return
	}
	if err != nil {
		h.logger.Error("Failed to get API key", "error", err.Error())
		(&APIError{Code: "GET_FAILED", Message: "Failed to retrieve API key", Status: http.StatusInternalServerError}).WriteHTTP(w)
		return
	}

	if expiresAt.Valid {
		row.ExpiresAt = &expiresAt.String
	}
	if lastUsedAt.Valid {
		row.LastUsedAt = &lastUsedAt.String
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(row)
}

func generateAPIKey(length int) (string, error) {
	bytes := make([]byte, length)
	for i := range bytes {
		n, err := rand.Int(rand.Reader, big.NewInt(62))
		if err != nil {
			return "", err
		}
		chars := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
		bytes[i] = chars[n.Int64()]
	}
	return string(bytes), nil
}

func hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

func generateID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return base64.RawURLEncoding.EncodeToString(bytes)
}