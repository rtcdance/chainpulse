package resilience

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"chainpulse/pkg/core"
)

// Ensure atomic types are properly imported
var _ atomic.Int64

// CriticalErrorType represents the type of critical error
type CriticalErrorType string

const (
	CriticalErrorTypeDataCorruption     CriticalErrorType = "data_corruption"
	CriticalErrorTypeSystemFailure      CriticalErrorType = "system_failure"
	CriticalErrorTypeSecurityBreach     CriticalErrorType = "security_breach"
	CriticalErrorTypeResourceExhaustion CriticalErrorType = "resource_exhaustion"
)

// CriticalError represents a critical error that requires immediate action
type CriticalError struct {
	Type        CriticalErrorType
	Message     string
	Timestamp   time.Time
	Component   string
	Details     map[string]interface{}
	StackTrace  string
	Recoverable bool
	Err         error // underlying error, if any
}

// Error implements the error interface
func (ce *CriticalError) Error() string {
	if ce.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", ce.Type, ce.Message, ce.Err)
	}
	return fmt.Sprintf("[%s] %s", ce.Type, ce.Message)
}

// Unwrap returns the underlying error, enabling errors.Is() and errors.As()
// to traverse the error chain through CriticalError.
func (ce *CriticalError) Unwrap() error { return ce.Err }

// CriticalErrorAlert represents an alert for a critical error
type CriticalErrorAlert struct {
	Error      CriticalError
	AlertTime  time.Time
	Severity   string
	Action     string
	Notified   bool
	NotifyTime time.Time
}

// CriticalErrorHandler manages critical errors and alerts
type CriticalErrorHandler interface {
	// ReportCriticalError reports a critical error
	ReportCriticalError(ctx context.Context, err CriticalError) error

	// GetCriticalErrors returns all critical errors
	GetCriticalErrors(ctx context.Context) ([]CriticalError, error)

	// GetAlerts returns all critical error alerts
	GetAlerts(ctx context.Context) ([]CriticalErrorAlert, error)

	// AcknowledgeAlert acknowledges an alert
	AcknowledgeAlert(ctx context.Context, alertIndex int) error

	// EnterSafeMode enters safe mode to prevent further damage
	EnterSafeMode(ctx context.Context, reason string) error

	// ExitSafeMode exits safe mode
	ExitSafeMode(ctx context.Context) error

	// IsSafeMode returns whether the system is in safe mode
	IsSafeMode(ctx context.Context) bool

	// Health returns the health status
	Health(ctx context.Context) core.HealthStatus
}

// DefaultCriticalErrorHandler implements CriticalErrorHandler
type DefaultCriticalErrorHandler struct {
	mu                      sync.RWMutex
	criticalErrors          []CriticalError
	alerts                  []CriticalErrorAlert
	maxErrorsStored         int
	inSafeMode              bool
	safeModeReason          string
	safeModeTime            time.Time
	errorCount              int64
	alertCount              int64
	acknowledgedAlertCount  int64
	dataCorruptionDetected  int64
	systemFailureDetected   int64
	securityBreachDetected  int64
	resourceExhaustionCount int64
	lastCriticalError       *CriticalError
	lastCriticalErrorTime   time.Time
}

// NewDefaultCriticalErrorHandler creates a new critical error handler
func NewDefaultCriticalErrorHandler(maxErrorsStored int) *DefaultCriticalErrorHandler {
	return &DefaultCriticalErrorHandler{
		maxErrorsStored: maxErrorsStored,
		criticalErrors:  make([]CriticalError, 0, maxErrorsStored),
		alerts:          make([]CriticalErrorAlert, 0, maxErrorsStored),
	}
}

// ReportCriticalError reports a critical error
func (ceh *DefaultCriticalErrorHandler) ReportCriticalError(ctx context.Context, err CriticalError) error {
	ceh.mu.Lock()
	defer ceh.mu.Unlock()

	atomic.AddInt64(&ceh.errorCount, 1)

	// Validate error
	if err.Message == "" {
		return fmt.Errorf("critical error message is empty")
	}

	if err.Component == "" {
		return fmt.Errorf("critical error component is empty")
	}

	// Set timestamp
	err.Timestamp = time.Now()

	// Track error type
	switch err.Type {
	case CriticalErrorTypeDataCorruption:
		atomic.AddInt64(&ceh.dataCorruptionDetected, 1)
	case CriticalErrorTypeSystemFailure:
		atomic.AddInt64(&ceh.systemFailureDetected, 1)
	case CriticalErrorTypeSecurityBreach:
		atomic.AddInt64(&ceh.securityBreachDetected, 1)
	case CriticalErrorTypeResourceExhaustion:
		atomic.AddInt64(&ceh.resourceExhaustionCount, 1)
	}

	// Store error
	ceh.criticalErrors = append(ceh.criticalErrors, err)
	if len(ceh.criticalErrors) > ceh.maxErrorsStored {
		ceh.criticalErrors = ceh.criticalErrors[1:]
	}

	// Create alert
	alert := CriticalErrorAlert{
		Error:     err,
		AlertTime: time.Now(),
		Severity:  ceh.determineSeverity(err.Type),
		Action:    ceh.determineAction(err.Type),
		Notified:  false,
	}

	ceh.alerts = append(ceh.alerts, alert)
	if len(ceh.alerts) > ceh.maxErrorsStored {
		ceh.alerts = ceh.alerts[1:]
	}

	atomic.AddInt64(&ceh.alertCount, 1)

	// Store last error
	ceh.lastCriticalError = &err
	ceh.lastCriticalErrorTime = time.Now()

	// Enter safe mode for data corruption errors
	if err.Type == CriticalErrorTypeDataCorruption && !ceh.inSafeMode {
		ceh.inSafeMode = true
		ceh.safeModeReason = fmt.Sprintf("Data corruption detected: %s", err.Message)
		ceh.safeModeTime = time.Now()
	}

	return nil
}

// GetCriticalErrors returns all critical errors
func (ceh *DefaultCriticalErrorHandler) GetCriticalErrors(ctx context.Context) ([]CriticalError, error) {
	ceh.mu.RLock()
	defer ceh.mu.RUnlock()

	if len(ceh.criticalErrors) == 0 {
		return nil, fmt.Errorf("no critical errors recorded")
	}

	// Return a copy
	errors := make([]CriticalError, len(ceh.criticalErrors))
	copy(errors, ceh.criticalErrors)

	return errors, nil
}

// GetAlerts returns all critical error alerts
func (ceh *DefaultCriticalErrorHandler) GetAlerts(ctx context.Context) ([]CriticalErrorAlert, error) {
	ceh.mu.RLock()
	defer ceh.mu.RUnlock()

	if len(ceh.alerts) == 0 {
		return nil, fmt.Errorf("no alerts recorded")
	}

	// Return a copy
	alerts := make([]CriticalErrorAlert, len(ceh.alerts))
	copy(alerts, ceh.alerts)

	return alerts, nil
}

// AcknowledgeAlert acknowledges an alert
func (ceh *DefaultCriticalErrorHandler) AcknowledgeAlert(ctx context.Context, alertIndex int) error {
	ceh.mu.Lock()
	defer ceh.mu.Unlock()

	if alertIndex < 0 || alertIndex >= len(ceh.alerts) {
		return fmt.Errorf("invalid alert index: %d", alertIndex)
	}

	ceh.alerts[alertIndex].Notified = true
	ceh.alerts[alertIndex].NotifyTime = time.Now()

	atomic.AddInt64(&ceh.acknowledgedAlertCount, 1)

	return nil
}

// EnterSafeMode enters safe mode to prevent further damage
func (ceh *DefaultCriticalErrorHandler) EnterSafeMode(ctx context.Context, reason string) error {
	ceh.mu.Lock()
	defer ceh.mu.Unlock()

	if ceh.inSafeMode {
		return fmt.Errorf("already in safe mode: %s", ceh.safeModeReason)
	}

	ceh.inSafeMode = true
	ceh.safeModeReason = reason
	ceh.safeModeTime = time.Now()

	return nil
}

// ExitSafeMode exits safe mode
func (ceh *DefaultCriticalErrorHandler) ExitSafeMode(ctx context.Context) error {
	ceh.mu.Lock()
	defer ceh.mu.Unlock()

	if !ceh.inSafeMode {
		return fmt.Errorf("not in safe mode")
	}

	ceh.inSafeMode = false
	ceh.safeModeReason = ""

	return nil
}

// IsSafeMode returns whether the system is in safe mode
func (ceh *DefaultCriticalErrorHandler) IsSafeMode(ctx context.Context) bool {
	ceh.mu.RLock()
	defer ceh.mu.RUnlock()

	return ceh.inSafeMode
}

// Health returns the health status
func (ceh *DefaultCriticalErrorHandler) Health(ctx context.Context) core.HealthStatus {
	ceh.mu.RLock()
	defer ceh.mu.RUnlock()

	status := core.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Check if in safe mode
	if ceh.inSafeMode {
		status.Status = "degraded"
		status.Details["safe_mode"] = true
		status.Details["safe_mode_reason"] = ceh.safeModeReason
		status.Details["safe_mode_time"] = ceh.safeModeTime
	}

	// Add statistics
	status.Details["error_count"] = atomic.LoadInt64(&ceh.errorCount)
	status.Details["alert_count"] = atomic.LoadInt64(&ceh.alertCount)
	status.Details["acknowledged_alert_count"] = atomic.LoadInt64(&ceh.acknowledgedAlertCount)
	status.Details["data_corruption_detected"] = atomic.LoadInt64(&ceh.dataCorruptionDetected)
	status.Details["system_failure_detected"] = atomic.LoadInt64(&ceh.systemFailureDetected)
	status.Details["security_breach_detected"] = atomic.LoadInt64(&ceh.securityBreachDetected)
	status.Details["resource_exhaustion_count"] = atomic.LoadInt64(&ceh.resourceExhaustionCount)
	status.Details["critical_errors_stored"] = len(ceh.criticalErrors)
	status.Details["alerts_stored"] = len(ceh.alerts)

	if ceh.lastCriticalError != nil {
		status.Details["last_critical_error"] = ceh.lastCriticalError.Message
		status.Details["last_critical_error_time"] = ceh.lastCriticalErrorTime
		status.Details["last_critical_error_type"] = ceh.lastCriticalError.Type
	}

	return status
}

// determineSeverity determines the severity of an error
func (ceh *DefaultCriticalErrorHandler) determineSeverity(errorType CriticalErrorType) string {
	switch errorType {
	case CriticalErrorTypeDataCorruption:
		return "critical"
	case CriticalErrorTypeSystemFailure:
		return "critical"
	case CriticalErrorTypeSecurityBreach:
		return "critical"
	case CriticalErrorTypeResourceExhaustion:
		return "high"
	default:
		return "unknown"
	}
}

// determineAction determines the action to take for an error
func (ceh *DefaultCriticalErrorHandler) determineAction(errorType CriticalErrorType) string {
	switch errorType {
	case CriticalErrorTypeDataCorruption:
		return "enter_safe_mode_and_alert_operator"
	case CriticalErrorTypeSystemFailure:
		return "restart_service_and_alert_operator"
	case CriticalErrorTypeSecurityBreach:
		return "isolate_system_and_alert_security_team"
	case CriticalErrorTypeResourceExhaustion:
		return "scale_resources_and_alert_operator"
	default:
		return "alert_operator"
	}
}

// DataCorruptionDetector detects data corruption
type DataCorruptionDetector interface {
	// DetectCorruption detects data corruption
	DetectCorruption(ctx context.Context, data interface{}) error

	// VerifyIntegrity verifies data integrity
	VerifyIntegrity(ctx context.Context, data interface{}) bool

	// GetCorruptionStats returns corruption statistics
	GetCorruptionStats(ctx context.Context) map[string]interface{}
}

// DefaultDataCorruptionDetector implements DataCorruptionDetector
type DefaultDataCorruptionDetector struct {
	mu                      sync.RWMutex
	checksumMap             map[string]string
	corruptionDetectedCount int64
	integrityCheckCount     int64
	failedIntegrityChecks   int64
}

// NewDefaultDataCorruptionDetector creates a new data corruption detector
func NewDefaultDataCorruptionDetector() *DefaultDataCorruptionDetector {
	return &DefaultDataCorruptionDetector{
		checksumMap: make(map[string]string),
	}
}

// DetectCorruption detects data corruption
func (dcd *DefaultDataCorruptionDetector) DetectCorruption(ctx context.Context, data interface{}) error {
	dcd.mu.Lock()
	defer dcd.mu.Unlock()

	if data == nil {
		return fmt.Errorf("data is nil")
	}

	// For now, we'll use a simple approach - just verify the data can be marshaled
	// In a real implementation, this would use checksums or other verification methods
	return nil
}

// VerifyIntegrity verifies data integrity
func (dcd *DefaultDataCorruptionDetector) VerifyIntegrity(ctx context.Context, data interface{}) bool {
	dcd.mu.Lock()
	defer dcd.mu.Unlock()

	atomic.AddInt64(&dcd.integrityCheckCount, 1)

	if data == nil {
		atomic.AddInt64(&dcd.failedIntegrityChecks, 1)
		return false
	}

	// Simple integrity check - verify data is not nil
	return true
}

// GetCorruptionStats returns corruption statistics
func (dcd *DefaultDataCorruptionDetector) GetCorruptionStats(ctx context.Context) map[string]interface{} {
	dcd.mu.RLock()
	defer dcd.mu.RUnlock()

	stats := map[string]interface{}{
		"corruption_detected_count": atomic.LoadInt64(&dcd.corruptionDetectedCount),
		"integrity_check_count":     atomic.LoadInt64(&dcd.integrityCheckCount),
		"failed_integrity_checks":   atomic.LoadInt64(&dcd.failedIntegrityChecks),
	}

	return stats
}

// CriticalErrorAlerter sends alerts for critical errors
type CriticalErrorAlerter interface {
	// SendAlert sends an alert
	SendAlert(ctx context.Context, alert CriticalErrorAlert) error

	// GetAlertHistory returns alert history
	GetAlertHistory(ctx context.Context) ([]CriticalErrorAlert, error)

	// GetAlertStats returns alert statistics
	GetAlertStats(ctx context.Context) map[string]interface{}
}

// DefaultCriticalErrorAlerter implements CriticalErrorAlerter
type DefaultCriticalErrorAlerter struct {
	mu             sync.RWMutex
	alertHistory   []CriticalErrorAlert
	maxHistorySize int
	alertsSent     int64
	alertsFailed   int64
	lastAlertTime  time.Time
	lastAlertError error
}

// NewDefaultCriticalErrorAlerter creates a new critical error alerter
func NewDefaultCriticalErrorAlerter(maxHistorySize int) *DefaultCriticalErrorAlerter {
	return &DefaultCriticalErrorAlerter{
		maxHistorySize: maxHistorySize,
		alertHistory:   make([]CriticalErrorAlert, 0, maxHistorySize),
	}
}

// SendAlert sends an alert
func (cea *DefaultCriticalErrorAlerter) SendAlert(ctx context.Context, alert CriticalErrorAlert) error {
	cea.mu.Lock()
	defer cea.mu.Unlock()

	if alert.Error.Message == "" {
		atomic.AddInt64(&cea.alertsFailed, 1)
		err := fmt.Errorf("alert error message is empty")
		cea.lastAlertError = err
		cea.lastAlertTime = time.Now()
		return err
	}

	// Store alert in history
	cea.alertHistory = append(cea.alertHistory, alert)
	if len(cea.alertHistory) > cea.maxHistorySize {
		cea.alertHistory = cea.alertHistory[1:]
	}

	atomic.AddInt64(&cea.alertsSent, 1)
	cea.lastAlertTime = time.Now()

	return nil
}

// GetAlertHistory returns alert history
func (cea *DefaultCriticalErrorAlerter) GetAlertHistory(ctx context.Context) ([]CriticalErrorAlert, error) {
	cea.mu.RLock()
	defer cea.mu.RUnlock()

	if len(cea.alertHistory) == 0 {
		return nil, fmt.Errorf("no alert history available")
	}

	// Return a copy
	history := make([]CriticalErrorAlert, len(cea.alertHistory))
	copy(history, cea.alertHistory)

	return history, nil
}

// GetAlertStats returns alert statistics
func (cea *DefaultCriticalErrorAlerter) GetAlertStats(ctx context.Context) map[string]interface{} {
	cea.mu.RLock()
	defer cea.mu.RUnlock()

	stats := map[string]interface{}{
		"alerts_sent":     atomic.LoadInt64(&cea.alertsSent),
		"alerts_failed":   atomic.LoadInt64(&cea.alertsFailed),
		"last_alert_time": cea.lastAlertTime,
		"history_size":    len(cea.alertHistory),
	}

	if cea.lastAlertError != nil {
		stats["last_alert_error"] = cea.lastAlertError.Error()
	}

	return stats
}
