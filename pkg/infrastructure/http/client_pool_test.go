package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSharedHTTPClientDefaults(t *testing.T) {
	client := NewSharedHTTPClient()
	assert.NotNil(t, client)
	assert.NotNil(t, client.Client())
	assert.Equal(t, 30*time.Second, client.Client().Timeout)
}

func TestSharedHTTPClientWithOptions(t *testing.T) {
	client := NewSharedHTTPClient(
		WithMaxIdleConnsPerHost(50),
		WithMaxConnsPerHost(100),
		WithMaxIdleConns(200),
		WithIdleConnTimeout(60*time.Second),
	)
	assert.NotNil(t, client)

	transport, ok := client.Client().Transport.(*http.Transport)
	require.True(t, ok)
	assert.Equal(t, 50, transport.MaxIdleConnsPerHost)
	assert.Equal(t, 100, transport.MaxConnsPerHost)
	assert.Equal(t, 200, transport.MaxIdleConns)
	assert.Equal(t, 60*time.Second, transport.IdleConnTimeout)
}

func TestSharedHTTPClientDoRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result":"0x1"}`))
	}))
	defer server.Close()

	client := NewSharedHTTPClient()
	req, err := http.NewRequest("POST", server.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDefaultSharedHTTPClient(t *testing.T) {
	assert.NotNil(t, DefaultSharedHTTPClient)
	assert.Same(t, DefaultSharedHTTPClient, DefaultSharedHTTPClient, "should be a singleton")
}

func TestSharedHTTPClientConnectionReuse(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewSharedHTTPClient(WithMaxIdleConnsPerHost(5))

	// Make multiple requests — they should reuse the same TCP connection
	for i := 0; i < 3; i++ {
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
	}

	assert.Equal(t, 3, callCount, "all requests should reach the server")
}

func TestSharedHTTPClientCloseIdleConnections(t *testing.T) {
	client := NewSharedHTTPClient()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Make a request to establish an idle connection
	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	// Close idle connections — should not panic
	client.CloseIdleConnections()

	// Can still make requests after closing idle connections
	req2, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)
	resp2, err := client.Do(req2)
	require.NoError(t, err)
	resp2.Body.Close()

	// Calling CloseIdleConnections multiple times is safe
	client.CloseIdleConnections()
	client.CloseIdleConnections()
}
