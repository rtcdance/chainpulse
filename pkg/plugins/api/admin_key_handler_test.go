package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestNewAdminKeyHandler(t *testing.T) {
	t.Parallel()
	h := NewAdminKeyHandler(nil, nil)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.db != nil {
		t.Error("expected nil db")
	}
}

func TestAdminKeyHandler_HandleRevokeKey_NoID(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	h := NewAdminKeyHandler(nil, logger)

	req := httptest.NewRequest(http.MethodDelete, "/admin/keys/", nil)
	rec := httptest.NewRecorder()
	h.HandleRevokeKey(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAdminKeyHandler_HandleToggleKey_NoID(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	h := NewAdminKeyHandler(nil, logger)

	req := httptest.NewRequest(http.MethodPut, "/admin/keys/", nil)
	rec := httptest.NewRecorder()
	h.HandleToggleKey(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAdminKeyHandler_HandleGetKeyByID_NoID(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	h := NewAdminKeyHandler(nil, logger)

	req := httptest.NewRequest(http.MethodGet, "/admin/keys/", nil)
	rec := httptest.NewRecorder()
	h.HandleGetKeyByID(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}