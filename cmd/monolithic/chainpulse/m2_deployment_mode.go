package main

import "strings"

const (
	deploymentModeMonolithic   = "monolithic"
	deploymentModeMicroservice = "microservice"
)

type deploymentModeProfile struct {
	Mode            string
	Recognized      bool
	RequestedMode   string
	Posture         string
	ReliabilityHint string
}

func (c Configuration) deploymentSummary() map[string]any {
	return map[string]any{
		"deployment_mode":    c.DeploymentMode,
		"deployment_posture": c.DeploymentPosture,
		"reliability_hint":   c.DeploymentHint,
	}
}

func resolveDeploymentModeProfile(raw string) deploymentModeProfile {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	switch normalized {
	case "", deploymentModeMonolithic:
		return deploymentModeProfile{
			Mode:            deploymentModeMonolithic,
			Recognized:      true,
			RequestedMode:   normalized,
			Posture:         "deployment-mode-monolithic",
			ReliabilityHint: "monolithic cmd wiring is running in its expected deployment mode baseline",
		}
	case deploymentModeMicroservice:
		return deploymentModeProfile{
			Mode:            deploymentModeMicroservice,
			Recognized:      true,
			RequestedMode:   normalized,
			Posture:         "deployment-mode-microservice-intent",
			ReliabilityHint: "microservice deployment intent is now parsed, but adapter switching is not yet fully closed in M2",
		}
	default:
		return deploymentModeProfile{
			Mode:            deploymentModeMonolithic,
			Recognized:      false,
			RequestedMode:   normalized,
			Posture:         "deployment-mode-fallback",
			ReliabilityHint: "unrecognized deployment mode fell back to monolithic; verify DEPLOYMENT_MODE before treating dual-mode switching as intentional",
		}
	}
}
