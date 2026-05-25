package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestNewAdminAPIKeyHandler(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	h := NewAdminAPIKeyHandler(nil, logger)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.store != nil {
		t.Error("expected nil store")
	}
}

func TestAdminAPIKeyHandler_HandleCreateAPIKey_NilStore(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	h := NewAdminAPIKeyHandler(nil, logger)

	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", nil)
	rec := httptest.NewRecorder()
	h.HandleCreateAPIKey(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}

func TestAdminAPIKeyHandler_HandleListAPIKeys_NilStore(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	h := NewAdminAPIKeyHandler(nil, logger)

	req := httptest.NewRequest(http.MethodGet, "/admin/api-keys", nil)
	rec := httptest.NewRecorder()
	h.HandleListAPIKeys(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}

func TestAdminAPIKeyHandler_HandleDeleteAPIKey_NilStore(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	h := NewAdminAPIKeyHandler(nil, logger)

	req := httptest.NewRequest(http.MethodDelete, "/admin/api-keys/some-id", nil)
	rec := httptest.NewRecorder()
	h.HandleDeleteAPIKey(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}

func TestAdminAPIKeyHandler_HandleToggleAPIKey_NilStore(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	h := NewAdminAPIKeyHandler(nil, logger)

	req := httptest.NewRequest(http.MethodPut, "/admin/api-keys/some-id/toggle", nil)
	rec := httptest.NewRecorder()
	h.HandleToggleAPIKey(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}
