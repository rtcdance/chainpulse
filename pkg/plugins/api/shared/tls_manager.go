package shared

import (
	"crypto/tls"
	"fmt"
	"sync"
	"time"
)

// TLSManager manages TLS certificates and configurations
type TLSManager struct {
	certFile   string
	keyFile    string
	config     *tls.Config
	mu         sync.RWMutex
	lastReload time.Time
	reloadTTL  time.Duration
	metrics    *TLSMetrics
}

// TLSMetrics tracks TLS metrics
type TLSMetrics struct {
	reloads      int64
	errors       int64
	lastReloadAt time.Time
	mu           sync.RWMutex
}

// NewTLSManager creates a new TLS manager
func NewTLSManager(certFile, keyFile string) (*TLSManager, error) {
	tm := &TLSManager{
		certFile:  certFile,
		keyFile:   keyFile,
		reloadTTL: 1 * time.Hour,
		metrics:   &TLSMetrics{},
	}

	if err := tm.loadCertificates(); err != nil {
		return nil, err
	}

	return tm, nil
}

// loadCertificates loads TLS certificates
func (tm *TLSManager) loadCertificates() error {
	cert, err := tls.LoadX509KeyPair(tm.certFile, tm.keyFile)
	if err != nil {
		tm.recordError()
		return fmt.Errorf("failed to load certificates: %w", err)
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.config = &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
	}
	tm.lastReload = time.Now()
	tm.recordReload()

	return nil
}

// GetConfig returns the TLS configuration
func (tm *TLSManager) GetConfig() *tls.Config {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.config
}

// ReloadIfNeeded reloads certificates if TTL expired
func (tm *TLSManager) ReloadIfNeeded() error {
	tm.mu.RLock()
	needsReload := time.Since(tm.lastReload) >= tm.reloadTTL
	tm.mu.RUnlock()

	if !needsReload {
		return nil
	}

	return tm.loadCertificates()
}

// SetReloadTTL sets the reload TTL
func (tm *TLSManager) SetReloadTTL(ttl time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.reloadTTL = ttl
}

// GetMetrics returns TLS metrics
func (tm *TLSManager) GetMetrics() map[string]interface{} {
	tm.mu.RLock()
	configPresent := tm.config != nil
	reloadTTL := tm.reloadTTL
	lastReload := tm.lastReload
	tm.mu.RUnlock()

	tm.metrics.mu.RLock()
	reloads := tm.metrics.reloads
	errors := tm.metrics.errors
	lastReloadAt := tm.metrics.lastReloadAt
	tm.metrics.mu.RUnlock()

	certificatePosture := classifyTLSCertificatePosture(configPresent, errors)
	reloadPosture := classifyTLSReloadPosture(lastReload, reloadTTL, reloads, errors)

	return map[string]interface{}{
		"reloads":             reloads,
		"errors":              errors,
		"last_reload_at":      lastReloadAt,
		"coverage_posture":    certificatePosture,
		"certificate_posture": certificatePosture,
		"reload_posture":      reloadPosture,
		"reliability_hint":    buildTLSReliabilityHint(certificatePosture, reloadPosture),
	}
}

// GetRuntimeMetrics returns a compact runtime surface for TLS readiness and
// certificate reload posture.
func (tm *TLSManager) GetRuntimeMetrics() map[string]interface{} {
	tm.mu.RLock()
	configPresent := tm.config != nil
	reloadTTL := tm.reloadTTL
	tm.mu.RUnlock()

	tm.metrics.mu.RLock()
	tm.metrics.mu.RUnlock()
	metrics := tm.GetMetrics()

	return map[string]interface{}{
		"enabled":             configPresent,
		"reload_ttl":          reloadTTL.String(),
		"reloads":             metrics["reloads"],
		"errors":              metrics["errors"],
		"last_reload_at":      metrics["last_reload_at"],
		"coverage_posture":    metrics["coverage_posture"],
		"certificate_posture": metrics["certificate_posture"],
		"reload_posture":      metrics["reload_posture"],
		"reliability_hint":    metrics["reliability_hint"],
	}
}

// recordReload records a successful reload
func (tm *TLSManager) recordReload() {
	tm.metrics.mu.Lock()
	defer tm.metrics.mu.Unlock()
	tm.metrics.reloads++
	tm.metrics.lastReloadAt = time.Now()
}

// recordError records an error
func (tm *TLSManager) recordError() {
	tm.metrics.mu.Lock()
	defer tm.metrics.mu.Unlock()
	tm.metrics.errors++
}

func classifyTLSCertificatePosture(configPresent bool, errors int64) string {
	if !configPresent {
		return "tls-unconfigured"
	}
	if errors > 0 {
		return "tls-degraded"
	}
	return "tls-ready"
}

func classifyTLSReloadPosture(lastReload time.Time, reloadTTL time.Duration, reloads int64, errors int64) string {
	if reloads == 0 {
		return "reload-unobserved"
	}
	if errors > 0 {
		return "reload-error"
	}
	if reloadTTL > 0 && time.Since(lastReload) >= reloadTTL {
		return "reload-due"
	}
	return "reload-fresh"
}

func buildTLSReliabilityHint(certificatePosture string, reloadPosture string) string {
	switch {
	case certificatePosture == "tls-degraded" || reloadPosture == "reload-error":
		return "tls runtime is degraded; inspect certificate loading and reload failures"
	case certificatePosture == "tls-ready" && reloadPosture == "reload-due":
		return "tls runtime is ready but certificate reload is due; verify reload cadence"
	case certificatePosture == "tls-ready":
		return "tls runtime is ready and certificate reload posture is healthy"
	default:
		return "tls runtime is not configured yet"
	}
}
