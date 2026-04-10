package resilience

import (
	"context"
	"testing"
	"time"
)

func TestDefaultCriticalErrorHandler_ReportCriticalError(t *testing.T) {
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	err := CriticalError{
		Type:        CriticalErrorTypeDataCorruption,
		Message:     "data corruption detected",
		Component:   "database",
		Details:     make(map[string]interface{}),
		Recoverable: false,
	}

	result := ceh.ReportCriticalError(ctx, err)
	if result != nil {
		t.Fatalf("ReportCriticalError failed: %v", result)
	}

	if !ceh.IsSafeMode(ctx) {
		t.Fatal("expected to be in safe mode after data corruption error")
	}
}

func TestDefaultCriticalErrorHandler_ReportCriticalError_InvalidError(t *testing.T) {
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	// Invalid error with empty message
	err := CriticalError{
		Type:      CriticalErrorTypeDataCorruption,
		Message:   "",
		Component: "database",
	}

	result := ceh.ReportCriticalError(ctx, err)
	if result == nil {
		t.Fatal("expected error for invalid critical error")
	}
}

func TestDefaultCriticalErrorHandler_GetCriticalErrors(t *testing.T) {
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	err := CriticalError{
		Type:        CriticalErrorTypeSystemFailure,
		Message:     "system failure",
		Component:   "core",
		Details:     make(map[string]interface{}),
		Recoverable: true,
	}

	_ = ceh.ReportCriticalError(ctx, err)

	errors, result := ceh.GetCriticalErrors(ctx)
	if result != nil {
		t.Fatalf("GetCriticalErrors failed: %v", result)
	}

	if len(errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(errors))
	}

	if errors[0].Message != "system failure" {
		t.Errorf("expected message 'system failure', got '%s'", errors[0].Message)
	}
}

func TestDefaultCriticalErrorHandler_GetCriticalErrors_NoErrors(t *testing.T) {
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	errors, result := ceh.GetCriticalErrors(ctx)
	if result == nil {
		t.Fatal("expected error when no critical errors recorded")
	}

	if errors != nil {
		t.Fatal("expected nil errors")
	}
}

func TestDefaultCriticalErrorHandler_GetAlerts(t *testing.T) {
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	err := CriticalError{
		Type:        CriticalErrorTypeSecurityBreach,
		Message:     "security breach detected",
		Component:   "api",
		Details:     make(map[string]interface{}),
		Recoverable: false,
	}

	_ = ceh.ReportCriticalError(ctx, err)

	alerts, result := ceh.GetAlerts(ctx)
	if result != nil {
		t.Fatalf("GetAlerts failed: %v", result)
	}

	if len(alerts) != 1 {
		t.Errorf("expected 1 alert, got %d", len(alerts))
	}

	if alerts[0].Severity != "critical" {
		t.Errorf("expected severity 'critical', got '%s'", alerts[0].Severity)
	}
}

func TestDefaultCriticalErrorHandler_AcknowledgeAlert(t *testing.T) {
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	err := CriticalError{
		Type:        CriticalErrorTypeResourceExhaustion,
		Message:     "resource exhaustion",
		Component:   "memory",
		Details:     make(map[string]interface{}),
		Recoverable: true,
	}

	_ = ceh.ReportCriticalError(ctx, err)

	result := ceh.AcknowledgeAlert(ctx, 0)
	if result != nil {
		t.Fatalf("AcknowledgeAlert failed: %v", result)
	}

	alerts, _ := ceh.GetAlerts(ctx)
	if !alerts[0].Notified {
		t.Fatal("expected alert to be notified")
	}
}

func TestDefaultCriticalErrorHandler_AcknowledgeAlert_InvalidIndex(t *testing.T) {
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	result := ceh.AcknowledgeAlert(ctx, 0)
	if result == nil {
		t.Fatal("expected error for invalid alert index")
	}
}

func TestDefaultCriticalErrorHandler_EnterSafeMode(t *testing.T) {
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	err := ceh.EnterSafeMode(ctx, "test reason")
	if err != nil {
		t.Fatalf("EnterSafeMode failed: %v", err)
	}

	if !ceh.IsSafeMode(ctx) {
		t.Fatal("expected to be in safe mode")
	}
}

func TestDefaultCriticalErrorHandler_EnterSafeMode_AlreadyInSafeMode(t *testing.T) {
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	_ = ceh.EnterSafeMode(ctx, "reason1")

	err := ceh.EnterSafeMode(ctx, "reason2")
	if err == nil {
		t.Fatal("expected error when already in safe mode")
	}
}

func TestDefaultCriticalErrorHandler_ExitSafeMode(t *testing.T) {
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	_ = ceh.EnterSafeMode(ctx, "test reason")

	err := ceh.ExitSafeMode(ctx)
	if err != nil {
		t.Fatalf("ExitSafeMode failed: %v", err)
	}

	if ceh.IsSafeMode(ctx) {
		t.Fatal("expected to not be in safe mode")
	}
}

func TestDefaultCriticalErrorHandler_ExitSafeMode_NotInSafeMode(t *testing.T) {
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	err := ceh.ExitSafeMode(ctx)
	if err == nil {
		t.Fatal("expected error when not in safe mode")
	}
}

func TestDefaultCriticalErrorHandler_Health(t *testing.T) {
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	health := ceh.Health(ctx)
	if health.Status != "healthy" {
		t.Errorf("expected healthy status, got %s", health.Status)
	}

	// Report error and enter safe mode
	err := CriticalError{
		Type:        CriticalErrorTypeDataCorruption,
		Message:     "data corruption",
		Component:   "database",
		Details:     make(map[string]interface{}),
		Recoverable: false,
	}

	_ = ceh.ReportCriticalError(ctx, err)

	health = ceh.Health(ctx)
	if health.Status != "degraded" {
		t.Errorf("expected degraded status, got %s", health.Status)
	}
}

func TestDefaultDataCorruptionDetector_VerifyIntegrity(t *testing.T) {
	dcd := NewDefaultDataCorruptionDetector()
	ctx := context.Background()

	// Valid data
	result := dcd.VerifyIntegrity(ctx, "test data")
	if !result {
		t.Fatal("expected integrity check to pass for valid data")
	}

	// Nil data
	result = dcd.VerifyIntegrity(ctx, nil)
	if result {
		t.Fatal("expected integrity check to fail for nil data")
	}
}

func TestDefaultDataCorruptionDetector_GetCorruptionStats(t *testing.T) {
	dcd := NewDefaultDataCorruptionDetector()
	ctx := context.Background()

	// Perform some checks
	dcd.VerifyIntegrity(ctx, "data1")
	dcd.VerifyIntegrity(ctx, "data2")
	dcd.VerifyIntegrity(ctx, nil)

	stats := dcd.GetCorruptionStats(ctx)
	if stats == nil {
		t.Fatal("stats is nil")
	}

	if stats["integrity_check_count"] != int64(3) {
		t.Errorf("expected 3 integrity checks, got %v", stats["integrity_check_count"])
	}

	if stats["failed_integrity_checks"] != int64(1) {
		t.Errorf("expected 1 failed integrity check, got %v", stats["failed_integrity_checks"])
	}
}

func TestDefaultCriticalErrorAlerter_SendAlert(t *testing.T) {
	cea := NewDefaultCriticalErrorAlerter(10)
	ctx := context.Background()

	alert := CriticalErrorAlert{
		Error: CriticalError{
			Type:      CriticalErrorTypeDataCorruption,
			Message:   "data corruption",
			Component: "database",
			Details:   make(map[string]interface{}),
		},
		AlertTime: time.Now(),
		Severity:  "critical",
		Action:    "enter_safe_mode",
		Notified:  false,
	}

	err := cea.SendAlert(ctx, alert)
	if err != nil {
		t.Fatalf("SendAlert failed: %v", err)
	}
}

func TestDefaultCriticalErrorAlerter_SendAlert_InvalidAlert(t *testing.T) {
	cea := NewDefaultCriticalErrorAlerter(10)
	ctx := context.Background()

	// Invalid alert with empty message
	alert := CriticalErrorAlert{
		Error: CriticalError{
			Type:      CriticalErrorTypeDataCorruption,
			Message:   "",
			Component: "database",
		},
		AlertTime: time.Now(),
		Severity:  "critical",
	}

	err := cea.SendAlert(ctx, alert)
	if err == nil {
		t.Fatal("expected error for invalid alert")
	}
}

func TestDefaultCriticalErrorAlerter_GetAlertHistory(t *testing.T) {
	cea := NewDefaultCriticalErrorAlerter(10)
	ctx := context.Background()

	alert := CriticalErrorAlert{
		Error: CriticalError{
			Type:      CriticalErrorTypeSystemFailure,
			Message:   "system failure",
			Component: "core",
			Details:   make(map[string]interface{}),
		},
		AlertTime: time.Now(),
		Severity:  "critical",
		Action:    "restart",
	}

	_ = cea.SendAlert(ctx, alert)

	history, err := cea.GetAlertHistory(ctx)
	if err != nil {
		t.Fatalf("GetAlertHistory failed: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("expected 1 alert in history, got %d", len(history))
	}
}

func TestDefaultCriticalErrorAlerter_GetAlertStats(t *testing.T) {
	cea := NewDefaultCriticalErrorAlerter(10)
	ctx := context.Background()

	alert := CriticalErrorAlert{
		Error: CriticalError{
			Type:      CriticalErrorTypeSecurityBreach,
			Message:   "security breach",
			Component: "api",
			Details:   make(map[string]interface{}),
		},
		AlertTime: time.Now(),
		Severity:  "critical",
		Action:    "isolate",
	}

	_ = cea.SendAlert(ctx, alert)

	stats := cea.GetAlertStats(ctx)
	if stats == nil {
		t.Fatal("stats is nil")
	}

	if stats["alerts_sent"] != int64(1) {
		t.Errorf("expected 1 alert sent, got %v", stats["alerts_sent"])
	}
}

func TestCriticalErrorHandler_MultipleErrors(t *testing.T) {
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	// Report multiple errors
	for i := 0; i < 5; i++ {
		err := CriticalError{
			Type:        CriticalErrorTypeSystemFailure,
			Message:     "system failure",
			Component:   "core",
			Details:     make(map[string]interface{}),
			Recoverable: true,
		}

		_ = ceh.ReportCriticalError(ctx, err)
	}

	errors, _ := ceh.GetCriticalErrors(ctx)
	if len(errors) != 5 {
		t.Errorf("expected 5 errors, got %d", len(errors))
	}
}

func TestCriticalErrorHandler_ErrorTypeTracking(t *testing.T) {
	ceh := NewDefaultCriticalErrorHandler(10)
	ctx := context.Background()

	// Report different error types
	errorTypes := []CriticalErrorType{
		CriticalErrorTypeDataCorruption,
		CriticalErrorTypeSystemFailure,
		CriticalErrorTypeSecurityBreach,
		CriticalErrorTypeResourceExhaustion,
	}

	for _, errorType := range errorTypes {
		err := CriticalError{
			Type:        errorType,
			Message:     "test error",
			Component:   "test",
			Details:     make(map[string]interface{}),
			Recoverable: true,
		}

		_ = ceh.ReportCriticalError(ctx, err)
	}

	health := ceh.Health(ctx)
	if health.Details["data_corruption_detected"] != int64(1) {
		t.Errorf("expected 1 data corruption error")
	}

	if health.Details["system_failure_detected"] != int64(1) {
		t.Errorf("expected 1 system failure error")
	}

	if health.Details["security_breach_detected"] != int64(1) {
		t.Errorf("expected 1 security breach error")
	}

	if health.Details["resource_exhaustion_count"] != int64(1) {
		t.Errorf("expected 1 resource exhaustion error")
	}
}

func TestCriticalErrorHandler_ConcurrentOperations(t *testing.T) {
	ceh := NewDefaultCriticalErrorHandler(100)
	ctx := context.Background()

	// First, report errors sequentially to ensure they're recorded
	for i := 0; i < 10; i++ {
		err := CriticalError{
			Type:        CriticalErrorTypeSystemFailure,
			Message:     "system failure",
			Component:   "core",
			Details:     make(map[string]interface{}),
			Recoverable: true,
		}
		result := ceh.ReportCriticalError(ctx, err)
		if result != nil {
			t.Fatalf("failed to report error: %v", result)
		}
	}

	done := make(chan error, 10)

	// Concurrent reads after all errors are recorded
	for i := 0; i < 10; i++ {
		go func() {
			_, err := ceh.GetCriticalErrors(ctx)
			done <- err
		}()
	}

	for i := 0; i < 10; i++ {
		err := <-done
		if err != nil {
			t.Fatalf("concurrent read operation failed: %v", err)
		}
	}
	close(done)
}
