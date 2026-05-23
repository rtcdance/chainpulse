package qerrors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestType_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "transient", TypeTransient.String())
	assert.Equal(t, "permanent", TypePermanent.String())
	assert.Equal(t, "critical", TypeCritical.String())
	assert.Equal(t, "unknown", TypeUnknown.String())
	assert.Equal(t, "unknown", Type(99).String())
}

func TestNewClassifier(t *testing.T) {
	t.Parallel()
	c := NewClassifier()
	assert.NotNil(t, c)
	assert.NotEmpty(t, c.transientPatterns)
	assert.NotEmpty(t, c.permanentPatterns)
	assert.NotEmpty(t, c.criticalPatterns)
}

func TestClassifier_ClassifyNil(t *testing.T) {
	t.Parallel()
	c := NewClassifier()
	assert.Equal(t, TypeUnknown, c.Classify(nil))
}

func TestClassifier_ClassifyTransient(t *testing.T) {
	t.Parallel()
	c := NewClassifier()

	assert.Equal(t, TypeTransient, c.Classify(errors.New("connection refused")))
	assert.Equal(t, TypeTransient, c.Classify(errors.New("timeout")))
	assert.Equal(t, TypeTransient, c.Classify(errors.New("too many connections")))
	assert.Equal(t, TypeTransient, c.Classify(errors.New("deadline exceeded")))
	assert.Equal(t, TypeTransient, c.Classify(errors.New("context deadline")))
	assert.Equal(t, TypeTransient, c.Classify(errors.New("no reachable servers")))
	assert.Equal(t, TypeTransient, c.Classify(errors.New("server selection timeout")))
}

func TestClassifier_ClassifyPermanent(t *testing.T) {
	t.Parallel()
	c := NewClassifier()

	assert.Equal(t, TypePermanent, c.Classify(errors.New("not found")))
	assert.Equal(t, TypePermanent, c.Classify(errors.New("invalid")))
	assert.Equal(t, TypePermanent, c.Classify(errors.New("permission denied")))
	assert.Equal(t, TypePermanent, c.Classify(errors.New("unauthorized")))
	assert.Equal(t, TypePermanent, c.Classify(errors.New("forbidden")))
	assert.Equal(t, TypePermanent, c.Classify(errors.New("already exists")))
	assert.Equal(t, TypePermanent, c.Classify(errors.New("constraint violation")))
	assert.Equal(t, TypePermanent, c.Classify(errors.New("duplicate key")))
}

func TestClassifier_ClassifyCritical(t *testing.T) {
	t.Parallel()
	c := NewClassifier()

	assert.Equal(t, TypeCritical, c.Classify(errors.New("panic")))
	assert.Equal(t, TypeCritical, c.Classify(errors.New("corruption")))
	assert.Equal(t, TypeCritical, c.Classify(errors.New("data loss")))
	assert.Equal(t, TypeCritical, c.Classify(errors.New("integrity violation")))
	assert.Equal(t, TypeCritical, c.Classify(errors.New("fatal")))
	assert.Equal(t, TypeCritical, c.Classify(errors.New("disk full")))
	assert.Equal(t, TypeCritical, c.Classify(errors.New("out of memory")))
}

func TestClassifier_ClassifyCriticalOverTransient(t *testing.T) {
	t.Parallel()
	c := NewClassifier()
	assert.Equal(t, TypeCritical, c.Classify(errors.New("fatal timeout")))
}

func TestClassifier_ClassifyUnknown(t *testing.T) {
	t.Parallel()
	c := NewClassifier()
	assert.Equal(t, TypeUnknown, c.Classify(errors.New("some random error")))
}

func TestClassifier_ClassifyCaseInsensitive(t *testing.T) {
	t.Parallel()
	c := NewClassifier()
	assert.Equal(t, TypeCritical, c.Classify(errors.New("FATAL")))
	assert.Equal(t, TypeTransient, c.Classify(errors.New("TIMEOUT")))
	assert.Equal(t, TypePermanent, c.Classify(errors.New("NOT FOUND")))
}

func TestClassifier_IsTransient(t *testing.T) {
	t.Parallel()
	c := NewClassifier()
	assert.True(t, c.IsTransient(errors.New("timeout")))
	assert.False(t, c.IsTransient(errors.New("not found")))
	assert.False(t, c.IsTransient(errors.New("panic")))
}

func TestClassifier_IsPermanent(t *testing.T) {
	t.Parallel()
	c := NewClassifier()
	assert.True(t, c.IsPermanent(errors.New("not found")))
	assert.False(t, c.IsPermanent(errors.New("timeout")))
	assert.False(t, c.IsPermanent(errors.New("panic")))
}

func TestClassifier_IsCritical(t *testing.T) {
	t.Parallel()
	c := NewClassifier()
	assert.True(t, c.IsCritical(errors.New("panic")))
	assert.False(t, c.IsCritical(errors.New("timeout")))
	assert.False(t, c.IsCritical(errors.New("not found")))
}

func TestClassifier_ClassifyWithContextNil(t *testing.T) {
	t.Parallel()
	c := NewClassifier()
	assert.Equal(t, TypeUnknown, c.ClassifyWithContext(nil, "some context"))
}

func TestClassifier_ClassifyWithContext(t *testing.T) {
	t.Parallel()
	c := NewClassifier()

	result := c.ClassifyWithContext(errors.New("some error"), "connection refused")
	assert.Equal(t, TypeTransient, result)

	result = c.ClassifyWithContext(errors.New("some error"), "permission denied")
	assert.Equal(t, TypePermanent, result)

	result = c.ClassifyWithContext(errors.New("some error"), "disk full")
	assert.Equal(t, TypeCritical, result)

	result = c.ClassifyWithContext(errors.New("some error"), "normal context")
	assert.Equal(t, TypeUnknown, result)
}

func TestClassifier_ClassifyWithContextCriticalOverrides(t *testing.T) {
	t.Parallel()
	c := NewClassifier()
	result := c.ClassifyWithContext(errors.New("timeout"), "disk full")
	assert.Equal(t, TypeCritical, result)
}

func TestClassifier_IsMongoErrorTransientNil(t *testing.T) {
	t.Parallel()
	c := NewClassifier()
	assert.False(t, c.IsMongoErrorTransient(nil))
}

func TestClassifier_IsMongoErrorTransientFallback(t *testing.T) {
	t.Parallel()
	c := NewClassifier()
	assert.True(t, c.IsMongoErrorTransient(errors.New("connection refused")))
	assert.False(t, c.IsMongoErrorTransient(errors.New("not found")))
}

func TestErrorsAsNil(t *testing.T) {
	t.Parallel()
	assert.False(t, errorsAs(nil, new(error)))
}

func TestErrorVars(t *testing.T) {
	t.Parallel()
	assert.EqualError(t, ErrQueryOptimizationFailed, "query optimization failed")
	assert.EqualError(t, ErrInvalidQuery, "invalid query")
	assert.EqualError(t, ErrQueryPlanAnalysisFailed, "query plan analysis failed")
	assert.EqualError(t, ErrIndexNotFound, "index not found")
	assert.EqualError(t, ErrIndexAlreadyExists, "index already exists")
	assert.EqualError(t, ErrInvalidIndexParameters, "invalid index parameters")
	assert.EqualError(t, ErrIndexCreationFailed, "index creation failed")
	assert.EqualError(t, ErrIndexDeletionFailed, "index deletion failed")
	assert.EqualError(t, ErrIndexRebuildFailed, "index rebuild failed")
	assert.EqualError(t, ErrIndexFragmentationAnalysisFailed, "index fragmentation analysis failed")
	assert.EqualError(t, ErrStatisticsCollectionFailed, "statistics collection failed")
	assert.EqualError(t, ErrCacheOperationFailed, "cache operation failed")
	assert.EqualError(t, ErrPaginationFailed, "pagination failed")
	assert.EqualError(t, ErrFilterOptimizationFailed, "filter optimization failed")
}
