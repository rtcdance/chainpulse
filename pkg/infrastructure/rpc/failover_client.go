package rpc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// EndpointHealth tracks the health of a single RPC endpoint.
type EndpointHealth struct {
	URL         string
	consecutive int64 // consecutive failures
	lastFailure time.Time
	circuitOpen bool // true = skip this endpoint
	openedAt    time.Time
}

// FailoverRPCClient rotates through RPC endpoints on failure,
// with per-endpoint circuit breaking, request rate limiting,
// and per-request timeout enforcement.
type FailoverRPCClient struct {
	mu        sync.RWMutex
	endpoints []*EndpointHealth
	client    *http.Client
	current   uint64 // atomic index into endpoints

	// Circuit breaker thresholds
	maxConsecutiveFailures int
	circuitResetTimeout    time.Duration

	// Rate limiter (token bucket)
	requestsPerSecond float64
	tokens            float64
	maxTokens         float64
	lastRefill        time.Time
	rateMu            sync.Mutex

	// Per-request timeout
	perRequestTimeout time.Duration

	// Metrics callback
	onEndpointSwitch func(from, to string)
	onCircuitOpen    func(url string)
	onRateLimited    func()
}

// FailoverConfig holds configuration for the FailoverRPCClient.
type FailoverConfig struct {
	PrimaryURL             string
	FallbackURLs           []string
	MaxConsecutiveFailures int           // default 5
	CircuitResetTimeout    time.Duration // default 30s
	RequestsPerSecond      float64       // default 50
	PerRequestTimeout      time.Duration // default 30s; applied per endpoint attempt
	HTTPClient             *http.Client
}

// NewFailoverRPCClient creates a new failover RPC client.
func NewFailoverRPCClient(cfg FailoverConfig) *FailoverRPCClient {
	if cfg.MaxConsecutiveFailures <= 0 {
		cfg.MaxConsecutiveFailures = 5
	}
	if cfg.CircuitResetTimeout <= 0 {
		cfg.CircuitResetTimeout = 30 * time.Second
	}
	if cfg.RequestsPerSecond <= 0 {
		cfg.RequestsPerSecond = 50
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}

	endpoints := []*EndpointHealth{
		{URL: cfg.PrimaryURL},
	}
	for _, url := range cfg.FallbackURLs {
		endpoints = append(endpoints, &EndpointHealth{URL: url})
	}

	return &FailoverRPCClient{
		endpoints:              endpoints,
		client:                 cfg.HTTPClient,
		maxConsecutiveFailures: cfg.MaxConsecutiveFailures,
		circuitResetTimeout:    cfg.CircuitResetTimeout,
		requestsPerSecond:      cfg.RequestsPerSecond,
		tokens:                 cfg.RequestsPerSecond,
		maxTokens:              cfg.RequestsPerSecond,
		lastRefill:             time.Now(),
		perRequestTimeout:      cfg.PerRequestTimeout,
	}
}

// OnEndpointSwitch registers a callback for endpoint switches.
func (f *FailoverRPCClient) OnEndpointSwitch(fn func(from, to string)) {
	f.onEndpointSwitch = fn
}

// OnCircuitOpen registers a callback for circuit open events.
func (f *FailoverRPCClient) OnCircuitOpen(fn func(url string)) {
	f.onCircuitOpen = fn
}

// Do executes an HTTP request with failover, rate limiting, and 429 handling.
func (f *FailoverRPCClient) Do(req *http.Request) (*http.Response, error) {
	if req.Context().Err() != nil {
		return nil, req.Context().Err()
	}

	if err := f.acquireTokenWithWait(req.Context()); err != nil {
		if f.onRateLimited != nil {
			f.onRateLimited()
		}
		return nil, err
	}

	// Read and buffer the request body so it can be reused across failover attempts.
	// Without buffering, the body stream is consumed on the first attempt and
	// subsequent retries to different endpoints will send an empty body.
	var bodyBytes []byte
	var err error
	if req.Body != nil {
		bodyBytes, err = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
	}

	return f.doWithFailover(req, bodyBytes, 0)
}

func (f *FailoverRPCClient) doWithFailover(req *http.Request, bodyBytes []byte, depth int) (*http.Response, error) {
	if depth > len(f.endpoints)*2 {
		return nil, fmt.Errorf("all RPC endpoints exhausted after %d attempts", depth)
	}

	ep := f.nextHealthyEndpoint()
	if ep == nil {
		return nil, fmt.Errorf("no healthy RPC endpoints available")
	}

	// Enforce per-request timeout: wrap the request context with a timeout
	// that is the minimum of the original context deadline and the configured
	// per-request timeout. This prevents a single slow endpoint from blocking
	// the entire pipeline indefinitely.
	ctx := req.Context()
	if f.perRequestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, f.perRequestTimeout)
		defer cancel()
	}

	// Create a new request targeting the selected endpoint with the timeout context.
	// Use the buffered body bytes so the body can be re-sent on failover.
	newReq, err := http.NewRequestWithContext(ctx, req.Method, ep.URL, io.NopCloser(bytes.NewReader(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request for endpoint %s: %w", ep.URL, err)
	}
	newReq.Header = req.Header

	resp, err := f.client.Do(newReq)
	if err != nil {
		f.recordFailure(ep)
		return f.doWithFailover(req, bodyBytes, depth+1)
	}

	// Handle 429 Too Many Requests: parse Retry-After header to determine
	// wait duration. If a short wait is indicated, sleep and retry the same
	// endpoint — this is critical for single-endpoint scenarios where
	// failing over is not an option and ignoring Retry-After causes
	// continued 429 responses.
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := f.parseRetryAfter(resp)
		resp.Body.Close() //nolint:errcheck

		// Only count as failure if this isn't the first 429 from this endpoint
		// (rate-limited but healthy endpoint should not be circuit-broken immediately)
		if depth > 0 {
			f.recordFailure(ep)
		}

		if retryAfter > 0 && retryAfter < 30*time.Second {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryAfter):
			}
		}
		return f.doWithFailover(req, bodyBytes, depth+1)
	}

	// Handle server errors (5xx) by trying next endpoint
	if resp.StatusCode >= 500 {
		resp.Body.Close() //nolint:errcheck // error on close not actionable
		f.recordFailure(ep)
		return f.doWithFailover(req, bodyBytes, depth+1)
	}

	// Success
	f.recordSuccess(ep)
	return resp, nil
}

// nextHealthyEndpoint returns the next healthy endpoint using round-robin.
func (f *FailoverRPCClient) nextHealthyEndpoint() *EndpointHealth {
	f.mu.Lock()
	defer f.mu.Unlock()

	n := len(f.endpoints)
	start := atomic.AddUint64(&f.current, 1) - 1

	for i := 0; i < n; i++ {
		idx := (int(start) + i) % n
		ep := f.endpoints[idx]

		// Check if circuit is open but has timed out (half-open)
		if ep.circuitOpen && time.Since(ep.openedAt) > f.circuitResetTimeout {
			// Allow one probe request (half-open)
			ep.circuitOpen = false
			ep.consecutive = 0
			return ep
		}

		if !ep.circuitOpen {
			return ep
		}
	}

	return nil
}

func (f *FailoverRPCClient) recordFailure(ep *EndpointHealth) {
	f.mu.Lock()
	defer f.mu.Unlock()

	ep.consecutive++
	ep.lastFailure = time.Now()

	if ep.consecutive >= int64(f.maxConsecutiveFailures) && !ep.circuitOpen {
		ep.circuitOpen = true
		ep.openedAt = time.Now()
		if f.onCircuitOpen != nil {
			f.onCircuitOpen(ep.URL)
		}
	}
}

func (f *FailoverRPCClient) recordSuccess(ep *EndpointHealth) {
	f.mu.Lock()
	defer f.mu.Unlock()

	ep.consecutive = 0
	ep.circuitOpen = false
}

// acquireTokenWithWait implements a token bucket rate limiter with bounded wait.
// When no token is available, it calculates the time until the next refill and
// sleeps rather than failing immediately. This prevents cascading failures
// during short bursts while still enforcing the configured rate limit.
// Waits at most 5 seconds before returning an error.
func (f *FailoverRPCClient) acquireTokenWithWait(ctx context.Context) error {
	const maxWait = 5 * time.Second
	deadline := time.Now().Add(maxWait)

	for {
		f.rateMu.Lock()
		now := time.Now()
		elapsed := now.Sub(f.lastRefill).Seconds()
		f.tokens += elapsed * f.requestsPerSecond
		if f.tokens > f.maxTokens {
			f.tokens = f.maxTokens
		}
		f.lastRefill = now

		if f.tokens >= 1 {
			f.tokens--
			f.rateMu.Unlock()
			return nil
		}

		waitTime := time.Duration((1.0 - f.tokens) / f.requestsPerSecond * float64(time.Second))
		f.rateMu.Unlock()

		if waitTime < time.Millisecond {
			waitTime = time.Millisecond
		}
		if time.Now().Add(waitTime).After(deadline) {
			return fmt.Errorf("rate limit exceeded: %.0f req/s (waited %v)", f.requestsPerSecond, time.Since(deadline.Add(-maxWait)).Round(time.Millisecond))
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}
	}
}

// parseRetryAfter parses the Retry-After header.
func (f *FailoverRPCClient) parseRetryAfter(resp *http.Response) time.Duration {
	val := resp.Header.Get("Retry-After")
	if val == "" {
		return 0
	}

	// Try seconds first
	if sec, err := strconv.Atoi(val); err == nil {
		return time.Duration(sec) * time.Second
	}

	// Try HTTP date
	if t, err := http.ParseTime(val); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}

	return 0
}

// HealthyEndpoints returns the number of healthy (non-circuit-open) endpoints.
// Includes endpoints whose circuit has timed out (half-open state).
func (f *FailoverRPCClient) HealthyEndpoints() int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	count := 0
	for _, ep := range f.endpoints {
		if !ep.circuitOpen || time.Since(ep.openedAt) > f.circuitResetTimeout {
			count++
		}
	}
	return count
}

// CurrentEndpoint returns the URL of the current primary endpoint.
func (f *FailoverRPCClient) CurrentEndpoint() string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	idx := int(atomic.LoadUint64(&f.current)) % len(f.endpoints)
	return f.endpoints[idx].URL
}

// ReadBody reads and closes the response body, returning its contents.
func ReadBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close() //nolint:errcheck // defer close
	return io.ReadAll(resp.Body)
}
