package bootstrap

import (
	"strings"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/configmodel"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

func TestBuildAlertManagerLoadsEnabledRules(t *testing.T) {
	t.Parallel()
	manager := BuildAlertManager(testhelpers.NewTestLogger(), configmodel.AlertsConfig{
		Enabled: true,
		Rules: []configmodel.AlertRuleConfig{
			{
				Name:            "lag",
				Metric:          "indexing_lag_blocks",
				Operator:        ">",
				Threshold:       100,
				DurationSeconds: 30,
				Severity:        "warning",
				Message:         "indexing lag high",
			},
		},
	})
	names := manager.ListRules()
	if len(names) != 1 || names[0] != "lag" {
		t.Fatalf("ListRules = %v, want [lag]", names)
	}
}

func TestBuildAlertManagerSkipsInvalidRules(t *testing.T) {
	t.Parallel()
	manager := BuildAlertManager(testhelpers.NewTestLogger(), configmodel.AlertsConfig{
		Enabled: true,
		Rules: []configmodel.AlertRuleConfig{
			{Name: "bad-operator", Metric: "m", Operator: "?", Threshold: 1, DurationSeconds: 1, Severity: "info", Message: "x"},
			{Name: "good", Metric: "m", Operator: ">", Threshold: 1, DurationSeconds: 1, Severity: "info", Message: "x"},
			{Name: "", Metric: "m", Operator: ">", Threshold: 1, DurationSeconds: 1, Severity: "info", Message: "x"},
			{Name: "zero-duration", Metric: "m", Operator: ">", Threshold: 1, DurationSeconds: 0, Severity: "info", Message: "x"},
		},
	})
	names := manager.ListRules()
	if len(names) != 1 || names[0] != "good" {
		t.Fatalf("ListRules = %v, want [good] only", names)
	}
}

func TestBuildAlertManagerDisabledLoadsNoRules(t *testing.T) {
	t.Parallel()
	manager := BuildAlertManager(testhelpers.NewTestLogger(), configmodel.AlertsConfig{
		Enabled: false,
		Rules: []configmodel.AlertRuleConfig{
			{Name: "x", Metric: "m", Operator: ">", Threshold: 1, DurationSeconds: 1, Severity: "info", Message: "x"},
		},
	})
	if names := manager.ListRules(); len(names) != 0 {
		t.Fatalf("ListRules = %v, want none when alerts disabled", names)
	}
}

func TestBuildAlertManagerNilLoggerSafe(t *testing.T) {
	t.Parallel()
	if manager := BuildAlertManager(nil, configmodel.AlertsConfig{Enabled: true}); manager == nil {
		t.Fatal("expected manager even with nil logger")
	}
}

func TestBuildAlertManagerLogSinkFires(t *testing.T) {
	t.Parallel()
	logger := testhelpers.NewTestLoggerWithCapture()
	manager := BuildAlertManager(logger, configmodel.AlertsConfig{
		Enabled: true,
		Rules: []configmodel.AlertRuleConfig{
			{
				Name:            "lag",
				Metric:          "indexing_lag_blocks",
				Operator:        ">",
				Threshold:       10,
				DurationSeconds: 1,
				Severity:        "warning",
				Message:         "lag high",
			},
		},
	})

	manager.Evaluate("indexing_lag_blocks", 50)
	time.Sleep(1100 * time.Millisecond)
	manager.Evaluate("indexing_lag_blocks", 50)

	joined := strings.Join(logger.GetMessages(), "\n")
	if !strings.Contains(joined, "alert triggered") {
		t.Fatalf("expected log-sink warning, messages: %v", logger.GetMessages())
	}
}
