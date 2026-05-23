package correlation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithCorrelationID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctx = WithCorrelationID(ctx, "corr-123")
	assert.Equal(t, "corr-123", CorrelationIDFromContext(ctx))
}

func TestCorrelationIDFromContextEmpty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	assert.Equal(t, "", CorrelationIDFromContext(ctx))
}

func TestCorrelationIDFromContextWrongType(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.Background(), correlationIDKey{}, 42)
	assert.Equal(t, "", CorrelationIDFromContext(ctx))
}

func TestCorrelationIDOrNewExisting(t *testing.T) {
	t.Parallel()
	ctx := WithCorrelationID(context.Background(), "existing-id")
	newCtx, id := CorrelationIDOrNew(ctx, "new-id")
	assert.Equal(t, "existing-id", id)
	assert.Equal(t, ctx, newCtx)
}

func TestCorrelationIDOrNewMissing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	newCtx, id := CorrelationIDOrNew(ctx, "generated-id")
	assert.Equal(t, "generated-id", id)
	assert.NotEqual(t, ctx, newCtx)
}

func TestCorrelationIDEmptyString(t *testing.T) {
	t.Parallel()
	ctx := WithCorrelationID(context.Background(), "")
	assert.Equal(t, "", CorrelationIDFromContext(ctx))
}

func TestCorrelationIDMultipleLevels(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctx = WithCorrelationID(ctx, "parent")
	assert.Equal(t, "parent", CorrelationIDFromContext(ctx))

	childCtx := context.WithValue(ctx, correlationIDKey{}, "child")
	assert.Equal(t, "child", CorrelationIDFromContext(childCtx))
}
