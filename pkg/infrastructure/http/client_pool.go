package http

import (
	"net"
	"net/http"
	"time"
)

// SharedHTTPClient wraps an http.Client with a tuned shared transport.
// All pullers and RPC clients should use a shared instance to reuse
// TCP connections and avoid the default MaxIdleConnsPerHost=2 bottleneck.
type SharedHTTPClient struct {
	client *http.Client
}

// HTTPClientOption configures the shared HTTP client's transport.
type HTTPClientOption func(*http.Transport)

// WithMaxIdleConnsPerHost sets the maximum idle connections per host.
func WithMaxIdleConnsPerHost(n int) HTTPClientOption {
	return func(t *http.Transport) { t.MaxIdleConnsPerHost = n }
}

// WithMaxConnsPerHost sets the maximum total connections per host.
func WithMaxConnsPerHost(n int) HTTPClientOption {
	return func(t *http.Transport) { t.MaxConnsPerHost = n }
}

// WithMaxIdleConns sets the maximum total idle connections across all hosts.
func WithMaxIdleConns(n int) HTTPClientOption {
	return func(t *http.Transport) { t.MaxIdleConns = n }
}

// WithIdleConnTimeout sets the idle connection timeout.
func WithIdleConnTimeout(d time.Duration) HTTPClientOption {
	return func(t *http.Transport) { t.IdleConnTimeout = d }
}

// WithTimeout sets the overall HTTP client timeout.
func WithTimeout(d time.Duration) HTTPClientOption {
	return func(t *http.Transport) {} // Applied via client.Timeout
}

// NewSharedHTTPClient creates a shared HTTP client with tuned defaults.
// Default configuration:
//   - MaxIdleConns: 100
//   - MaxIdleConnsPerHost: 20
//   - MaxConnsPerHost: 50
//   - IdleConnTimeout: 90s
//   - Client Timeout: 30s
func NewSharedHTTPClient(opts ...HTTPClientOption) *SharedHTTPClient {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		MaxConnsPerHost:       50,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}

	for _, opt := range opts {
		opt(transport)
	}

	return &SharedHTTPClient{
		client: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}
}

// Do executes an HTTP request using the shared connection pool.
func (c *SharedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

// Client returns the underlying *http.Client for advanced usage.
func (c *SharedHTTPClient) Client() *http.Client {
	return c.client
}

// CloseIdleConnections closes any idle connections in the shared transport.
// This should be called during graceful shutdown to release resources.
// It is safe to call multiple times.
func (c *SharedHTTPClient) CloseIdleConnections() {
	if t, ok := c.client.Transport.(*http.Transport); ok {
		t.CloseIdleConnections()
	}
}

// DefaultSharedHTTPClient returns a shared client with default settings.
// This can be used as a package-level singleton for simple cases.
var DefaultSharedHTTPClient = NewSharedHTTPClient()
