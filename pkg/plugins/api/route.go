package api

import (
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"time"
)

// Route represents a route pattern with handlers
type Route struct {
	ID          string
	Pattern     string
	Method      string
	Priority    int // Higher priority routes are matched first (default 0)
	Handlers    []*RequestHandler
	Middleware  []func(http.Handler) http.Handler
	CreatedAt   time.Time
	UpdatedAt   time.Time
	mu          sync.RWMutex
	compileOnce sync.Once
	pathRegex   *regexp.Regexp
	paramNames  []string
}

// NewRoute creates a new route
func NewRoute(id, pattern, method string) *Route {
	return &Route{
		ID:         id,
		Pattern:    pattern,
		Method:     method,
		Priority:   0,
		Handlers:   make([]*RequestHandler, 0),
		Middleware: make([]func(http.Handler) http.Handler, 0),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// SetPriority sets the route priority (higher priority routes are matched first)
func (r *Route) SetPriority(priority int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Priority = priority
	r.UpdatedAt = time.Now()
}

// AddHandler adds a handler to the route
func (r *Route) AddHandler(handler *RequestHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if handler already exists
	for _, h := range r.Handlers {
		if h.ID == handler.ID {
			return fmt.Errorf("handler %s already exists", handler.ID)
		}
	}

	r.Handlers = append(r.Handlers, handler)
	r.UpdatedAt = time.Now()
	return nil
}

// RemoveHandler removes a handler from the route
func (r *Route) RemoveHandler(handlerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, h := range r.Handlers {
		if h.ID == handlerID {
			r.Handlers = append(r.Handlers[:i], r.Handlers[i+1:]...)
			r.UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("handler %s not found", handlerID)
}

// GetHandlers returns a copy of the handlers list
func (r *Route) GetHandlers() []*RequestHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handlers := make([]*RequestHandler, len(r.Handlers))
	copy(handlers, r.Handlers)
	return handlers
}

// Match checks if a path matches this route and extracts parameters
func (r *Route) Match(path string) (map[string]string, bool) {
	// Compile regex exactly once (safe for concurrent Match calls)
	r.compileOnce.Do(func() {
		_ = r.compilePattern()
	})
	if r.pathRegex == nil {
		return nil, false
	}

	matches := r.pathRegex.FindStringSubmatch(path)
	if matches == nil {
		return nil, false
	}

	// Extract parameters
	params := make(map[string]string)
	for i, name := range r.paramNames {
		if i+1 < len(matches) {
			params[name] = matches[i+1]
		}
	}

	return params, true
}

// compilePattern compiles the route pattern into a regex
func (r *Route) compilePattern() error {
	pattern := r.Pattern
	paramNames := make([]string, 0)

	// Find all parameters in the pattern (e.g., :id, :chainId)
	paramRegex := regexp.MustCompile(`:(\w+)`)
	matches := paramRegex.FindAllStringSubmatchIndex(pattern, -1)

	// Replace parameters with regex groups
	offset := 0
	for _, match := range matches {
		start := match[0] + offset
		end := match[1] + offset
		paramName := pattern[match[2]:match[3]]
		paramNames = append(paramNames, paramName)

		// Replace :paramName with regex group
		pattern = pattern[:start] + "([^/]+)" + pattern[end:]
		offset += len("([^/]+)") - (end - start)
	}

	// Anchor the pattern
	pattern = "^" + pattern + "$"

	// Compile regex
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("failed to compile route pattern: %w", err)
	}

	r.pathRegex = regex
	r.paramNames = paramNames
	return nil
}

// AddMiddleware adds middleware to the route
func (r *Route) AddMiddleware(middleware func(http.Handler) http.Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Middleware = append(r.Middleware, middleware)
	r.UpdatedAt = time.Now()
}

// GetMiddleware returns a copy of the middleware list
func (r *Route) GetMiddleware() []func(http.Handler) http.Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	middleware := make([]func(http.Handler) http.Handler, len(r.Middleware))
	copy(middleware, r.Middleware)
	return middleware
}

// String returns a string representation of the route
func (r *Route) String() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return fmt.Sprintf("Route{ID: %s, Pattern: %s, Method: %s, Handlers: %d}",
		r.ID, r.Pattern, r.Method, len(r.Handlers))
}
