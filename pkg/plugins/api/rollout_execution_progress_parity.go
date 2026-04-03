package api

import (
	"fmt"
	"strings"
)

// ValidateRolloutExecutionProgressReasonCoverage validates that a rollout
// reason string contains the execution-progress details implied by the shared
// execution-progress facade.
func ValidateRolloutExecutionProgressReasonCoverage(reason string, progress RolloutExecutionProgress) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("rollout reason is required")
	}

	parts := []string{}
	AppendRolloutExecutionProgressReason(&parts, progress)
	for _, part := range parts {
		if !strings.Contains(reason, part) {
			return fmt.Errorf("expected rollout reason to contain %q, got %q", part, reason)
		}
	}
	return nil
}

// ValidateRolloutExecutionProgressPostureReasonCoverage validates that a
// rollout reason string contains the compact posture details implied by the
// shared execution-progress facade.
func ValidateRolloutExecutionProgressPostureReasonCoverage(reason string, progress RolloutExecutionProgress) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("rollout reason is required")
	}

	parts := []string{}
	AppendRolloutExecutionProgressPostureReason(&parts, progress)
	for _, part := range parts {
		if !strings.Contains(reason, part) {
			return fmt.Errorf("expected rollout reason to contain %q, got %q", part, reason)
		}
	}
	return nil
}

// ValidateRolloutExecutionOperatorHintReasonCoverage validates that a rollout
// reason string contains the operator-hint details implied by the shared hint
// facade.
func ValidateRolloutExecutionOperatorHintReasonCoverage(reason string, hint RolloutExecutionOperatorHint) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("rollout reason is required")
	}

	parts := []string{}
	AppendRolloutExecutionOperatorHintReason(&parts, hint)
	for _, part := range parts {
		if !strings.Contains(reason, part) {
			return fmt.Errorf("expected rollout reason to contain %q, got %q", part, reason)
		}
	}
	return nil
}
