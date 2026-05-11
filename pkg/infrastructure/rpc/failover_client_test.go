package rpc

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestFailoverRPCClientBasicRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","result":"0x1","id":1}`))
	}))
	defer server.Close()

	client := NewFailoverRPCClient(FailoverConfig{
		PrimaryURL:        server.URL,
		RequestsPerSecond: 100,
	})

	req, err := http.NewRequest("POST", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestFailoverRPCClientFailover(t *testing.T) {
	var primaryCalls, fallbackCalls int32

	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&primaryCalls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer primary.Close()

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&fallbackCalls, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","result":"0x1","id":1}`))
	}))
	defer fallback.Close()

	client := NewFailoverRPCClient(FailoverConfig{
		PrimaryURL:             primary.URL,
		FallbackURLs:           []string{fallback.URL},
		MaxConsecutiveFailures: 1, // trip circuit on first failure
		RequestsPerSecond:      100,
	})

	req, err := http.NewRequest("POST", primary.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 from fallback, got %d", resp.StatusCode)
	}

	if atomic.LoadInt32(&primaryCalls) == 0 {
		t.Error("expected primary to be called")
	}
	if atomic.LoadInt32(&fallbackCalls) == 0 {
		t.Error("expected fallback to be called")
	}
}

func TestFailoverRPCClient429Handling(t *testing.T) {
	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		if count <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`ok`))
	}))
	defer server.Close()

	client := NewFailoverRPCClient(FailoverConfig{
		PrimaryURL:             server.URL,
		MaxConsecutiveFailures: 10, // don't trip circuit
		RequestsPerSecond:      100,
	})

	req, err := http.NewRequest("POST", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 after 429 retry, got %d", resp.StatusCode)
	}
}

func TestFailoverRPCClientCircuitBreaker(t *testing.T) {
	var calls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewFailoverRPCClient(FailoverConfig{
		PrimaryURL:             server.URL,
		MaxConsecutiveFailures: 3,
		CircuitResetTimeout:    100 * time.Millisecond,
		RequestsPerSecond:      1000,
	})

	// Make requests until circuit opens
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("POST", server.URL, nil)
		client.Do(req)
	}

	if client.HealthyEndpoints() > 0 {
		t.Error("expected all endpoints to be circuit-open after failures")
	}

	// Wait for circuit reset
	time.Sleep(150 * time.Millisecond)

	if client.HealthyEndpoints() == 0 {
		t.Error("expected circuit to reset after timeout")
	}
}

func TestFailoverRPCClientRateLimiting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewFailoverRPCClient(FailoverConfig{
		PrimaryURL:        server.URL,
		RequestsPerSecond: 2,
	})

	// Should be able to make a few requests
	req, _ := http.NewRequest("POST", server.URL, nil)
	if _, err := client.Do(req); err != nil {
		t.Errorf("first request should succeed: %v", err)
	}

	// Drain the bucket
	for i := 0; i < 10; i++ {
		client.acquireToken()
	}

	// Next should fail
	if err := client.acquireToken(); err == nil {
		t.Error("expected rate limit error after draining token bucket")
	}
}
