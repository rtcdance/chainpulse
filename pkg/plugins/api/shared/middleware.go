package shared

import (
	"sync"

	"github.com/rtcdance/chainpulse/pkg/plugins/api/core"
)

// MiddlewareChain represents a chain of middleware
type MiddlewareChain struct {
	middlewares []core.Middleware
	mu          sync.RWMutex
}

// NewMiddlewareChain creates a new middleware chain
func NewMiddlewareChain() *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: make([]core.Middleware, 0),
	}
}

// Add adds middleware to the chain
func (c *MiddlewareChain) Add(middleware ...core.Middleware) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.middlewares = append(c.middlewares, middleware...)
}

// Build builds the middleware chain into a single handler
func (c *MiddlewareChain) Build(handler core.Handler) core.Handler {
	c.mu.RLock()
	middlewares := make([]core.Middleware, len(c.middlewares))
	copy(middlewares, c.middlewares)
	c.mu.RUnlock()

	// Apply middleware in reverse order
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return handler
}

// SecurityMiddlewareGroup groups security-related middleware
type SecurityMiddlewareGroup struct {
	authMiddleware core.Middleware
	tlsMiddleware  core.Middleware
	mu             sync.RWMutex
}

// NewSecurityMiddlewareGroup creates a new security middleware group
func NewSecurityMiddlewareGroup() *SecurityMiddlewareGroup {
	return &SecurityMiddlewareGroup{}
}

// SetAuthMiddleware sets the authentication middleware
func (g *SecurityMiddlewareGroup) SetAuthMiddleware(middleware core.Middleware) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.authMiddleware = middleware
}

// SetTLSMiddleware sets the TLS middleware
func (g *SecurityMiddlewareGroup) SetTLSMiddleware(middleware core.Middleware) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.tlsMiddleware = middleware
}

// GetMiddlewares returns all security middleware
func (g *SecurityMiddlewareGroup) GetMiddlewares() []core.Middleware {
	g.mu.RLock()
	defer g.mu.RUnlock()

	middlewares := make([]core.Middleware, 0)
	if g.authMiddleware != nil {
		middlewares = append(middlewares, g.authMiddleware)
	}
	if g.tlsMiddleware != nil {
		middlewares = append(middlewares, g.tlsMiddleware)
	}

	return middlewares
}

// ObservabilityMiddlewareGroup groups observability-related middleware
type ObservabilityMiddlewareGroup struct {
	healthMiddleware     core.Middleware
	monitoringMiddleware core.Middleware
	mu                   sync.RWMutex
}

// NewObservabilityMiddlewareGroup creates a new observability middleware group
func NewObservabilityMiddlewareGroup() *ObservabilityMiddlewareGroup {
	return &ObservabilityMiddlewareGroup{}
}

// SetHealthMiddleware sets the health check middleware
func (g *ObservabilityMiddlewareGroup) SetHealthMiddleware(middleware core.Middleware) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.healthMiddleware = middleware
}

// SetMonitoringMiddleware sets the monitoring middleware
func (g *ObservabilityMiddlewareGroup) SetMonitoringMiddleware(middleware core.Middleware) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.monitoringMiddleware = middleware
}

// GetMiddlewares returns all observability middleware
func (g *ObservabilityMiddlewareGroup) GetMiddlewares() []core.Middleware {
	g.mu.RLock()
	defer g.mu.RUnlock()

	middlewares := make([]core.Middleware, 0)
	if g.healthMiddleware != nil {
		middlewares = append(middlewares, g.healthMiddleware)
	}
	if g.monitoringMiddleware != nil {
		middlewares = append(middlewares, g.monitoringMiddleware)
	}

	return middlewares
}

// PerformanceMiddlewareGroup groups performance-related middleware
type PerformanceMiddlewareGroup struct {
	compressionMiddleware core.Middleware
	batchingMiddleware    core.Middleware
	poolMiddleware        core.Middleware
	mu                    sync.RWMutex
}

// NewPerformanceMiddlewareGroup creates a new performance middleware group
func NewPerformanceMiddlewareGroup() *PerformanceMiddlewareGroup {
	return &PerformanceMiddlewareGroup{}
}

// SetCompressionMiddleware sets the compression middleware
func (g *PerformanceMiddlewareGroup) SetCompressionMiddleware(middleware core.Middleware) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.compressionMiddleware = middleware
}

// SetBatchingMiddleware sets the batching middleware
func (g *PerformanceMiddlewareGroup) SetBatchingMiddleware(middleware core.Middleware) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.batchingMiddleware = middleware
}

// SetPoolMiddleware sets the connection pool middleware
func (g *PerformanceMiddlewareGroup) SetPoolMiddleware(middleware core.Middleware) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.poolMiddleware = middleware
}

// GetMiddlewares returns all performance middleware
func (g *PerformanceMiddlewareGroup) GetMiddlewares() []core.Middleware {
	g.mu.RLock()
	defer g.mu.RUnlock()

	middlewares := make([]core.Middleware, 0)
	if g.compressionMiddleware != nil {
		middlewares = append(middlewares, g.compressionMiddleware)
	}
	if g.batchingMiddleware != nil {
		middlewares = append(middlewares, g.batchingMiddleware)
	}
	if g.poolMiddleware != nil {
		middlewares = append(middlewares, g.poolMiddleware)
	}

	return middlewares
}

// ErrorHandlingMiddleware provides unified error handling
type ErrorHandlingMiddleware struct {
	errorMapper core.ErrorMapper
	mu          sync.RWMutex
}

// NewErrorHandlingMiddleware creates a new error handling middleware
func NewErrorHandlingMiddleware(errorMapper core.ErrorMapper) *ErrorHandlingMiddleware {
	return &ErrorHandlingMiddleware{
		errorMapper: errorMapper,
	}
}

// Middleware returns the middleware function
func (m *ErrorHandlingMiddleware) Middleware() core.Middleware {
	return func(next core.Handler) core.Handler {
		return core.HandlerFunc(func(req core.Request) (core.Response, error) {
			resp, err := next.Handle(req)
			if err != nil {
				m.mu.RLock()
				errorMapper := m.errorMapper
				m.mu.RUnlock()

				if errorMapper != nil {
					status, headers, body := errorMapper.MapError(err)
					resp := core.NewBaseResponse(nil)
					resp.SetStatus(status)
					for k, v := range headers {
						resp.SetHeader(k, v)
					}
					resp.SetBody(body)
					return resp, nil
				}
			}
			return resp, err
		})
	}
}

// MiddlewareBuilder provides a fluent interface for building middleware chains
type MiddlewareBuilder struct {
	chain *MiddlewareChain
}

// NewMiddlewareBuilder creates a new middleware builder
func NewMiddlewareBuilder() *MiddlewareBuilder {
	return &MiddlewareBuilder{
		chain: NewMiddlewareChain(),
	}
}

// WithSecurity adds security middleware
func (b *MiddlewareBuilder) WithSecurity(group *SecurityMiddlewareGroup) *MiddlewareBuilder {
	b.chain.Add(group.GetMiddlewares()...)
	return b
}

// WithObservability adds observability middleware
func (b *MiddlewareBuilder) WithObservability(group *ObservabilityMiddlewareGroup) *MiddlewareBuilder {
	b.chain.Add(group.GetMiddlewares()...)
	return b
}

// WithPerformance adds performance middleware
func (b *MiddlewareBuilder) WithPerformance(group *PerformanceMiddlewareGroup) *MiddlewareBuilder {
	b.chain.Add(group.GetMiddlewares()...)
	return b
}

// WithErrorHandling adds error handling middleware
func (b *MiddlewareBuilder) WithErrorHandling(middleware core.Middleware) *MiddlewareBuilder {
	b.chain.Add(middleware)
	return b
}

// Build builds the middleware chain
func (b *MiddlewareBuilder) Build(handler core.Handler) core.Handler {
	return b.chain.Build(handler)
}

// MiddlewareRegistry manages middleware groups
type MiddlewareRegistry struct {
	security      *SecurityMiddlewareGroup
	observability *ObservabilityMiddlewareGroup
	performance   *PerformanceMiddlewareGroup
	errorHandling *ErrorHandlingMiddleware
	mu            sync.RWMutex
}

// NewMiddlewareRegistry creates a new middleware registry
func NewMiddlewareRegistry() *MiddlewareRegistry {
	return &MiddlewareRegistry{
		security:      NewSecurityMiddlewareGroup(),
		observability: NewObservabilityMiddlewareGroup(),
		performance:   NewPerformanceMiddlewareGroup(),
	}
}

// GetSecurityGroup returns the security middleware group
func (r *MiddlewareRegistry) GetSecurityGroup() *SecurityMiddlewareGroup {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.security
}

// GetObservabilityGroup returns the observability middleware group
func (r *MiddlewareRegistry) GetObservabilityGroup() *ObservabilityMiddlewareGroup {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.observability
}

// GetPerformanceGroup returns the performance middleware group
func (r *MiddlewareRegistry) GetPerformanceGroup() *PerformanceMiddlewareGroup {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.performance
}

// SetErrorHandling sets the error handling middleware
func (r *MiddlewareRegistry) SetErrorHandling(middleware *ErrorHandlingMiddleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errorHandling = middleware
}

// BuildChain builds a complete middleware chain
func (r *MiddlewareRegistry) BuildChain() *MiddlewareChain {
	r.mu.RLock()
	defer r.mu.RUnlock()

	chain := NewMiddlewareChain()

	// Add security middleware
	if r.security != nil {
		chain.Add(r.security.GetMiddlewares()...)
	}

	// Add observability middleware
	if r.observability != nil {
		chain.Add(r.observability.GetMiddlewares()...)
	}

	// Add performance middleware
	if r.performance != nil {
		chain.Add(r.performance.GetMiddlewares()...)
	}

	// Add error handling middleware
	if r.errorHandling != nil {
		chain.Add(r.errorHandling.Middleware())
	}

	return chain
}

// GetAllMiddleware returns all registered middleware
func (r *MiddlewareRegistry) GetAllMiddleware() []core.Middleware {
	r.mu.RLock()
	defer r.mu.RUnlock()

	middlewares := make([]core.Middleware, 0)

	if r.security != nil {
		middlewares = append(middlewares, r.security.GetMiddlewares()...)
	}

	if r.observability != nil {
		middlewares = append(middlewares, r.observability.GetMiddlewares()...)
	}

	if r.performance != nil {
		middlewares = append(middlewares, r.performance.GetMiddlewares()...)
	}

	if r.errorHandling != nil {
		middlewares = append(middlewares, r.errorHandling.Middleware())
	}

	return middlewares
}

// GetRuntimeMetrics returns a compact runtime surface for middleware coverage
// and registry readiness on top of the grouped middleware configuration.
func (r *MiddlewareRegistry) GetRuntimeMetrics() map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	securityCount := 0
	observabilityCount := 0
	performanceCount := 0
	errorHandlingEnabled := false

	if r.security != nil {
		securityCount = len(r.security.GetMiddlewares())
	}
	if r.observability != nil {
		observabilityCount = len(r.observability.GetMiddlewares())
	}
	if r.performance != nil {
		performanceCount = len(r.performance.GetMiddlewares())
	}
	if r.errorHandling != nil {
		errorHandlingEnabled = true
	}

	totalMiddleware := securityCount + observabilityCount + performanceCount
	if errorHandlingEnabled {
		totalMiddleware++
	}

	coveragePosture := classifyMiddlewareCoveragePosture(totalMiddleware, securityCount, observabilityCount, performanceCount, errorHandlingEnabled)
	runtimePosture := classifyMiddlewareRuntimePosture(totalMiddleware, securityCount, observabilityCount, performanceCount, errorHandlingEnabled)

	return map[string]any{
		"total_middleware":       totalMiddleware,
		"security_count":         securityCount,
		"observability_count":    observabilityCount,
		"performance_count":      performanceCount,
		"error_handling_enabled": errorHandlingEnabled,
		"coverage_posture":       coveragePosture,
		"runtime_posture":        runtimePosture,
		"reliability_hint":       buildMiddlewareReliabilityHint(coveragePosture, runtimePosture),
	}
}

func classifyMiddlewareCoveragePosture(totalMiddleware int, securityCount int, observabilityCount int, performanceCount int, errorHandlingEnabled bool) string {
	if totalMiddleware == 0 {
		return "middleware-unconfigured"
	}
	if securityCount > 0 && observabilityCount > 0 && performanceCount > 0 && errorHandlingEnabled {
		return "middleware-full-stack"
	}
	if securityCount == 0 && observabilityCount == 0 && performanceCount == 0 && errorHandlingEnabled {
		return "middleware-error-only"
	}
	return "middleware-partial"
}

func classifyMiddlewareRuntimePosture(totalMiddleware int, securityCount int, observabilityCount int, performanceCount int, errorHandlingEnabled bool) string {
	if totalMiddleware == 0 {
		return "middleware-unobserved"
	}
	if securityCount == 0 || observabilityCount == 0 {
		return "middleware-degraded"
	}
	if !errorHandlingEnabled {
		return "middleware-watch"
	}
	if performanceCount == 0 {
		return "middleware-balanced"
	}
	return "middleware-ready"
}

func buildMiddlewareReliabilityHint(coveragePosture string, runtimePosture string) string {
	switch {
	case runtimePosture == "middleware-degraded":
		return "middleware registry is missing core security or observability coverage; verify baseline protection before relying on the stack"
	case runtimePosture == "middleware-watch":
		return "middleware registry has core groups but lacks error-handling wiring; verify mapped error behavior before treating the stack as complete"
	case coveragePosture == "middleware-error-only":
		return "middleware registry is only exposing error handling; add baseline security and observability coverage before relying on the chain"
	case coveragePosture == "middleware-partial":
		return "middleware registry has partial group coverage; continue observing whether the configured stack matches route expectations"
	case coveragePosture == "middleware-full-stack":
		return "middleware registry has a balanced stack with security, observability, performance, and error handling coverage"
	default:
		return "middleware registry has not been configured yet"
	}
}
