package bootstrap

import (
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/rtcdance/chainpulse/pkg/configmodel"

	"github.com/rtcdance/chainpulse/pkg/observability"
)

// BuildAlertManager constructs the AlertManager with a log-sink notifier (D26)
// and config-driven rules (D6). Invalid rules are skipped with a warning so a
// single bad entry cannot block startup. The manager is always returned; with
// cfg.Enabled false it holds the sink but no armed rules.
func BuildAlertManager(logger portcontract.Logger, cfg configmodel.AlertsConfig) *observability.AlertManager {
	manager := observability.NewAlertManager()
	if logger == nil {
		return manager
	}

	manager.AddCallback(func(rule observability.AlertRule, value float64, triggeredAt time.Time) {
		logger.Warn("alert triggered",
			"rule", rule.Name,
			"metric", rule.Metric,
			"value", value,
			"operator", rule.Operator,
			"threshold", rule.Threshold,
			"severity", string(rule.Severity),
			"message", rule.Message,
			"triggered_at", triggeredAt.Format(time.RFC3339),
		)
	})

	if !cfg.Enabled {
		return manager
	}

	for _, ruleCfg := range cfg.Rules {
		rule := observability.AlertRule{
			Name:      ruleCfg.Name,
			Metric:    ruleCfg.Metric,
			Operator:  ruleCfg.Operator,
			Threshold: ruleCfg.Threshold,
			Duration:  time.Duration(ruleCfg.DurationSeconds) * time.Second,
			Severity:  observability.AlertSeverity(ruleCfg.Severity),
			Message:   ruleCfg.Message,
		}
		if err := manager.AddRule(rule); err != nil {
			logger.Warn("skipping invalid alert rule",
				"rule", ruleCfg.Name, "error", err.Error())
		}
	}
	return manager
}
