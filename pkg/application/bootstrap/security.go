package bootstrap

import (
	"fmt"
	"strings"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/env"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
)

// SecurityControlsConfig holds the configuration needed to build auth and rate limit middleware.
// All four microservices (gateway, api-service, event-processor, puller) use the same wiring.
type SecurityControlsConfig struct {
	AuthEnabled        bool
	AuthJWTSecret      core.SecretString
	AuthAPIKeys        []string
	RateLimitEnabled   bool
	RateLimitPerMinute int
	ServiceName        string
	EnvPrefix          string
}

// BuildSecurityControls creates AuthMiddleware and RateLimitMiddleware from a shared config.
// Returns (nil, nil, nil) when both auth and rate limit are disabled.
func BuildSecurityControls(cfg SecurityControlsConfig, logger core.Logger, metrics core.MetricsCollector) (*api.AuthMiddleware, *api.RateLimitMiddleware, error) {
	if !cfg.AuthEnabled && !cfg.RateLimitEnabled {
		return nil, nil, nil
	}

	var authMiddleware *api.AuthMiddleware
	if cfg.AuthEnabled {
		jwtSecret := cfg.AuthJWTSecret.Value()
		if strings.TrimSpace(jwtSecret) == "" {
			return nil, nil, fmt.Errorf("%s auth is enabled but %s_AUTH_JWT_SECRET is empty", cfg.ServiceName, cfg.EnvPrefix)
		}

		tokenValidator := api.NewTokenValidator(jwtSecret, logger, metrics)
		for _, entry := range cfg.AuthAPIKeys {
			apiKey, clientID, ok := env.ParseKeyValuePair(entry)
			if !ok {
				return nil, nil, fmt.Errorf("invalid %s_AUTH_API_KEYS entry %q; expected key=clientID or key:clientID", cfg.EnvPrefix, entry)
			}
			if err := tokenValidator.RegisterAPIKey(apiKey, clientID, "operator"); err != nil {
				return nil, nil, fmt.Errorf("register api key for client %q: %w", clientID, err)
			}
		}

		rbacChecker := api.NewRBACChecker(logger, metrics)
		if err := rbacChecker.RegisterDefaultRoles(); err != nil {
			return nil, nil, fmt.Errorf("failed to register default RBAC roles: %w", err)
		}
		auditLogger := api.NewAuditLogger(logger, metrics)
		authMiddleware = api.NewAuthMiddleware(tokenValidator, rbacChecker, auditLogger, logger, metrics)
	}

	var rateLimitMiddleware *api.RateLimitMiddleware
	if cfg.RateLimitEnabled {
		rateLimiter := api.NewRateLimiter(logger, metrics, &api.RateLimitConfig{
			DefaultRequestsPerSecond: api.RequestsPerMinuteToPerSecond(cfg.RateLimitPerMinute),
			DefaultBurstSize:         api.BurstSizeFromRequestsPerMinute(cfg.RateLimitPerMinute),
			CleanupInterval:          5 * time.Minute,
		})
		rateLimitMiddleware = api.NewRateLimitMiddleware(rateLimiter, logger)
	}

	return authMiddleware, rateLimitMiddleware, nil
}
