package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// ErrorHandler provides error classification and handling
type ErrorHandler struct {
	logger             core.Logger
	metricsCollector   core.MetricsCollector
	errorClassifier    ErrorClassifier
	errorLoggers       map[string]ErrorLogger
	errorLoggersMu     sync.RWMutex
	contextProviders   map[string]ContextProvider
	contextProvidersMu sync.RWMutex
}

// ErrorClassifier classifies errors into categories
type ErrorClassifier interface {
	Classify(err error) ErrorCategory
	IsTransient(err error) bool
	IsPermanent(err error) bool
	IsCritical(err error) bool
}

// ErrorLogger logs errors with context
type ErrorLogger interface {
	LogError(ctx context.Context, err error, category ErrorCategory, context map[string]interface{})
}

// ContextProvider provides context for error logging
type ContextProvider interface {
	GetContext(ctx context.Context) map[string]interface{}
}

// ErrorCategory represents the category of an error
type ErrorCategory string

const (
	ErrorCategoryTransient ErrorCategory = "transient"
	ErrorCategoryPermanent ErrorCategory = "permanent"
	ErrorCategoryCritical  ErrorCategory = "critical"
	ErrorCategoryUnknown   ErrorCategory = "unknown"
)

// DefaultErrorClassifier implements ErrorClassifier
type DefaultErrorClassifier struct {
	transientPatterns []string
	permanentPatterns []string
	criticalPatterns  []string
}

// NewDefaultErrorClassifier creates a new default error classifier
func NewDefaultErrorClassifier() *DefaultErrorClassifier {
	return &DefaultErrorClassifier{
		transientPatterns: []string{
			"timeout",
			"connection refused",
			"connection reset",
			"temporary failure",
			"unavailable",
			"deadline exceeded",
		},
		permanentPatterns: []string{
			"invalid",
			"unauthorized",
			"forbidden",
			"not found",
			"bad request",
			"corrupted",
			"authentication failed",
		},
		criticalPatterns: []string{
			"data corruption",
			"critical error",
			"fatal",
			"panic",
		},
	}
}

// Classify classifies an error into a category
func (c *DefaultErrorClassifier) Classify(err error) ErrorCategory {
	if err == nil {
		return ErrorCategoryUnknown
	}

	errMsg := err.Error()

	// Check critical patterns first
	for _, pattern := range c.criticalPatterns {
		if contains(errMsg, pattern) {
			return ErrorCategoryCritical
		}
	}

	// Check transient patterns
	for _, pattern := range c.transientPatterns {
		if contains(errMsg, pattern) {
			return ErrorCategoryTransient
		}
	}

	// Check permanent patterns
	for _, pattern := range c.permanentPatterns {
		if contains(errMsg, pattern) {
			return ErrorCategoryPermanent
		}
	}

	return ErrorCategoryUnknown
}

// IsTransient checks if an error is transient
func (c *DefaultErrorClassifier) IsTransient(err error) bool {
	return c.Classify(err) == ErrorCategoryTransient
}

// IsPermanent checks if an error is permanent
func (c *DefaultErrorClassifier) IsPermanent(err error) bool {
	return c.Classify(err) == ErrorCategoryPermanent
}

// IsCritical checks if an error is critical
func (c *DefaultErrorClassifier) IsCritical(err error) bool {
	return c.Classify(err) == ErrorCategoryCritical
}

// DefaultErrorLogger implements ErrorLogger
type DefaultErrorLogger struct {
	logger core.Logger
}

// NewDefaultErrorLogger creates a new default error logger
func NewDefaultErrorLogger(logger core.Logger) *DefaultErrorLogger {
	return &DefaultErrorLogger{
		logger: logger,
	}
}

// LogError logs an error with context
func (l *DefaultErrorLogger) LogError(ctx context.Context, err error, category ErrorCategory, context map[string]interface{}) {
	if err == nil {
		return
	}

	// Build log message with context
	logMsg := fmt.Sprintf("Error [%s]: %v", category, err)

	// Add context information
	if len(context) > 0 {
		logMsg += " | Context: "
		for key, value := range context {
			logMsg += fmt.Sprintf("%s=%v ", key, value)
		}
	}

	// Log based on category
	switch category {
	case ErrorCategoryCritical:
		l.logger.Error(logMsg)
	case ErrorCategoryPermanent:
		l.logger.Warn(logMsg)
	case ErrorCategoryTransient:
		l.logger.Info(logMsg)
	default:
		l.logger.Debug(logMsg)
	}
}

// DefaultContextProvider implements ContextProvider
type DefaultContextProvider struct {
	correlationID string
	requestID     string
	userID        string
}

// NewDefaultContextProvider creates a new default context provider
func NewDefaultContextProvider(correlationID, requestID, userID string) *DefaultContextProvider {
	return &DefaultContextProvider{
		correlationID: correlationID,
		requestID:     requestID,
		userID:        userID,
	}
}

// GetContext returns context information
func (p *DefaultContextProvider) GetContext(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"correlationID": p.correlationID,
		"requestID":     p.requestID,
		"userID":        p.userID,
		"timestamp":     time.Now().Unix(),
	}
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger core.Logger, metricsCollector core.MetricsCollector) *ErrorHandler {
	return &ErrorHandler{
		logger:           logger,
		metricsCollector: metricsCollector,
		errorClassifier:  NewDefaultErrorClassifier(),
		errorLoggers:     make(map[string]ErrorLogger),
		contextProviders: make(map[string]ContextProvider),
	}
}

// RegisterErrorLogger registers an error logger
func (h *ErrorHandler) RegisterErrorLogger(name string, logger ErrorLogger) {
	h.errorLoggersMu.Lock()
	defer h.errorLoggersMu.Unlock()
	h.errorLoggers[name] = logger
}

// UnregisterErrorLogger unregisters an error logger
func (h *ErrorHandler) UnregisterErrorLogger(name string) {
	h.errorLoggersMu.Lock()
	defer h.errorLoggersMu.Unlock()
	delete(h.errorLoggers, name)
}

// RegisterContextProvider registers a context provider
func (h *ErrorHandler) RegisterContextProvider(name string, provider ContextProvider) {
	h.contextProvidersMu.Lock()
	defer h.contextProvidersMu.Unlock()
	h.contextProviders[name] = provider
}

// UnregisterContextProvider unregisters a context provider
func (h *ErrorHandler) UnregisterContextProvider(name string) {
	h.contextProvidersMu.Lock()
	defer h.contextProvidersMu.Unlock()
	delete(h.contextProviders, name)
}

// HandleError handles an error with classification and logging
func (h *ErrorHandler) HandleError(ctx context.Context, err error, source string) {
	if err == nil {
		return
	}

	// Classify error
	category := h.errorClassifier.Classify(err)

	// Record metrics
	h.metricsCollector.RecordCounter("error_count", 1, map[string]string{
		"category": string(category),
		"source":   source,
	})

	// Collect context
	errorContext := make(map[string]interface{})
	errorContext["source"] = source
	errorContext["category"] = category

	// Add context from providers
	h.contextProvidersMu.RLock()
	for name, provider := range h.contextProviders {
		providerContext := provider.GetContext(ctx)
		for key, value := range providerContext {
			errorContext[name+"_"+key] = value
		}
	}
	h.contextProvidersMu.RUnlock()

	// Log error with all registered loggers
	h.errorLoggersMu.RLock()
	for _, logger := range h.errorLoggers {
		logger.LogError(ctx, err, category, errorContext)
	}
	h.errorLoggersMu.RUnlock()

	// Log with default logger
	defaultLogger := NewDefaultErrorLogger(h.logger)
	defaultLogger.LogError(ctx, err, category, errorContext)
}

// IsTransient checks if an error is transient
func (h *ErrorHandler) IsTransient(err error) bool {
	return h.errorClassifier.IsTransient(err)
}

// IsPermanent checks if an error is permanent
func (h *ErrorHandler) IsPermanent(err error) bool {
	return h.errorClassifier.IsPermanent(err)
}

// IsCritical checks if an error is critical
func (h *ErrorHandler) IsCritical(err error) bool {
	return h.errorClassifier.IsCritical(err)
}

// Classify classifies an error
func (h *ErrorHandler) Classify(err error) ErrorCategory {
	return h.errorClassifier.Classify(err)
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	// Simple substring check (case-sensitive for now)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
