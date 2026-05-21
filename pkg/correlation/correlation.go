package correlation

import "context"

type correlationIDKey struct{}

func WithID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey{}, id)
}

func FromContext(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey{}).(string); ok {
		return id
	}
	return ""
}

func FromContextOrNew(ctx context.Context, newID string) (context.Context, string) {
	if id := FromContext(ctx); id != "" {
		return ctx, id
	}
	return WithID(ctx, newID), newID
}
