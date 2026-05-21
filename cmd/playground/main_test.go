package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func newTestPlayground(t *testing.T) *playground {
	t.Helper()
	return newPlayground(core.NewDefaultLogger(core.LogLevelError))
}

func TestPlayground_Home(t *testing.T) {
	pg := newTestPlayground(t)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	pg.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ChainPulse") {
		t.Fatal("home page should contain ChainPulse")
	}
}

func TestPlayground_Stats(t *testing.T) {
	pg := newTestPlayground(t)
	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()
	pg.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["mode"] != "in-memory" {
		t.Fatalf("expected mode=in-memory, got %v", body["mode"])
	}
	if body["version"] != "chainpulse-playground" {
		t.Fatalf("expected version=chainpulse-playground, got %v", body["version"])
	}
}

func TestPlayground_GenerateEvents(t *testing.T) {
	pg := newTestPlayground(t)
	req := httptest.NewRequest("POST", "/generate", nil)
	w := httptest.NewRecorder()
	pg.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	generated := body["generated"].(float64)
	if generated != 5 {
		t.Fatalf("expected 5 generated events, got %v", generated)
	}
}

func TestPlayground_ListEventsEmpty(t *testing.T) {
	pg := newTestPlayground(t)
	req := httptest.NewRequest("GET", "/events", nil)
	w := httptest.NewRecorder()
	pg.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["total"] != float64(0) {
		t.Fatalf("expected 0 events initially, got %v", body["total"])
	}
}

func TestPlayground_ListEventsAfterGenerate(t *testing.T) {
	pg := newTestPlayground(t)

	genReq := httptest.NewRequest("POST", "/generate", nil)
	genW := httptest.NewRecorder()
	pg.ServeHTTP(genW, genReq)

	listReq := httptest.NewRequest("GET", "/events", nil)
	listW := httptest.NewRecorder()
	pg.ServeHTTP(listW, listReq)

	var body map[string]any
	if err := json.NewDecoder(listW.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["total"] != float64(5) {
		t.Fatalf("expected 5 events after generate, got %v", body["total"])
	}
}

func TestPlayground_GenerateSwap(t *testing.T) {
	pg := newTestPlayground(t)
	req := httptest.NewRequest("POST", "/generate-swap", nil)
	w := httptest.NewRecorder()
	pg.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	eventName := body["event"].(string)
	if eventName != "Swap (via defi.ConstantProductAMM)" {
		t.Fatalf("expected Swap, got %s", eventName)
	}
}

func TestPlayground_GenerateAA(t *testing.T) {
	pg := newTestPlayground(t)
	req := httptest.NewRequest("POST", "/generate-aa", nil)
	w := httptest.NewRecorder()
	pg.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	eventName := body["event"].(string)
	if eventName != "UserOperationEvent (ERC-4337)" {
		t.Fatalf("expected UserOperationEvent, got %s", eventName)
	}
}

func TestPlayground_ReplayCheck(t *testing.T) {
	pg := newTestPlayground(t)
	req := httptest.NewRequest("GET", "/replay-check?v=37", nil)
	w := httptest.NewRecorder()
	pg.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	chainID := body["extracted_chain"].(float64)
	if chainID != 1 {
		t.Fatalf("expected chain 1 for v=37, got %v", chainID)
	}
}

func TestPlayground_NotFound(t *testing.T) {
	pg := newTestPlayground(t)
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()
	pg.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestPlayground_Tutorial(t *testing.T) {
	pg := newTestPlayground(t)
	req := httptest.NewRequest("GET", "/tutorial", nil)
	w := httptest.NewRecorder()
	pg.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestPlayground_Concepts(t *testing.T) {
	pg := newTestPlayground(t)
	req := httptest.NewRequest("GET", "/concepts", nil)
	w := httptest.NewRecorder()
	pg.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestPlayground_PoolDemo(t *testing.T) {
	pg := newTestPlayground(t)
	req := httptest.NewRequest("GET", "/pool", nil)
	w := httptest.NewRecorder()
	pg.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestPlayground_Publish(t *testing.T) {
	pg := newTestPlayground(t)
	req := httptest.NewRequest("POST", "/publish?payload=test", nil)
	w := httptest.NewRecorder()
	pg.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestPlayground_StatsAfterGenerate(t *testing.T) {
	pg := newTestPlayground(t)

	genReq := httptest.NewRequest("POST", "/generate", nil)
	genW := httptest.NewRecorder()
	pg.ServeHTTP(genW, genReq)

	statsReq := httptest.NewRequest("GET", "/stats", nil)
	statsW := httptest.NewRecorder()
	pg.ServeHTTP(statsW, statsReq)

	var body map[string]any
	if err := json.NewDecoder(statsW.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["total_events"] != float64(5) {
		t.Fatalf("expected total_events=5, got %v", body["total_events"])
	}
}
