package core

import "context"

// correlationIDKey is the context key for storing correlation IDs.
type correlationIDKey struct{}

// WithCorrelationID returns a copy of the parent context with the correlation ID set.
// Correlation IDs enable distributed tracing across service boundaries.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey{}, id)
}

// CorrelationIDFromContext extracts the correlation ID from a context.
// Returns an empty string if no correlation ID is set.
func CorrelationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey{}).(string); ok {
		return id
	}
	return ""
}

// CorrelationIDOrNew returns the existing correlation ID from the context,
// or generates a new one if none exists.
func CorrelationIDOrNew(ctx context.Context, newID string) (context.Context, string) {
	if id := CorrelationIDFromContext(ctx); id != "" {
		return ctx, id
	}
	return WithCorrelationID(ctx, newID), newID
}
