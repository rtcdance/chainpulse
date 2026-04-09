package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestHandlerCheckHealthUsesConfiguredHeaders(t *testing.T) {
	t.Helper()

	apiKeySeen := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKeySeen = r.Header.Get("X-API-Key")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	handler := NewHandler("test", "test", server.URL)
	handler.SetHealthHeaders(map[string]string{"X-API-Key": "upstream-key"})

	if !handler.CheckHealth() {
		t.Fatal("expected health check to succeed")
	}
	if apiKeySeen != "upstream-key" {
		t.Fatalf("expected X-API-Key header upstream-key, got %q", apiKeySeen)
	}
}
