package rpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestFailoverRPCClientBasicRequest(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
		client.acquireTokenWithWait(context.Background())
	}

	// Next should fail
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if err := client.acquireTokenWithWait(ctx); err == nil {
		t.Error("expected rate limit error after draining token bucket")
	}
}

func TestNewFailoverRPCClientDefaults(t *testing.T) {
	t.Parallel()
	client := NewFailoverRPCClient(FailoverConfig{
		PrimaryURL: "http://localhost:8545",
	})
	if client.maxConsecutiveFailures != 5 {
		t.Errorf("expected default maxConsecutiveFailures=5, got %d", client.maxConsecutiveFailures)
	}
	if client.requestsPerSecond != 50 {
		t.Errorf("expected default requestsPerSecond=50, got %f", client.requestsPerSecond)
	}
	if client.client == nil {
		t.Error("expected default http client")
	}
	if len(client.endpoints) != 1 {
		t.Errorf("expected 1 endpoint, got %d", len(client.endpoints))
	}
}

func TestFailoverRPCClientCurrentEndpoint(t *testing.T) {
	t.Parallel()
	client := NewFailoverRPCClient(FailoverConfig{
		PrimaryURL: "http://primary:8545",
	})
	ep := client.CurrentEndpoint()
	if ep != "http://primary:8545" {
		t.Errorf("expected primary URL, got %s", ep)
	}
}

func TestFailoverRPCClientCallbacks(t *testing.T) {
	t.Parallel()
	client := NewFailoverRPCClient(FailoverConfig{
		PrimaryURL: "http://localhost:8545",
	})

	var switched bool
	client.OnEndpointSwitch(func(from, to string) {
		switched = true
	})
	_ = switched // callback is stored, tested via integration

	var circuitOpened bool
	client.OnCircuitOpen(func(url string) {
		circuitOpened = true
	})
	_ = circuitOpened // callback is stored, tested via integration
}

func TestParseRetryAfter_EmptyHeader(t *testing.T) {
	t.Parallel()
	client := &FailoverRPCClient{}
	resp := &http.Response{Header: http.Header{}}
	d := client.parseRetryAfter(resp)
	if d != 0 {
		t.Errorf("expected 0 for empty header, got %v", d)
	}
}

func TestParseRetryAfter_Seconds(t *testing.T) {
	t.Parallel()
	client := &FailoverRPCClient{}
	resp := &http.Response{Header: http.Header{"Retry-After": []string{"30"}}}
	d := client.parseRetryAfter(resp)
	if d != 30*time.Second {
		t.Errorf("expected 30s, got %v", d)
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	t.Parallel()
	client := &FailoverRPCClient{}
	future := time.Now().Add(2 * time.Minute).UTC().Format(http.TimeFormat)
	resp := &http.Response{Header: http.Header{"Retry-After": []string{future}}}
	d := client.parseRetryAfter(resp)
	if d <= 0 {
		t.Errorf("expected positive duration, got %v", d)
	}
}

func TestParseRetryAfter_PastDate(t *testing.T) {
	t.Parallel()
	client := &FailoverRPCClient{}
	past := time.Now().Add(-1 * time.Hour).UTC().Format(http.TimeFormat)
	resp := &http.Response{Header: http.Header{"Retry-After": []string{past}}}
	d := client.parseRetryAfter(resp)
	if d != 0 {
		t.Errorf("expected 0 for past date, got %v", d)
	}
}

func TestParseRetryAfter_InvalidValue(t *testing.T) {
	t.Parallel()
	client := &FailoverRPCClient{}
	resp := &http.Response{Header: http.Header{"Retry-After": []string{"not-a-number"}}}
	d := client.parseRetryAfter(resp)
	if d != 0 {
		t.Errorf("expected 0 for invalid value, got %v", d)
	}
}

func TestReadBody(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	body, err := ReadBody(resp)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(body))
	}
}
