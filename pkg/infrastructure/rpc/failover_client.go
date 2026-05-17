package rpc

import (
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
// with per-endpoint circuit breaking and request rate limiting.
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
	if err := f.acquireToken(); err != nil {
		if f.onRateLimited != nil {
			f.onRateLimited()
		}
		return nil, err
	}

	return f.doWithFailover(req, 0)
}

func (f *FailoverRPCClient) doWithFailover(req *http.Request, depth int) (*http.Response, error) {
	if depth > len(f.endpoints)*2 {
		return nil, fmt.Errorf("all RPC endpoints exhausted after %d attempts", depth)
	}

	ep := f.nextHealthyEndpoint()
	if ep == nil {
		return nil, fmt.Errorf("no healthy RPC endpoints available")
	}

	// Create a new request targeting the selected endpoint
	newReq, err := http.NewRequestWithContext(req.Context(), req.Method, ep.URL, req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for endpoint %s: %w", ep.URL, err)
	}
	newReq.Header = req.Header

	resp, err := f.client.Do(newReq)
	if err != nil {
		f.recordFailure(ep)
		return f.doWithFailover(req, depth+1)
	}

	// Handle 429 Too Many Requests
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := f.parseRetryAfter(resp)
		resp.Body.Close() //nolint:errcheck // error on close not actionable		// Wait for Retry-After duration, then retry
		if retryAfter > 0 && retryAfter <= 60*time.Second {
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(retryAfter):
			}
		}
		return f.doWithFailover(req, depth+1)
	}

	// Handle server errors (5xx) by trying next endpoint
	if resp.StatusCode >= 500 {
		resp.Body.Close() //nolint:errcheck // error on close not actionable
		f.recordFailure(ep)
		return f.doWithFailover(req, depth+1)
	}

	// Success
	f.recordSuccess(ep)
	return resp, nil
}

// nextHealthyEndpoint returns the next healthy endpoint using round-robin.
func (f *FailoverRPCClient) nextHealthyEndpoint() *EndpointHealth {
	f.mu.RLock()
	defer f.mu.RUnlock()

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

// acquireToken implements a simple token bucket rate limiter.
func (f *FailoverRPCClient) acquireToken() error {
	f.rateMu.Lock()
	defer f.rateMu.Unlock()

	now := time.Now()
	elapsed := now.Sub(f.lastRefill).Seconds()
	f.tokens += elapsed * f.requestsPerSecond
	if f.tokens > f.maxTokens {
		f.tokens = f.maxTokens
	}
	f.lastRefill = now

	if f.tokens < 1 {
		return fmt.Errorf("rate limit exceeded: %.0f req/s", f.requestsPerSecond)
	}

	f.tokens--
	return nil
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
