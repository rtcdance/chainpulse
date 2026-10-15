package api

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// internalLeakMarker mimics the most damaging class of internal detail a store
// failure can surface: a DSN carrying credentials. It is planted in the error
// the fake database returns and must never appear in any response body.
const internalLeakMarker = "SECRET_DSN postgres://chainpulse_admin:hunter2@db.internal:5432/chainpulse"

// failingConnector turns every connection attempt into an error carrying
// internalLeakMarker, which database/sql propagates through the store's
// error wrapping into the handler — a forced dependency failure without
// needing a live database.
type failingConnector struct{ err error }

func (c failingConnector) Connect(context.Context) (driver.Conn, error) { return nil, c.err }
func (c failingConnector) Driver() driver.Driver                        { return failingDriver{} }

type failingDriver struct{}

func (failingDriver) Open(string) (driver.Conn, error) {
	return nil, fmt.Errorf("unused")
}

func newFailingDB(t *testing.T) *sql.DB {
	t.Helper()

	db := sql.OpenDB(failingConnector{err: fmt.Errorf("%s", internalLeakMarker)})
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// recordingLogger captures Error log entries so the test can assert the
// internal detail still reaches the server-side log after redaction.
type recordingLogger struct {
	mu sync.Mutex
	MockLogger
	errorLogs []string
}

func (l *recordingLogger) Error(msg string, fields ...any) {
	l.mu.Lock()
	l.errorLogs = append(l.errorLogs, fmt.Sprintf("%s %v", msg, fields))
	l.mu.Unlock()
	l.MockLogger.Error(msg, fields...)
}

func (l *recordingLogger) loggedErrors() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.errorLogs...)
}

// TestAdminHandlers_RedactInternalStoreErrors forces a store failure on every
// admin handler that talks to the database and asserts the end-to-end
// contract: a 500 body carries only the stable machine-readable code, never
// the internal cause, while the cause is still logged server-side.
func TestAdminHandlers_RedactInternalStoreErrors(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	metrics := NewMockMetricsCollector()

	apiKeyHandler := NewAdminAPIKeyHandler(NewAPIKeyStore(newFailingDB(t), logger, metrics), logger)
	webhookHandler := NewAdminWebhookHandler(NewWebhookStore(newFailingDB(t), logger, metrics), logger)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		handle func(w http.ResponseWriter, r *http.Request)
	}{
		{
			name:   "admin api key create",
			method: http.MethodPost,
			path:   "/admin/api-keys",
			body:   `{"clientId":"operator-a","name":"ci"}`,
			handle: apiKeyHandler.HandleCreateAPIKey,
		},
		{
			name:   "admin api key list",
			method: http.MethodGet,
			path:   "/admin/api-keys?clientId=operator-a",
			handle: apiKeyHandler.HandleListAPIKeys,
		},
		{
			name:   "admin api key delete",
			method: http.MethodDelete,
			path:   "/admin/api-keys/ak_1",
			handle: apiKeyHandler.HandleDeleteAPIKey,
		},
		{
			name:   "admin api key toggle",
			method: http.MethodPut,
			path:   "/admin/api-keys/ak_1/toggle",
			body:   `{"enabled":true}`,
			handle: apiKeyHandler.HandleToggleAPIKey,
		},
		{
			name:   "admin webhook create",
			method: http.MethodPost,
			path:   "/admin/webhooks",
			body:   `{"clientId":"operator-a","name":"ci","url":"https://hooks.example.test/x"}`,
			handle: webhookHandler.HandleCreateWebhook,
		},
		{
			name:   "admin webhook list",
			method: http.MethodGet,
			path:   "/admin/webhooks?clientId=operator-a",
			handle: webhookHandler.HandleListWebhooks,
		},
		{
			name:   "admin webhook delete",
			method: http.MethodDelete,
			path:   "/admin/webhooks/wh_1",
			handle: webhookHandler.HandleDeleteWebhook,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.path, body)
			rec := httptest.NewRecorder()
			tt.handle(rec, req)

			require.Equal(t, http.StatusInternalServerError, rec.Code)

			raw := rec.Body.String()
			var envelope APIEnvelope
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope), "body must stay machine-readable: %s", raw)
			require.NotNil(t, envelope.Error, "body must carry an error envelope: %s", raw)
			assert.Equal(t, "INTERNAL_SERVER_ERROR", envelope.Error.Code)

			for _, secret := range []string{"hunter2", "db.internal", "SECRET_DSN", "failed to", "sql"} {
				assert.NotContains(t, raw, secret, "500 body must not carry internal detail: %s", raw)
			}

			logs := strings.Join(logger.loggedErrors(), "\n")
			require.NotEmpty(t, logs, "a redacted response must still log the cause server-side")
			assert.Contains(t, logs, "hunter2", "the internal detail belongs in the log, not the response")
		})
	}
}
