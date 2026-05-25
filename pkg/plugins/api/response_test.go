package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/plugins/api/apierrors"
)

func TestWriteEnvelope_Success(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	WriteEnvelope(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("expected Content-Type application/json")
	}

	var env APIEnvelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Meta == nil || env.Meta.Timestamp == 0 {
		t.Error("expected meta with timestamp")
	}
}

func TestWriteEnvelope_Created(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	WriteEnvelope(w, http.StatusCreated, []string{"a", "b"})

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
}

func TestWriteEnvelope_NilData(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	WriteEnvelope(w, http.StatusNoContent, nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
}

func TestWriteErrorEnvelope_WithAPIError(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	apiErr := &apierrors.APIError{Code: "NOT_FOUND", Message: "not found", Status: http.StatusNotFound}
	WriteErrorEnvelope(w, apiErr)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var env APIEnvelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Error == nil {
		t.Fatal("expected error in envelope")
	}
}

func TestWriteErrorEnvelope_PlainError(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	WriteErrorEnvelope(w, errors.New("something broke"))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestWriteErrorEnvelope_NilError(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	WriteErrorEnvelope(w, nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for nil error, got %d", w.Code)
	}
}
