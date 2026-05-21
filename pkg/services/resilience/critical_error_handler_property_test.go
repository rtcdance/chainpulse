package resilience

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Property 21: Critical Error Safety
// Validates that critical errors are handled safely and prevent data corruption

func TestProperty_CriticalErrorSafety_ErrorReporting(t *testing.T) {
	t.Parallel()
	// Property: All reported critical errors must be retrievable
	ceh := NewDefaultCriticalErrorHandler(100)
	ctx := context.Background()

	for trial := 0; trial < 50; trial++ {
		err := CriticalError{
			Type:        CriticalErrorTypeSystemFailure,
			Message:     fmt.Sprintf("error_%d", trial),
			Component:   "test",
			Details:     make(map[string]any),
			Recoverable: true,
		}

		result := ceh.ReportCriticalError(ctx, err)
		if result != nil {
			t.Fatalf("trial %d: ReportCriticalError failed: %v", trial, result)
		}
	}

	errors, _ := ceh.GetCriticalErrors(ctx)
	if len(errors) != 50 {
		t.Errorf("expected 50 errors, got %d", len(errors))
	}
}

func TestProperty_CriticalErrorSafety_SafeModeTransitions(t *testing.T) {
	t.Parallel()
	// Property: Safe mode transitions must be valid
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	for trial := 0; trial < 50; trial++ {
		// Enter safe mode
		err := ceh.EnterSafeMode(ctx, fmt.Sprintf("reason_%d", trial))
		if err != nil {
			t.Fatalf("trial %d: EnterSafeMode failed: %v", trial, err)
		}

		if !ceh.IsSafeMode(ctx) {
			t.Fatalf("trial %d: expected to be in safe mode", trial)
		}

		// Exit safe mode
		err = ceh.ExitSafeMode(ctx)
		if err != nil {
			t.Fatalf("trial %d: ExitSafeMode failed: %v", trial, err)
		}

		if ceh.IsSafeMode(ctx) {
			t.Fatalf("trial %d: expected to not be in safe mode", trial)
		}
	}
}

func TestProperty_CriticalErrorSafety_DataCorruptionDetection(t *testing.T) {
	t.Parallel()
	// Property: Data corruption errors must trigger safe mode
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	for trial := 0; trial < 50; trial++ {
		err := CriticalError{
			Type:        CriticalErrorTypeDataCorruption,
			Message:     fmt.Sprintf("corruption_%d", trial),
			Component:   "database",
			Details:     make(map[string]any),
			Recoverable: false,
		}

		_ = ceh.ReportCriticalError(ctx, err)

		if !ceh.IsSafeMode(ctx) {
			t.Fatalf("trial %d: expected to be in safe mode after data corruption", trial)
		}

		// Exit safe mode for next iteration
		_ = ceh.ExitSafeMode(ctx)
	}
}

func TestProperty_CriticalErrorSafety_AlertGeneration(t *testing.T) {
	t.Parallel()
	// Property: Each critical error must generate an alert
	ceh := NewDefaultCriticalErrorHandler(100)
	ctx := context.Background()

	for trial := 0; trial < 50; trial++ {
		err := CriticalError{
			Type:        CriticalErrorTypeSystemFailure,
			Message:     fmt.Sprintf("failure_%d", trial),
			Component:   "core",
			Details:     make(map[string]any),
			Recoverable: true,
		}

		_ = ceh.ReportCriticalError(ctx, err)
	}

	alerts, _ := ceh.GetAlerts(ctx)
	if len(alerts) != 50 {
		t.Errorf("expected 50 alerts, got %d", len(alerts))
	}
}

func TestProperty_CriticalErrorSafety_AlertAcknowledgment(t *testing.T) {
	t.Parallel()
	// Property: Acknowledged alerts must be marked as notified
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	for trial := 0; trial < 10; trial++ {
		err := CriticalError{
			Type:        CriticalErrorTypeResourceExhaustion,
			Message:     fmt.Sprintf("exhaustion_%d", trial),
			Component:   "memory",
			Details:     make(map[string]any),
			Recoverable: true,
		}

		if err := ceh.ReportCriticalError(ctx, err); err != nil {
			t.Fatalf("trial %d: ReportCriticalError failed: %v", trial, err)
		}

		result := ceh.AcknowledgeAlert(ctx, trial)
		if result != nil {
			t.Fatalf("trial %d: AcknowledgeAlert failed: %v", trial, result)
		}
	}

	alerts, _ := ceh.GetAlerts(ctx)
	for i, alert := range alerts {
		if !alert.Notified {
			t.Errorf("alert %d: expected to be notified", i)
		}
	}
}

func TestProperty_CriticalErrorSafety_ErrorTypeTracking(t *testing.T) {
	t.Parallel()
	// Property: Error types must be tracked accurately
	ceh := NewDefaultCriticalErrorHandler(100)
	ctx := context.Background()

	errorCounts := map[CriticalErrorType]int{
		CriticalErrorTypeDataCorruption:     10,
		CriticalErrorTypeSystemFailure:      20,
		CriticalErrorTypeSecurityBreach:     15,
		CriticalErrorTypeResourceExhaustion: 5,
	}

	for errorType, count := range errorCounts {
		for i := 0; i < count; i++ {
			err := CriticalError{
				Type:        errorType,
				Message:     fmt.Sprintf("error_%s_%d", errorType, i),
				Component:   "test",
				Details:     make(map[string]any),
				Recoverable: true,
			}

			if err := ceh.ReportCriticalError(ctx, err); err != nil {
				t.Fatalf("Failed to report error: %v", err)
			}
		}
	}

	health := ceh.Health(ctx)

	if health.Details["data_corruption_detected"] != int64(10) {
		t.Errorf("expected 10 data corruption errors, got %v", health.Details["data_corruption_detected"])
	}

	if health.Details["system_failure_detected"] != int64(20) {
		t.Errorf("expected 20 system failure errors, got %v", health.Details["system_failure_detected"])
	}

	if health.Details["security_breach_detected"] != int64(15) {
		t.Errorf("expected 15 security breach errors, got %v", health.Details["security_breach_detected"])
	}

	if health.Details["resource_exhaustion_count"] != int64(5) {
		t.Errorf("expected 5 resource exhaustion errors, got %v", health.Details["resource_exhaustion_count"])
	}
}

func TestProperty_CriticalErrorSafety_HealthStatus(t *testing.T) {
	t.Parallel()
	// Property: Health status must reflect safe mode state
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	// Without safe mode
	health := ceh.Health(ctx)
	if health.Status != "healthy" {
		t.Errorf("expected healthy status, got %s", health.Status)
	}

	// With safe mode
	_ = ceh.EnterSafeMode(ctx, "test reason")
	health = ceh.Health(ctx)
	if health.Status != "degraded" {
		t.Errorf("expected degraded status, got %s", health.Status)
	}
}

func TestProperty_CriticalErrorSafety_DataCorruptionDetector(t *testing.T) {
	t.Parallel()
	// Property: Data corruption detector must verify integrity correctly
	dcd := NewDefaultDataCorruptionDetector()
	ctx := context.Background()

	for trial := 0; trial < 50; trial++ {
		// Valid data
		result := dcd.VerifyIntegrity(ctx, fmt.Sprintf("data_%d", trial))
		if !result {
			t.Fatalf("trial %d: expected integrity check to pass for valid data", trial)
		}
	}

	stats := dcd.GetCorruptionStats(ctx)
	if stats["integrity_check_count"] != int64(50) {
		t.Errorf("expected 50 integrity checks, got %v", stats["integrity_check_count"])
	}

	if stats["failed_integrity_checks"] != int64(0) {
		t.Errorf("expected 0 failed integrity checks, got %v", stats["failed_integrity_checks"])
	}
}

func TestProperty_CriticalErrorSafety_AlerterHistory(t *testing.T) {
	t.Parallel()
	// Property: Alert history must maintain insertion order
	cea := NewDefaultCriticalErrorAlerter(100)
	ctx := context.Background()

	for trial := 0; trial < 50; trial++ {
		alert := CriticalErrorAlert{
			Error: CriticalError{
				Type:      CriticalErrorTypeSystemFailure,
				Message:   fmt.Sprintf("alert_%d", trial),
				Component: "test",
				Details:   make(map[string]any),
			},
			AlertTime: time.Now(),
			Severity:  "critical",
			Action:    "restart",
		}

		_ = cea.SendAlert(ctx, alert)
	}

	history, _ := cea.GetAlertHistory(ctx)
	if len(history) != 50 {
		t.Errorf("expected 50 alerts in history, got %d", len(history))
	}

	// Verify order
	for i := 0; i < len(history)-1; i++ {
		if history[i].AlertTime.After(history[i+1].AlertTime) {
			t.Errorf("alert order violated at index %d", i)
		}
	}
}

func TestProperty_CriticalErrorSafety_SeverityDetermination(t *testing.T) {
	t.Parallel()
	// Property: Error severity must be determined correctly
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	testCases := []struct {
		errorType CriticalErrorType
		severity  string
	}{
		{CriticalErrorTypeDataCorruption, "critical"},
		{CriticalErrorTypeSystemFailure, "critical"},
		{CriticalErrorTypeSecurityBreach, "critical"},
		{CriticalErrorTypeResourceExhaustion, "high"},
	}

	for _, tc := range testCases {
		err := CriticalError{
			Type:        tc.errorType,
			Message:     "test",
			Component:   "test",
			Details:     make(map[string]any),
			Recoverable: true,
		}

		if err := ceh.ReportCriticalError(ctx, err); err != nil {
			t.Fatalf("Failed to report error: %v", err)
		}
	}

	alerts, _ := ceh.GetAlerts(ctx)
	for i, alert := range alerts {
		if alert.Severity != testCases[i].severity {
			t.Errorf("alert %d: expected severity %s, got %s", i, testCases[i].severity, alert.Severity)
		}
	}
}

func TestProperty_CriticalErrorSafety_ActionDetermination(t *testing.T) {
	t.Parallel()
	// Property: Error action must be determined correctly
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	testCases := []struct {
		errorType CriticalErrorType
		action    string
	}{
		{CriticalErrorTypeDataCorruption, "enter_safe_mode_and_alert_operator"},
		{CriticalErrorTypeSystemFailure, "restart_service_and_alert_operator"},
		{CriticalErrorTypeSecurityBreach, "isolate_system_and_alert_security_team"},
		{CriticalErrorTypeResourceExhaustion, "scale_resources_and_alert_operator"},
	}

	for _, tc := range testCases {
		err := CriticalError{
			Type:        tc.errorType,
			Message:     "test",
			Component:   "test",
			Details:     make(map[string]any),
			Recoverable: true,
		}

		_ = ceh.ReportCriticalError(ctx, err)
	}

	alerts, _ := ceh.GetAlerts(ctx)
	for i, alert := range alerts {
		if alert.Action != testCases[i].action {
			t.Errorf("alert %d: expected action %s, got %s", i, testCases[i].action, alert.Action)
		}
	}
}

func TestProperty_CriticalErrorSafety_ConcurrentErrorReporting(t *testing.T) {
	t.Parallel()
	// Property: Concurrent error reporting must be safe
	ceh := NewDefaultCriticalErrorHandler(100)
	ctx := context.Background()

	done := make(chan error, 50)

	for i := 0; i < 50; i++ {
		go func(index int) {
			err := CriticalError{
				Type:        CriticalErrorTypeSystemFailure,
				Message:     fmt.Sprintf("error_%d", index),
				Component:   "test",
				Details:     make(map[string]any),
				Recoverable: true,
			}

			result := ceh.ReportCriticalError(ctx, err)
			done <- result
		}(i)
	}

	for i := 0; i < 50; i++ {
		err := <-done
		if err != nil {
			t.Fatalf("concurrent error reporting failed: %v", err)
		}
	}

	errors, _ := ceh.GetCriticalErrors(ctx)
	if len(errors) != 50 {
		t.Errorf("expected 50 errors, got %d", len(errors))
	}
}

func TestProperty_CriticalErrorSafety_ErrorHistoryLimit(t *testing.T) {
	t.Parallel()
	// Property: Error history must respect size limit
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	// Report more errors than the limit
	for i := 0; i < 20; i++ {
		err := CriticalError{
			Type:        CriticalErrorTypeSystemFailure,
			Message:     fmt.Sprintf("error_%d", i),
			Component:   "test",
			Details:     make(map[string]any),
			Recoverable: true,
		}

		_ = ceh.ReportCriticalError(ctx, err)
	}

	errors, _ := ceh.GetCriticalErrors(ctx)
	if len(errors) > 10 {
		t.Errorf("error history exceeds limit: expected <= 10, got %d", len(errors))
	}
}

func TestProperty_CriticalErrorSafety_LastErrorTracking(t *testing.T) {
	t.Parallel()
	// Property: Last critical error must be tracked
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	for trial := 0; trial < 10; trial++ {
		err := CriticalError{
			Type:        CriticalErrorTypeSystemFailure,
			Message:     fmt.Sprintf("error_%d", trial),
			Component:   "test",
			Details:     make(map[string]any),
			Recoverable: true,
		}

		_ = ceh.ReportCriticalError(ctx, err)
	}

	health := ceh.Health(ctx)
	if health.Details["last_critical_error"] != "error_9" {
		t.Errorf("expected last error 'error_9', got %v", health.Details["last_critical_error"])
	}
}

func TestProperty_CriticalErrorSafety_AlerterHistoryLimit(t *testing.T) {
	t.Parallel()
	// Property: Alert history must respect size limit
	cea := NewDefaultCriticalErrorAlerter(10)
	ctx := context.Background()

	// Send more alerts than the limit
	for i := 0; i < 20; i++ {
		alert := CriticalErrorAlert{
			Error: CriticalError{
				Type:      CriticalErrorTypeSystemFailure,
				Message:   fmt.Sprintf("alert_%d", i),
				Component: "test",
				Details:   make(map[string]any),
			},
			AlertTime: time.Now(),
			Severity:  "critical",
			Action:    "restart",
		}

		_ = cea.SendAlert(ctx, alert)
	}

	history, _ := cea.GetAlertHistory(ctx)
	if len(history) > 10 {
		t.Errorf("alert history exceeds limit: expected <= 10, got %d", len(history))
	}
}
