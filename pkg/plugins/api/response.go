package api

import (
	"encoding/json"
	"net/http"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"
)

// APIEnvelope is the unified JSON response wrapper for all API responses.
// Every endpoint returns this structure, providing consistent metadata
// for clients to parse regardless of the specific resource.
type APIEnvelope struct {
	Data  any           `json:"data,omitempty"`
	Error *APIError     `json:"error,omitempty"`
	Meta  *EnvelopeMeta `json:"meta,omitempty"`
}

// EnvelopeMeta carries request-scoped metadata for observability and debugging.
type EnvelopeMeta struct {
	RequestID  string `json:"request_id,omitempty"`
	Timestamp  int64  `json:"timestamp"`
	APIVersion string `json:"api_version,omitempty"`
}

// WriteEnvelope serializes a successful response with the standard envelope.
// Sets Content-Type to application/json and writes the HTTP response.
func WriteEnvelope(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIEnvelope{
		Data: data,
		Meta: &EnvelopeMeta{
			Timestamp: time.Now().Unix(),
		},
	})
}

// WriteErrorEnvelope serializes an error response with the standard envelope.
// If err is an *APIError, it uses its status code and structured fields.
// Otherwise it maps to a generic 500.
func WriteErrorEnvelope(w http.ResponseWriter, err error) {
	apiErr := MapErrorToAPIError(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.Status)
	_ = json.NewEncoder(w).Encode(APIEnvelope{
		Error: apiErr,
		Meta: &EnvelopeMeta{
			Timestamp: time.Now().Unix(),
		},
	})
}

// HealthResponse is the standard health check response body.
type HealthResponse struct {
	Status     string                      `json:"status"`
	Timestamp  int64                       `json:"timestamp"`
	Version    string                      `json:"version"`
	Components map[string]*ComponentStatus `json:"components,omitempty"`
}

// writeRedactedInternalError logs the underlying cause server-side and writes
// a stable machine-readable 500 body. Internal error text — DSNs, SQL, driver
// messages, stack frames — must never reach the client: it discloses
// infrastructure (credentials, host names, schema) to any caller that can
// provoke a store or dependency failure.
func writeRedactedInternalError(w http.ResponseWriter, logger portcontract.Logger, op string, err error) {
	if logger != nil && err != nil {
		logger.Error("internal error", "op", op, "error", err.Error())
	}
	WriteErrorEnvelope(w, ErrInternalServer("internal error"))
}
