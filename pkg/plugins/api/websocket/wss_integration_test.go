package websocket

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"chainpulse/pkg/plugins/api/core"
)

// generateTestCertificate generates a self-signed certificate for testing
func generateTestCertificate(t *testing.T, certFile, keyFile string) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(24 * time.Hour),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		DNSNames: []string{"localhost", "127.0.0.1"},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	certOut, err := os.Create(certFile)
	if err != nil {
		t.Fatalf("Failed to create cert file: %v", err)
	}
	defer func() { _ = certOut.Close() }()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		t.Fatalf("Failed to encode certificate: %v", err)
	}

	keyOut, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	defer func() { _ = keyOut.Close() }()

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to marshal private key: %v", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyBytes}); err != nil {
		t.Fatalf("Failed to encode private key: %v", err)
	}
}

func TestWSSConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	certFile := "test_wss_cert.pem"
	keyFile := "test_wss_key.pem"
	defer func() { _ = os.Remove(certFile) }()
	defer func() { _ = os.Remove(keyFile) }()

	generateTestCertificate(t, certFile, keyFile)

	apiLayer := core.NewAPILayer()
	apiLayer.RegisterHandlerFunc("/ws", func(req core.Request) (core.Response, error) {
		resp := core.NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetBody([]byte("OK"))
		return resp, nil
	})

	plugin, err := NewWebSocketPluginWithTLS("wss-test", 8090, 8491, certFile, keyFile, apiLayer)
	if err != nil {
		t.Fatalf("Failed to create WSS plugin: %v", err)
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("Failed to start plugin: %v", err)
	}
	defer func() {
		if err := plugin.Stop(); err != nil {
			t.Logf("failed to stop plugin: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Create WSS client with insecure skip verify for testing
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	u := url.URL{Scheme: "wss", Host: "localhost:8491", Path: "/ws"}
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Failed to connect to WSS: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Send test message
	err = conn.WriteMessage(websocket.TextMessage, []byte("test"))
	if err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	// Read response
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	if string(message) != "OK" {
		t.Errorf("Expected message 'OK', got '%s'", string(message))
	}
}

func TestWSAndWSSConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	certFile := "test_ws_wss_cert.pem"
	keyFile := "test_ws_wss_key.pem"
	defer func() { _ = os.Remove(certFile) }()
	defer func() { _ = os.Remove(keyFile) }()

	generateTestCertificate(t, certFile, keyFile)

	apiLayer := core.NewAPILayer()
	apiLayer.RegisterHandlerFunc("/ws", func(req core.Request) (core.Response, error) {
		resp := core.NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetBody([]byte("OK"))
		return resp, nil
	})

	plugin, err := NewWebSocketPluginWithTLS("concurrent-wss-test", 8091, 8492, certFile, keyFile, apiLayer)
	if err != nil {
		t.Fatalf("Failed to create WSS plugin: %v", err)
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("Failed to start plugin: %v", err)
	}
	defer func() {
		if err := plugin.Stop(); err != nil {
			t.Logf("failed to stop plugin: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Test WS (unencrypted)
	u := url.URL{Scheme: "ws", Host: "localhost:8091", Path: "/ws"}
	wsConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Failed to connect to WS: %v", err)
	}
	defer func() {
		if err := wsConn.Close(); err != nil {
			t.Logf("failed to close connection: %v", err)
		}
	}()

	err = wsConn.WriteMessage(websocket.TextMessage, []byte("test"))
	if err != nil {
		t.Fatalf("Failed to write WS message: %v", err)
	}

	_, wsMessage, err := wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read WS message: %v", err)
	}

	if string(wsMessage) != "OK" {
		t.Errorf("Expected WS message 'OK', got '%s'", string(wsMessage))
	}

	// Test WSS (encrypted)
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	wssURL := url.URL{Scheme: "wss", Host: "localhost:8492", Path: "/ws"}
	wssConn, _, err := dialer.Dial(wssURL.String(), nil)
	if err != nil {
		t.Fatalf("Failed to connect to WSS: %v", err)
	}
	defer func() {
		if err := wssConn.Close(); err != nil {
			t.Logf("failed to close connection: %v", err)
		}
	}()

	err = wssConn.WriteMessage(websocket.TextMessage, []byte("test"))
	if err != nil {
		t.Fatalf("Failed to write WSS message: %v", err)
	}

	_, wssMessage, err := wssConn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read WSS message: %v", err)
	}

	if string(wssMessage) != "OK" {
		t.Errorf("Expected WSS message 'OK', got '%s'", string(wssMessage))
	}
}

func TestWSSPortConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	certFile := "test_wss_port_cert.pem"
	keyFile := "test_wss_port_key.pem"
	defer func() { _ = os.Remove(certFile) }()
	defer func() { _ = os.Remove(keyFile) }()

	generateTestCertificate(t, certFile, keyFile)

	apiLayer := core.NewAPILayer()
	plugin, err := NewWebSocketPluginWithTLS("port-wss-test", 8092, 8493, certFile, keyFile, apiLayer)
	if err != nil {
		t.Fatalf("Failed to create WSS plugin: %v", err)
	}

	if plugin.GetWSSPort() != 8493 {
		t.Errorf("Expected WSS port 8493, got %d", plugin.GetWSSPort())
	}

	plugin.SetWSSPort(8494)
	if plugin.GetWSSPort() != 8494 {
		t.Errorf("Expected WSS port 8494 after set, got %d", plugin.GetWSSPort())
	}
}

func TestWSSMetricsCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	certFile := "test_wss_metrics_cert.pem"
	keyFile := "test_wss_metrics_key.pem"
	defer func() { _ = os.Remove(certFile) }()
	defer func() { _ = os.Remove(keyFile) }()

	generateTestCertificate(t, certFile, keyFile)

	apiLayer := core.NewAPILayer()
	plugin, err := NewWebSocketPluginWithTLS("metrics-wss-test", 8093, 8495, certFile, keyFile, apiLayer)
	if err != nil {
		t.Fatalf("Failed to create WSS plugin: %v", err)
	}

	metrics := plugin.GetTLSMetrics()
	if metrics == nil {
		t.Fatal("TLS metrics is nil")
	}

	reloads, ok := metrics["reloads"].(int64)
	if !ok {
		t.Fatal("Reloads metric not found or wrong type")
	}

	if reloads < 1 {
		t.Errorf("Expected at least 1 reload, got %d", reloads)
	}
}

func TestWebSocketPluginWithoutTLS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	apiLayer := core.NewAPILayer()
	apiLayer.RegisterHandlerFunc("/ws", func(req core.Request) (core.Response, error) {
		resp := core.NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetBody([]byte("OK"))
		return resp, nil
	})

	plugin := NewWebSocketPlugin("no-tls-ws-test", 8094, apiLayer)
	if err := plugin.Start(); err != nil {
		t.Fatalf("Failed to start plugin: %v", err)
	}
	defer func() { _ = plugin.Stop() }()

	time.Sleep(100 * time.Millisecond)

	u := url.URL{Scheme: "ws", Host: "localhost:8094", Path: "/ws"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Failed to connect to WS: %v", err)
	}
	defer func() { _ = conn.Close() }()

	err = conn.WriteMessage(websocket.TextMessage, []byte("test"))
	if err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	if string(message) != "OK" {
		t.Errorf("Expected message 'OK', got '%s'", string(message))
	}

	// Verify no TLS metrics when TLS is not configured
	metrics := plugin.GetTLSMetrics()
	if metrics != nil {
		t.Error("Expected nil TLS metrics when TLS is not configured")
	}
}

func TestWSSClientCount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	certFile := "test_wss_client_count_cert.pem"
	keyFile := "test_wss_client_count_key.pem"
	defer func() { _ = os.Remove(certFile) }()
	defer func() { _ = os.Remove(keyFile) }()

	generateTestCertificate(t, certFile, keyFile)

	apiLayer := core.NewAPILayer()
	plugin, err := NewWebSocketPluginWithTLS("client-count-test", 8095, 8496, certFile, keyFile, apiLayer)
	if err != nil {
		t.Fatalf("Failed to create WSS plugin: %v", err)
	}

	if err := plugin.Start(); err != nil {
		t.Fatalf("Failed to start plugin: %v", err)
	}
	defer func() {
		if err := plugin.Stop(); err != nil {
			t.Logf("failed to stop plugin: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	if plugin.GetClientCount() != 0 {
		t.Errorf("Expected 0 clients initially, got %d", plugin.GetClientCount())
	}

	// Connect a client
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	u := url.URL{Scheme: "wss", Host: "localhost:8496", Path: "/ws"}
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Failed to connect to WSS: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	if plugin.GetClientCount() != 1 {
		t.Errorf("Expected 1 client after connection, got %d", plugin.GetClientCount())
	}

	if err := conn.Close(); err != nil {
		t.Logf("Warning: failed to close connection: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	if plugin.GetClientCount() != 0 {
		t.Errorf("Expected 0 clients after disconnection, got %d", plugin.GetClientCount())
	}
}
