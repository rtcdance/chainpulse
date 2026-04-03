package shared

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"
)

// generateTestCertificate generates a self-signed certificate for testing
func generateTestCertificate(t *testing.T, certFile, keyFile string) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create certificate template
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

	// Create certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	// Write certificate to file
	certOut, err := os.Create(certFile)
	if err != nil {
		t.Fatalf("Failed to create cert file: %v", err)
	}
	defer func() {
		if err := certOut.Close(); err != nil {
			t.Logf("failed to close cert file: %v", err)
		}
	}()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		t.Fatalf("Failed to encode certificate: %v", err)
	}

	// Write private key to file
	keyOut, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	defer func() {
		if err := keyOut.Close(); err != nil {
			t.Logf("failed to close key file: %v", err)
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

func TestNewTLSManager(t *testing.T) {
	certFile := "test_cert.pem"
	keyFile := "test_key.pem"
	defer func() {
		if err := os.Remove(certFile); err != nil {
			t.Logf("failed to remove cert file: %v", err)
		}
	}()
	defer func() {
		if err := os.Remove(keyFile); err != nil {
			t.Logf("failed to remove key file: %v", err)
		}
	}()

	generateTestCertificate(t, certFile, keyFile)

	tm, err := NewTLSManager(certFile, keyFile)
	if err != nil {
		t.Fatalf("Failed to create TLS manager: %v", err)
	}

	if tm == nil {
		t.Fatal("TLS manager is nil")
	}

	if tm.certFile != certFile {
		t.Errorf("Expected cert file %s, got %s", certFile, tm.certFile)
	}

	if tm.keyFile != keyFile {
		t.Errorf("Expected key file %s, got %s", keyFile, tm.keyFile)
	}
}

func TestTLSManagerGetConfig(t *testing.T) {
	certFile := "test_cert.pem"
	keyFile := "test_key.pem"
	defer func() {
		if err := os.Remove(certFile); err != nil {
			t.Logf("failed to remove cert file: %v", err)
		}
	}()
	defer func() {
		if err := os.Remove(keyFile); err != nil {
			t.Logf("failed to remove key file: %v", err)
		}
	}()

	generateTestCertificate(t, certFile, keyFile)

	tm, err := NewTLSManager(certFile, keyFile)
	if err != nil {
		t.Fatalf("Failed to create TLS manager: %v", err)
	}

	config := tm.GetConfig()
	if config == nil {
		t.Fatal("TLS config is nil")
	}

	if config.MinVersion != tls.VersionTLS12 {
		t.Errorf("Expected MinVersion TLS 1.2, got %v", config.MinVersion)
	}

	if config.MaxVersion != tls.VersionTLS13 {
		t.Errorf("Expected MaxVersion TLS 1.3, got %v", config.MaxVersion)
	}

	if len(config.Certificates) == 0 {
		t.Fatal("No certificates in config")
	}
}

func TestTLSManagerReloadIfNeeded(t *testing.T) {
	certFile := "test_cert.pem"
	keyFile := "test_key.pem"
	defer func() {
		if err := os.Remove(certFile); err != nil {
			t.Logf("failed to remove cert file: %v", err)
		}
	}()
	defer func() {
		if err := os.Remove(keyFile); err != nil {
			t.Logf("failed to remove key file: %v", err)
		}
	}()

	generateTestCertificate(t, certFile, keyFile)

	tm, err := NewTLSManager(certFile, keyFile)
	if err != nil {
		t.Fatalf("Failed to create TLS manager: %v", err)
	}

	// Set short TTL
	tm.SetReloadTTL(100 * time.Millisecond)

	// Should not reload immediately
	err = tm.ReloadIfNeeded()
	if err != nil {
		t.Fatalf("Unexpected error on first reload check: %v", err)
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Should reload now
	err = tm.ReloadIfNeeded()
	if err != nil {
		t.Fatalf("Failed to reload certificates: %v", err)
	}
}

func TestTLSManagerMetrics(t *testing.T) {
	certFile := "test_cert.pem"
	keyFile := "test_key.pem"
	defer func() {
		if err := os.Remove(certFile); err != nil {
			t.Logf("failed to remove cert file: %v", err)
		}
	}()
	defer func() {
		if err := os.Remove(keyFile); err != nil {
			t.Logf("failed to remove key file: %v", err)
		}
	}()

	generateTestCertificate(t, certFile, keyFile)

	tm, err := NewTLSManager(certFile, keyFile)
	if err != nil {
		t.Fatalf("Failed to create TLS manager: %v", err)
	}

	metrics := tm.GetMetrics()
	if metrics == nil {
		t.Fatal("Metrics is nil")
	}

	reloads, ok := metrics["reloads"].(int64)
	if !ok {
		t.Fatal("Reloads metric not found or wrong type")
	}

	if reloads < 1 {
		t.Errorf("Expected at least 1 reload, got %d", reloads)
	}

	errors, ok := metrics["errors"].(int64)
	if !ok {
		t.Fatal("Errors metric not found or wrong type")
	}

	if errors != 0 {
		t.Errorf("Expected 0 errors, got %d", errors)
	}
}

func TestTLSManagerInvalidCertificate(t *testing.T) {
	certFile := "nonexistent_cert.pem"
	keyFile := "nonexistent_key.pem"

	_, err := NewTLSManager(certFile, keyFile)
	if err == nil {
		t.Fatal("Expected error for nonexistent certificate")
	}
}

func TestTLSManagerSetReloadTTL(t *testing.T) {
	certFile := "test_cert.pem"
	keyFile := "test_key.pem"
	defer func() {
		if err := os.Remove(certFile); err != nil {
			t.Logf("failed to remove cert file: %v", err)
		}
	}()
	defer func() {
		if err := os.Remove(keyFile); err != nil {
			t.Logf("failed to remove key file: %v", err)
		}
	}()

	generateTestCertificate(t, certFile, keyFile)

	tm, err := NewTLSManager(certFile, keyFile)
	if err != nil {
		t.Fatalf("Failed to create TLS manager: %v", err)
	}

	newTTL := 30 * time.Minute
	tm.SetReloadTTL(newTTL)

	// Verify TTL was set (by checking internal state through reload behavior)
	tm.mu.RLock()
	if tm.reloadTTL != newTTL {
		t.Errorf("Expected TTL %v, got %v", newTTL, tm.reloadTTL)
	}
	tm.mu.RUnlock()
}

func TestTLSManagerRuntimeMetricsReady(t *testing.T) {
	certFile := "test_cert.pem"
	keyFile := "test_key.pem"
	defer func() {
		if err := os.Remove(certFile); err != nil {
			t.Logf("failed to remove cert file: %v", err)
		}
	}()
	defer func() {
		if err := os.Remove(keyFile); err != nil {
			t.Logf("failed to remove key file: %v", err)
		}
	}()

	generateTestCertificate(t, certFile, keyFile)

	tm, err := NewTLSManager(certFile, keyFile)
	if err != nil {
		t.Fatalf("Failed to create TLS manager: %v", err)
	}

	metrics := tm.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "tls-ready" {
		t.Errorf("expected tls-ready coverage posture, got %v", metrics["coverage_posture"])
	}
	if metrics["certificate_posture"] != "tls-ready" {
		t.Errorf("expected tls-ready, got %v", metrics["certificate_posture"])
	}
	if metrics["reload_posture"] != "reload-fresh" {
		t.Errorf("expected reload-fresh, got %v", metrics["reload_posture"])
	}
	if metrics["reliability_hint"] != "tls runtime is ready and certificate reload posture is healthy" {
		t.Errorf("unexpected reliability hint: %v", metrics["reliability_hint"])
	}
}

func TestTLSManagerMetricsIncludesPostureFields(t *testing.T) {
	certFile := "test_cert.pem"
	keyFile := "test_key.pem"
	defer func() {
		if err := os.Remove(certFile); err != nil {
			t.Logf("failed to remove cert file: %v", err)
		}
	}()
	defer func() {
		if err := os.Remove(keyFile); err != nil {
			t.Logf("failed to remove key file: %v", err)
		}
	}()

	generateTestCertificate(t, certFile, keyFile)

	tm, err := NewTLSManager(certFile, keyFile)
	if err != nil {
		t.Fatalf("Failed to create TLS manager: %v", err)
	}

	metrics := tm.GetMetrics()
	if metrics["coverage_posture"] != "tls-ready" {
		t.Errorf("expected tls-ready coverage posture, got %v", metrics["coverage_posture"])
	}
	if metrics["certificate_posture"] != "tls-ready" {
		t.Errorf("expected tls-ready certificate posture, got %v", metrics["certificate_posture"])
	}
	if metrics["reload_posture"] != "reload-fresh" {
		t.Errorf("expected reload-fresh, got %v", metrics["reload_posture"])
	}
	if metrics["reliability_hint"] != "tls runtime is ready and certificate reload posture is healthy" {
		t.Errorf("unexpected reliability hint: %v", metrics["reliability_hint"])
	}
}

func TestTLSManagerRuntimeMetricsReloadDue(t *testing.T) {
	certFile := "test_cert.pem"
	keyFile := "test_key.pem"
	defer func() {
		if err := os.Remove(certFile); err != nil {
			t.Logf("failed to remove cert file: %v", err)
		}
	}()
	defer func() {
		if err := os.Remove(keyFile); err != nil {
			t.Logf("failed to remove key file: %v", err)
		}
	}()

	generateTestCertificate(t, certFile, keyFile)

	tm, err := NewTLSManager(certFile, keyFile)
	if err != nil {
		t.Fatalf("Failed to create TLS manager: %v", err)
	}

	tm.mu.Lock()
	tm.reloadTTL = time.Second
	tm.lastReload = time.Now().Add(-2 * time.Second)
	tm.mu.Unlock()

	metrics := tm.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "tls-ready" {
		t.Errorf("expected tls-ready coverage posture, got %v", metrics["coverage_posture"])
	}
	if metrics["reload_posture"] != "reload-due" {
		t.Errorf("expected reload-due, got %v", metrics["reload_posture"])
	}
}

func TestTLSManagerRuntimeMetricsUnobserved(t *testing.T) {
	tm := &TLSManager{
		reloadTTL: time.Hour,
		metrics:   &TLSMetrics{},
	}

	metrics := tm.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "tls-unconfigured" {
		t.Errorf("expected tls-unconfigured coverage posture, got %v", metrics["coverage_posture"])
	}
	if metrics["certificate_posture"] != "tls-unconfigured" {
		t.Errorf("expected tls-unconfigured, got %v", metrics["certificate_posture"])
	}
	if metrics["reload_posture"] != "reload-unobserved" {
		t.Errorf("expected reload-unobserved, got %v", metrics["reload_posture"])
	}
	if metrics["reliability_hint"] != "tls runtime is not configured yet" {
		t.Errorf("unexpected reliability hint: %v", metrics["reliability_hint"])
	}
}

func TestTLSManagerRuntimeMetricsDegraded(t *testing.T) {
	tm := &TLSManager{
		config:     &tls.Config{},
		lastReload: time.Now(),
		reloadTTL:  time.Hour,
		metrics: &TLSMetrics{
			reloads: 1,
			errors:  1,
		},
	}

	metrics := tm.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "tls-degraded" {
		t.Errorf("expected tls-degraded coverage posture, got %v", metrics["coverage_posture"])
	}
	if metrics["certificate_posture"] != "tls-degraded" {
		t.Errorf("expected tls-degraded certificate posture, got %v", metrics["certificate_posture"])
	}
	if metrics["reload_posture"] != "reload-error" {
		t.Errorf("expected reload-error, got %v", metrics["reload_posture"])
	}
	if metrics["reliability_hint"] != "tls runtime is degraded; inspect certificate loading and reload failures" {
		t.Errorf("unexpected reliability hint: %v", metrics["reliability_hint"])
	}
}
