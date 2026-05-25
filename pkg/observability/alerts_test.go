package observability

import (
	"testing"
	"time"
)

func TestAlertRule_Validate_Success(t *testing.T) {
	t.Parallel()

	rule := AlertRule{
		Name:      "test-alert",
		Metric:    "cpu_usage",
		Operator:  ">",
		Threshold: 80.0,
		Duration:  5 * time.Minute,
		Severity:  AlertSeverityWarning,
	}

	err := rule.Validate()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestAlertRule_Validate_EmptyName(t *testing.T) {
	t.Parallel()

	rule := AlertRule{
		Metric:    "cpu_usage",
		Operator:  ">",
		Threshold: 80.0,
		Duration:  5 * time.Minute,
	}

	err := rule.Validate()
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestAlertRule_Validate_EmptyMetric(t *testing.T) {
	t.Parallel()

	rule := AlertRule{
		Name:      "test-alert",
		Operator:  ">",
		Threshold: 80.0,
		Duration:  5 * time.Minute,
	}

	err := rule.Validate()
	if err == nil {
		t.Error("expected error for empty metric")
	}
}

func TestAlertRule_Validate_InvalidOperator(t *testing.T) {
	t.Parallel()

	rule := AlertRule{
		Name:      "test-alert",
		Metric:    "cpu_usage",
		Operator:  "invalid",
		Threshold: 80.0,
		Duration:  5 * time.Minute,
	}

	err := rule.Validate()
	if err == nil {
		t.Error("expected error for invalid operator")
	}
}

func TestAlertRule_Validate_ZeroDuration(t *testing.T) {
	t.Parallel()

	rule := AlertRule{
		Name:      "test-alert",
		Metric:    "cpu_usage",
		Operator:  ">",
		Threshold: 80.0,
		Duration:  0,
	}

	err := rule.Validate()
	if err == nil {
		t.Error("expected error for zero duration")
	}
}

func TestAlertRule_Validate_ValidOperators(t *testing.T) {
	t.Parallel()

	operators := []string{">", ">=", "<", "<=", "==", "!="}
	for _, op := range operators {
		op := op
		t.Run(op, func(t *testing.T) {
			t.Parallel()
			rule := AlertRule{
				Name:      "test-alert",
				Metric:    "cpu_usage",
				Operator:  op,
				Threshold: 80.0,
				Duration:  5 * time.Minute,
			}
			if err := rule.Validate(); err != nil {
				t.Errorf("operator %s should be valid, got error: %v", op, err)
			}
		})
	}
}

func TestNewAlertManager(t *testing.T) {
	t.Parallel()

	am := NewAlertManager()
	if am == nil {
		t.Fatal("NewAlertManager returned nil")
	}
	if am.rules == nil {
		t.Error("rules map is nil")
	}
	if am.conditionStart == nil {
		t.Error("conditionStart map is nil")
	}
}

func TestAlertManager_AddRule(t *testing.T) {
	t.Parallel()

	am := NewAlertManager()
	rule := AlertRule{
		Name:      "cpu-alert",
		Metric:    "cpu_usage",
		Operator:  ">",
		Threshold: 90.0,
		Duration:  5 * time.Minute,
		Severity:  AlertSeverityCritical,
	}

	err := am.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	if len(am.rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(am.rules))
	}
}

func TestAlertManager_AddRule_Invalid(t *testing.T) {
	t.Parallel()

	am := NewAlertManager()
	err := am.AddRule(AlertRule{})
	if err == nil {
		t.Error("expected error for invalid rule")
	}
}

func TestAlertManager_RemoveRule(t *testing.T) {
	t.Parallel()

	am := NewAlertManager()
	rule := AlertRule{
		Name:      "cpu-alert",
		Metric:    "cpu_usage",
		Operator:  ">",
		Threshold: 90.0,
		Duration:  5 * time.Minute,
	}

	am.AddRule(rule)
	am.RemoveRule("cpu-alert")

	if len(am.rules) != 0 {
		t.Errorf("expected 0 rules after remove, got %d", len(am.rules))
	}
}

func TestAlertManager_RemoveRule_NotExist(t *testing.T) {
	t.Parallel()

	am := NewAlertManager()
	am.RemoveRule("nonexistent")
}

func TestAlertManager_GetRule(t *testing.T) {
	t.Parallel()

	am := NewAlertManager()
	rule := AlertRule{
		Name:      "mem-alert",
		Metric:    "memory_usage",
		Operator:  ">=",
		Threshold: 95.0,
		Duration:  3 * time.Minute,
	}

	am.AddRule(rule)

	got, ok := am.GetRule("mem-alert")
	if !ok {
		t.Error("expected rule to exist")
	}
	if got.Name != "mem-alert" {
		t.Errorf("Name = %s, want mem-alert", got.Name)
	}
}

func TestAlertManager_GetRule_NotFound(t *testing.T) {
	t.Parallel()

	am := NewAlertManager()
	_, ok := am.GetRule("nonexistent")
	if ok {
		t.Error("expected rule not found")
	}
}

func TestAlertManager_ListRules(t *testing.T) {
	t.Parallel()

	am := NewAlertManager()
	am.AddRule(AlertRule{
		Name: "rule1", Metric: "m1", Operator: ">", Threshold: 1, Duration: time.Minute,
	})
	am.AddRule(AlertRule{
		Name: "rule2", Metric: "m2", Operator: "<", Threshold: 2, Duration: time.Minute,
	})

	names := am.ListRules()
	if len(names) != 2 {
		t.Errorf("expected 2 rules, got %d", len(names))
	}
}

func TestAlertManager_AddCallback(t *testing.T) {
	t.Parallel()

	am := NewAlertManager()
	am.AddCallback(func(rule AlertRule, value float64, triggeredAt time.Time) {})

	if len(am.callbacks) != 1 {
		t.Errorf("expected 1 callback, got %d", len(am.callbacks))
	}
}

func TestAlertManager_EvaluateCondition(t *testing.T) {
	t.Parallel()

	am := NewAlertManager()

	tests := []struct {
		operator  string
		threshold float64
		value     float64
		want      bool
	}{
		{">", 50, 60, true},
		{">", 50, 50, false},
		{">", 50, 40, false},
		{">=", 50, 50, true},
		{">=", 50, 60, true},
		{">=", 50, 40, false},
		{"<", 50, 40, true},
		{"<", 50, 50, false},
		{"<", 50, 60, false},
		{"<=", 50, 50, true},
		{"<=", 50, 40, true},
		{"<=", 50, 60, false},
		{"==", 50, 50, true},
		{"==", 50, 51, false},
		{"!=", 50, 51, true},
		{"!=", 50, 50, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.operator, func(t *testing.T) {
			rule := AlertRule{Name: "test", Metric: "m", Operator: tt.operator, Threshold: tt.threshold, Duration: time.Minute}
			got := am.evaluateCondition(rule, tt.value)
			if got != tt.want {
				t.Errorf("evaluateCondition(operator=%s, threshold=%v, value=%v) = %v, want %v",
					tt.operator, tt.threshold, tt.value, got, tt.want)
			}
		})
	}
}

func TestAlertManager_Evaluate_NoMatch(t *testing.T) {
	t.Parallel()

	am := NewAlertManager()
	am.Evaluate("nonexistent_metric", 100)
}

func TestAlertManager_Evaluate_TriggersCallback(t *testing.T) {
	am := NewAlertManager()
	am.AddRule(AlertRule{
		Name:      "test-alert",
		Metric:    "cpu_usage",
		Operator:  ">",
		Threshold: 80.0,
		Duration:  1 * time.Minute,
		Severity:  AlertSeverityWarning,
	})

	var triggered bool
	am.AddCallback(func(rule AlertRule, value float64, triggeredAt time.Time) {
		triggered = true
	})

	am.mu.Lock()
	am.conditionStart["test-alert"] = time.Now().Add(-2 * time.Minute)
	am.mu.Unlock()

	am.Evaluate("cpu_usage", 90.0)

	if !triggered {
		t.Error("expected callback to be triggered")
	}
}

func TestAlertManager_Reset(t *testing.T) {
	t.Parallel()

	am := NewAlertManager()
	am.AddRule(AlertRule{
		Name: "test-alert", Metric: "cpu", Operator: ">", Threshold: 80, Duration: time.Minute,
	})
	am.Evaluate("cpu", 90)

	if len(am.conditionStart) == 0 {
		t.Error("conditionStart should have entries before reset")
	}

	am.Reset()

	if len(am.conditionStart) != 0 {
		t.Errorf("conditionStart should be empty after reset, got %d entries", len(am.conditionStart))
	}
}
