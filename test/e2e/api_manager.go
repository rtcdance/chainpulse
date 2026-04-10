package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIManager manages API interactions for E2E tests
type APIManager struct {
	baseURL     string
	client      *http.Client
	headers     map[string]string
	initialized bool
}

// NewAPIManager creates a new API manager
func NewAPIManager(baseURL string) *APIManager {
	return &APIManager{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		headers: make(map[string]string),
	}
}

// Initialize initializes the API manager
func (am *APIManager) Initialize(ctx context.Context) error {
	if am.initialized {
		return fmt.Errorf("API manager already initialized")
	}

	// Set default headers
	am.headers["Content-Type"] = "application/json"
	am.headers["Accept"] = "application/json"

	am.initialized = true
	return nil
}

// Close closes the API manager
func (am *APIManager) Close() error {
	if am.client != nil {
		am.client.CloseIdleConnections()
	}
	am.initialized = false
	return nil
}

// SetHeader sets a header for all requests
func (am *APIManager) SetHeader(key, value string) {
	am.headers[key] = value
}

// GetRequest performs a GET request
func (am *APIManager) GetRequest(ctx context.Context, path string) ([]byte, int, error) {
	if !am.initialized {
		return nil, 0, fmt.Errorf("API manager not initialized")
	}

	url := am.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range am.headers {
		req.Header.Set(key, value)
	}

	resp, err := am.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to perform request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, resp.StatusCode, nil
}

// PostRequest performs a POST request
func (am *APIManager) PostRequest(ctx context.Context, path string, payload interface{}) ([]byte, int, error) {
	if !am.initialized {
		return nil, 0, fmt.Errorf("API manager not initialized")
	}

	url := am.baseURL + path

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range am.headers {
		req.Header.Set(key, value)
	}

	resp, err := am.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to perform request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// PutRequest performs a PUT request
func (am *APIManager) PutRequest(ctx context.Context, path string, payload interface{}) ([]byte, int, error) {
	if !am.initialized {
		return nil, 0, fmt.Errorf("API manager not initialized")
	}

	url := am.baseURL + path

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range am.headers {
		req.Header.Set(key, value)
	}

	resp, err := am.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to perform request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// DeleteRequest performs a DELETE request
func (am *APIManager) DeleteRequest(ctx context.Context, path string) ([]byte, int, error) {
	if !am.initialized {
		return nil, 0, fmt.Errorf("API manager not initialized")
	}

	url := am.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range am.headers {
		req.Header.Set(key, value)
	}

	resp, err := am.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to perform request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, resp.StatusCode, nil
}

// ParseJSONResponse parses a JSON response
func (am *APIManager) ParseJSONResponse(data []byte, v interface{}) error {
	err := json.Unmarshal(data, v)
	if err != nil {
		return fmt.Errorf("failed to parse JSON response: %w", err)
	}
	return nil
}

// IsHealthy checks if the API is healthy
func (am *APIManager) IsHealthy(ctx context.Context) bool {
	if !am.initialized {
		return false
	}

	_, statusCode, err := am.GetRequest(ctx, "/health")
	return err == nil && statusCode == http.StatusOK
}

// WaitForHealthy waits for the API to be healthy
func (am *APIManager) WaitForHealthy(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for API to be healthy: %w", ctx.Err())
		case <-ticker.C:
			if am.IsHealthy(ctx) {
				return nil
			}
		}
	}
}

// GetBaseURL returns the base URL
func (am *APIManager) GetBaseURL() string {
	return am.baseURL
}

// SetTimeout sets the request timeout
func (am *APIManager) SetTimeout(timeout time.Duration) {
	am.client.Timeout = timeout
}
