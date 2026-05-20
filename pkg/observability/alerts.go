package observability

import (
	"fmt"
	"sync"
	"time"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertRule defines a condition that, when met, triggers an alert.
type AlertRule struct {
	Name      string            // Unique identifier for the alert
	Metric    string            // Metric name to evaluate
	Operator  string            // Comparison operator: ">", ">=", "<", "<=", "==", "!="
	Threshold float64           // Threshold value for comparison
	Duration  time.Duration     // How long the condition must be true before alerting
	Severity  AlertSeverity     // Severity level
	Message   string            // Human-readable alert message
	Tags      map[string]string // Additional tags for filtering
}

// Validate checks that the AlertRule has valid configuration
func (r AlertRule) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("alert rule name is required")
	}
	if r.Metric == "" {
		return fmt.Errorf("metric is required for alert rule %s", r.Name)
	}
	switch r.Operator {
	case ">", ">=", "<", "<=", "==", "!=":
		// valid
	default:
		return fmt.Errorf("invalid operator %q for alert rule %s (must be >, >=, <, <=, ==, !=)", r.Operator, r.Name)
	}
	if r.Duration <= 0 {
		return fmt.Errorf("duration must be positive for alert rule %s", r.Name)
	}
	return nil
}

// AlertCallback is called when an alert is triggered
type AlertCallback func(rule AlertRule, currentValue float64, triggeredAt time.Time)

// AlertManager manages alert rules and evaluates them against metric values.
type AlertManager struct {
	mu        sync.RWMutex
	rules     map[string]AlertRule
	callbacks []AlertCallback
	// Track when each rule's condition became true
	conditionStart map[string]time.Time
}

// NewAlertManager creates a new AlertManager instance
func NewAlertManager() *AlertManager {
	return &AlertManager{
		rules:          make(map[string]AlertRule),
		conditionStart: make(map[string]time.Time),
	}
}

// AddRule adds or updates an alert rule
func (am *AlertManager) AddRule(rule AlertRule) error {
	if err := rule.Validate(); err != nil {
		return err
	}
	am.mu.Lock()
	defer am.mu.Unlock()
	am.rules[rule.Name] = rule
	return nil
}

// RemoveRule removes an alert rule
func (am *AlertManager) RemoveRule(name string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.rules, name)
	delete(am.conditionStart, name)
}

// GetRule returns an alert rule by name
func (am *AlertManager) GetRule(name string) (AlertRule, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	rule, ok := am.rules[name]
	return rule, ok
}

// ListRules returns all registered alert rule names
func (am *AlertManager) ListRules() []string {
	am.mu.RLock()
	defer am.mu.RUnlock()
	names := make([]string, 0, len(am.rules))
	for name := range am.rules {
		names = append(names, name)
	}
	return names
}

// AddCallback registers a callback to be called when an alert is triggered
func (am *AlertManager) AddCallback(cb AlertCallback) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.callbacks = append(am.callbacks, cb)
}

// Evaluate checks a metric value against all applicable alert rules
func (am *AlertManager) Evaluate(metric string, value float64) {
	am.mu.RLock()
	rules := make([]AlertRule, 0, len(am.rules))
	for _, rule := range am.rules {
		if rule.Metric == metric {
			rules = append(rules, rule)
		}
	}
	callbacks := make([]AlertCallback, len(am.callbacks))
	copy(callbacks, am.callbacks)
	am.mu.RUnlock()

	now := time.Now()
	for _, rule := range rules {
		triggered := am.evaluateCondition(rule, value)
		if triggered {
			// Check if condition has been true for the required duration
			am.mu.Lock()
			startTime, exists := am.conditionStart[rule.Name]
			if !exists {
				am.conditionStart[rule.Name] = now
				startTime = now
			}
			am.mu.Unlock()

			if now.Sub(startTime) >= rule.Duration {
				for _, cb := range callbacks {
					cb(rule, value, now)
				}
			}
		} else {
			am.mu.Lock()
			delete(am.conditionStart, rule.Name)
			am.mu.Unlock()
		}
	}
}

// evaluateCondition checks if a rule's condition is met for the given value
func (am *AlertManager) evaluateCondition(rule AlertRule, value float64) bool {
	switch rule.Operator {
	case ">":
		return value > rule.Threshold
	case ">=":
		return value >= rule.Threshold
	case "<":
		return value < rule.Threshold
	case "<=":
		return value <= rule.Threshold
	case "==":
		return value == rule.Threshold
	case "!=":
		return value != rule.Threshold
	default:
		return false
	}
}

// Reset clears all alert state (for testing or reconfiguration)
func (am *AlertManager) Reset() {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.conditionStart = make(map[string]time.Time)
}
