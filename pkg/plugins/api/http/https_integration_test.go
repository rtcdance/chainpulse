package http

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"

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
	defer func() {
		if err := certOut.Close(); err != nil {
			_ = err // Log but continue
		}
	}()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		t.Fatalf("Failed to encode certificate: %v", err)
	}

	keyOut, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	defer func() {
		if err := keyOut.Close(); err != nil {
			_ = err // Log but continue
		}
	}()

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to marshal private key: %v", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyBytes}); err != nil {
		t.Fatalf("Failed to encode private key: %v", err)
	}
}

func TestHTTPSConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	certFile := "test_https_cert.pem"
	keyFile := "test_https_key.pem"
	defer func() {
		_ = os.Remove(certFile)
		_ = os.Remove(keyFile)
	}()

	generateTestCertificate(t, certFile, keyFile)

	apiLayer := core.NewAPILayer()
	apiLayer.RegisterHandlerFunc("/test", func(req core.Request) (core.Response, error) {
		resp := core.NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetBody([]byte("OK"))
		return resp, nil
	})

	plugin, err := NewHTTPPluginWithTLS("https-test", 8080, 8443, certFile, keyFile, apiLayer)
	if err != nil {
		t.Fatalf("Failed to create HTTPS plugin: %v", err)
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

	// Create HTTPS client with insecure skip verify for testing
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	resp, err := client.Get("https://localhost:8443/test")
	if err != nil {
		t.Fatalf("Failed to make HTTPS request: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(body) != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", string(body))
	}
}

func TestHTTPAndHTTPSConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	certFile := "test_concurrent_cert.pem"
	keyFile := "test_concurrent_key.pem"
	defer func() {
		_ = os.Remove(certFile)
		_ = os.Remove(keyFile)
	}()

	generateTestCertificate(t, certFile, keyFile)

	apiLayer := core.NewAPILayer()
	apiLayer.RegisterHandlerFunc("/test", func(req core.Request) (core.Response, error) {
		resp := core.NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetBody([]byte("OK"))
		return resp, nil
	})

	plugin, err := NewHTTPPluginWithTLS("concurrent-test", 8081, 8444, certFile, keyFile, apiLayer)
	if err != nil {
		t.Fatalf("Failed to create HTTPS plugin: %v", err)
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

	// Test HTTP
	httpResp, err := http.Get("http://localhost:8081/test")
	if err != nil {
		t.Fatalf("Failed to make HTTP request: %v", err)
	}
	defer func() {
		if err := httpResp.Body.Close(); err != nil {
			t.Logf("failed to close response body: %v", err)
		}
	}()

	if httpResp.StatusCode != 200 {
		t.Errorf("Expected HTTP status 200, got %d", httpResp.StatusCode)
	}

	// Test HTTPS
	httpsClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	httpsResp, err := httpsClient.Get("https://localhost:8444/test")
	if err != nil {
		t.Fatalf("Failed to make HTTPS request: %v", err)
	}
	defer func() {
		if err := httpsResp.Body.Close(); err != nil {
			t.Logf("failed to close response body: %v", err)
		}
	}()

	if httpsResp.StatusCode != 200 {
		t.Errorf("Expected HTTPS status 200, got %d", httpsResp.StatusCode)
	}
}

func TestHTTPSPortConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	certFile := "test_port_cert.pem"
	keyFile := "test_port_key.pem"
	defer func() { _ = os.Remove(certFile) }()
	defer func() { _ = os.Remove(keyFile) }()

	generateTestCertificate(t, certFile, keyFile)

	apiLayer := core.NewAPILayer()
	plugin, err := NewHTTPPluginWithTLS("port-test", 8082, 8445, certFile, keyFile, apiLayer)
	if err != nil {
		t.Fatalf("Failed to create HTTPS plugin: %v", err)
	}

	if plugin.GetHTTPSPort() != 8445 {
		t.Errorf("Expected HTTPS port 8445, got %d", plugin.GetHTTPSPort())
	}

	plugin.SetHTTPSPort(8446)
	if plugin.GetHTTPSPort() != 8446 {
		t.Errorf("Expected HTTPS port 8446 after set, got %d", plugin.GetHTTPSPort())
	}
}

func TestTLSMetricsCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	certFile := "test_metrics_cert.pem"
	keyFile := "test_metrics_key.pem"
	defer func() { _ = os.Remove(certFile) }()
	defer func() { _ = os.Remove(keyFile) }()

	generateTestCertificate(t, certFile, keyFile)

	apiLayer := core.NewAPILayer()
	plugin, err := NewHTTPPluginWithTLS("metrics-test", 8083, 8447, certFile, keyFile, apiLayer)
	if err != nil {
		t.Fatalf("Failed to create HTTPS plugin: %v", err)
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

func TestHTTPPluginWithoutTLS(t *testing.T) {
	apiLayer := core.NewAPILayer()
	apiLayer.RegisterHandlerFunc("/test", func(req core.Request) (core.Response, error) {
		resp := core.NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetBody([]byte("OK"))
		return resp, nil
	})

	plugin := NewHTTPPlugin("no-tls-test", 8084, apiLayer)
	if err := plugin.Start(); err != nil {
		t.Fatalf("Failed to start plugin: %v", err)
	}
	defer func() {
		if err := plugin.Stop(); err != nil {
			t.Logf("failed to stop plugin: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:8084/test")
	if err != nil {
		t.Fatalf("Failed to make HTTP request: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify no TLS metrics when TLS is not configured
	metrics := plugin.GetTLSMetrics()
	if metrics != nil {
		t.Error("Expected nil TLS metrics when TLS is not configured")
	}
}
