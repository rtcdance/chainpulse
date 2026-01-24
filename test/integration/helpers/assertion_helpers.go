package helpers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/services/query"
	"chainpulse/test/integration/fixtures"
)

// AssertionHelper provides assertion helpers for integration tests
type AssertionHelper struct {
	t *testing.T
}

// NewAssertionHelper creates a new assertion helper
func NewAssertionHelper(t *testing.T) *AssertionHelper {
	return &AssertionHelper{t: t}
}

// AssertEventExists asserts that an event exists in the database
func (ah *AssertionHelper) AssertEventExists(ctx context.Context, fixture *fixtures.DatabaseFixture, eventID string) {
	event, err := fixture.GetEvent(ctx, eventID)
	if err != nil {
		ah.t.Errorf("expected event %s to exist, but got error: %v", eventID, err)
		return
	}
	if event == nil {
		ah.t.Errorf("expected event %s to exist, but got nil", eventID)
	}
}

// AssertEventNotExists asserts that an event does not exist in the database
func (ah *AssertionHelper) AssertEventNotExists(ctx context.Context, fixture *fixtures.DatabaseFixture, eventID string) {
	event, err := fixture.GetEvent(ctx, eventID)
	if err == nil && event != nil {
		ah.t.Errorf("expected event %s to not exist, but it does", eventID)
	}
}

// AssertEventCount asserts that the database contains the expected number of events
func (ah *AssertionHelper) AssertEventCount(ctx context.Context, fixture *fixtures.DatabaseFixture, expectedCount int64) {
	count, err := fixture.EventCount(ctx)
	if err != nil {
		ah.t.Errorf("failed to get event count: %v", err)
		return
	}
	if count != expectedCount {
		ah.t.Errorf("expected %d events, but got %d", expectedCount, count)
	}
}

// AssertEventCountGreaterThan asserts that the database contains more than the expected number of events
func (ah *AssertionHelper) AssertEventCountGreaterThan(ctx context.Context, fixture *fixtures.DatabaseFixture, minCount int64) {
	count, err := fixture.EventCount(ctx)
	if err != nil {
		ah.t.Errorf("failed to get event count: %v", err)
		return
	}
	if count <= minCount {
		ah.t.Errorf("expected more than %d events, but got %d", minCount, count)
	}
}

// AssertEventCountLessThan asserts that the database contains fewer than the expected number of events
func (ah *AssertionHelper) AssertEventCountLessThan(ctx context.Context, fixture *fixtures.DatabaseFixture, maxCount int64) {
	count, err := fixture.EventCount(ctx)
	if err != nil {
		ah.t.Errorf("failed to get event count: %v", err)
		return
	}
	if count >= maxCount {
		ah.t.Errorf("expected fewer than %d events, but got %d", maxCount, count)
	}
}

// AssertCacheHit asserts that a query result came from cache
func (ah *AssertionHelper) AssertCacheHit(result *query.QueryResult) {
	if !result.CacheHit {
		ah.t.Errorf("expected cache hit, but got cache miss (source: %s)", result.Source)
	}
}

// AssertCacheMiss asserts that a query result did not come from cache
func (ah *AssertionHelper) AssertCacheMiss(result *query.QueryResult) {
	if result.CacheHit {
		ah.t.Errorf("expected cache miss, but got cache hit")
	}
}

// AssertQuerySource asserts that a query result came from the expected source
func (ah *AssertionHelper) AssertQuerySource(result *query.QueryResult, expectedSource string) {
	if result.Source != expectedSource {
		ah.t.Errorf("expected source %s, but got %s", expectedSource, result.Source)
	}
}

// AssertQueryResultCount asserts that a query result contains the expected number of events
func (ah *AssertionHelper) AssertQueryResultCount(result *query.QueryResult, expectedCount int64) {
	if result.Total != expectedCount {
		ah.t.Errorf("expected %d events in result, but got %d", expectedCount, result.Total)
	}
}

// AssertQueryResultCountGreaterThan asserts that a query result contains more than the expected number of events
func (ah *AssertionHelper) AssertQueryResultCountGreaterThan(result *query.QueryResult, minCount int64) {
	if result.Total <= minCount {
		ah.t.Errorf("expected more than %d events in result, but got %d", minCount, result.Total)
	}
}

// AssertCacheKeyExists asserts that a cache key exists
func (ah *AssertionHelper) AssertCacheKeyExists(ctx context.Context, fixture *fixtures.CacheFixture, key string) {
	exists, err := fixture.Exists(ctx, key)
	if err != nil {
		ah.t.Errorf("failed to check cache key existence: %v", err)
		return
	}
	if !exists {
		ah.t.Errorf("expected cache key %s to exist, but it doesn't", key)
	}
}

// AssertCacheKeyNotExists asserts that a cache key does not exist
func (ah *AssertionHelper) AssertCacheKeyNotExists(ctx context.Context, fixture *fixtures.CacheFixture, key string) {
	exists, err := fixture.Exists(ctx, key)
	if err != nil {
		ah.t.Errorf("failed to check cache key existence: %v", err)
		return
	}
	if exists {
		ah.t.Errorf("expected cache key %s to not exist, but it does", key)
	}
}

// AssertCacheValue asserts that a cache key has the expected value
func (ah *AssertionHelper) AssertCacheValue(ctx context.Context, fixture *fixtures.CacheFixture, key string, expectedValue []byte) {
	value, err := fixture.Get(ctx, key)
	if err != nil {
		ah.t.Errorf("failed to get cache key %s: %v", key, err)
		return
	}
	if string(value) != string(expectedValue) {
		ah.t.Errorf("expected cache value %s, but got %s", string(expectedValue), string(value))
	}
}

// AssertMessagePublished asserts that a message was published to a topic
func (ah *AssertionHelper) AssertMessagePublished(fixture *fixtures.MessageQueueFixture, topic string) {
	count := fixture.GetPublishedMessageCount(topic)
	if count == 0 {
		ah.t.Errorf("expected message to be published to topic %s, but no messages were published", topic)
	}
}

// AssertMessageCount asserts that the expected number of messages were published to a topic
func (ah *AssertionHelper) AssertMessageCount(fixture *fixtures.MessageQueueFixture, topic string, expectedCount int) {
	count := fixture.GetPublishedMessageCount(topic)
	if count != expectedCount {
		ah.t.Errorf("expected %d messages on topic %s, but got %d", expectedCount, topic, count)
	}
}

// AssertMessageCountGreaterThan asserts that more than the expected number of messages were published to a topic
func (ah *AssertionHelper) AssertMessageCountGreaterThan(fixture *fixtures.MessageQueueFixture, topic string, minCount int) {
	count := fixture.GetPublishedMessageCount(topic)
	if count <= minCount {
		ah.t.Errorf("expected more than %d messages on topic %s, but got %d", minCount, topic, count)
	}
}

// AssertHealthy asserts that a health status is healthy
func (ah *AssertionHelper) AssertHealthy(status *core.HealthStatus) {
	if status.Status != "healthy" {
		ah.t.Errorf("expected healthy status, but got %s", status.Status)
	}
}

// AssertUnhealthy asserts that a health status is unhealthy
func (ah *AssertionHelper) AssertUnhealthy(status *core.HealthStatus) {
	if status.Status == "healthy" {
		ah.t.Errorf("expected unhealthy status, but got healthy")
	}
}

// AssertNoError asserts that an error is nil
func (ah *AssertionHelper) AssertNoError(err error, message string) {
	if err != nil {
		ah.t.Errorf("%s: %v", message, err)
	}
}

// AssertError asserts that an error is not nil
func (ah *AssertionHelper) AssertError(err error, message string) {
	if err == nil {
		ah.t.Errorf("%s: expected error but got nil", message)
	}
}

// AssertErrorContains asserts that an error message contains a substring
func (ah *AssertionHelper) AssertErrorContains(err error, substring string) {
	if err == nil {
		ah.t.Errorf("expected error containing %s, but got nil", substring)
		return
	}
	if !contains(err.Error(), substring) {
		ah.t.Errorf("expected error containing %s, but got: %v", substring, err)
	}
}

// AssertEqual asserts that two values are equal
func (ah *AssertionHelper) AssertEqual(actual, expected interface{}, message string) {
	if actual != expected {
		ah.t.Errorf("%s: expected %v, but got %v", message, expected, actual)
	}
}

// AssertNotEqual asserts that two values are not equal
func (ah *AssertionHelper) AssertNotEqual(actual, expected interface{}, message string) {
	if actual == expected {
		ah.t.Errorf("%s: expected not equal to %v, but got %v", message, expected, actual)
	}
}

// AssertTrue asserts that a condition is true
func (ah *AssertionHelper) AssertTrue(condition bool, message string) {
	if !condition {
		ah.t.Errorf("%s: expected true, but got false", message)
	}
}

// AssertFalse asserts that a condition is false
func (ah *AssertionHelper) AssertFalse(condition bool, message string) {
	if condition {
		ah.t.Errorf("%s: expected false, but got true", message)
	}
}

// AssertNil asserts that a value is nil
func (ah *AssertionHelper) AssertNil(value interface{}, message string) {
	if value != nil {
		ah.t.Errorf("%s: expected nil, but got %v", message, value)
	}
}

// AssertNotNil asserts that a value is not nil
func (ah *AssertionHelper) AssertNotNil(value interface{}, message string) {
	if value == nil {
		ah.t.Errorf("%s: expected not nil, but got nil", message)
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// AssertEventFields asserts that an event has the expected field values
func (ah *AssertionHelper) AssertEventFields(event *core.BlockchainEvent, expectedChainID string, expectedBlockNum uint64) {
	if event.ChainID != expectedChainID {
		ah.t.Errorf("expected chain ID %s, but got %s", expectedChainID, event.ChainID)
	}
	if event.BlockNumber != expectedBlockNum {
		ah.t.Errorf("expected block number %d, but got %d", expectedBlockNum, event.BlockNumber)
	}
}

// AssertResponseTime asserts that a query response time is within expected bounds
func (ah *AssertionHelper) AssertResponseTime(result *query.QueryResult, maxMillis int64) {
	if result.ResponseTime > maxMillis {
		ah.t.Errorf("expected response time <= %dms, but got %dms", maxMillis, result.ResponseTime)
	}
}

// AssertResponseTimeGreaterThan asserts that a query response time is greater than expected
func (ah *AssertionHelper) AssertResponseTimeGreaterThan(result *query.QueryResult, minMillis int64) {
	if result.ResponseTime <= minMillis {
		ah.t.Errorf("expected response time > %dms, but got %dms", minMillis, result.ResponseTime)
	}
}

// AssertEventIDFormat asserts that an event ID has the expected format
func (ah *AssertionHelper) AssertEventIDFormat(eventID string) {
	if eventID == "" {
		ah.t.Errorf("expected non-empty event ID")
	}
	if len(eventID) < 5 {
		ah.t.Errorf("expected event ID with at least 5 characters, but got %d", len(eventID))
	}
}

// AssertChainIDFormat asserts that a chain ID has the expected format
func (ah *AssertionHelper) AssertChainIDFormat(chainID string) {
	if chainID == "" {
		ah.t.Errorf("expected non-empty chain ID")
	}
	if len(chainID) < 3 {
		ah.t.Errorf("expected chain ID with at least 3 characters, but got %d", len(chainID))
	}
}

// AssertBlockNumberValid asserts that a block number is valid
func (ah *AssertionHelper) AssertBlockNumberValid(blockNum uint64) {
	if blockNum == 0 {
		ah.t.Errorf("expected non-zero block number")
	}
}

// AssertEventNameNotEmpty asserts that an event name is not empty
func (ah *AssertionHelper) AssertEventNameNotEmpty(eventName string) {
	if eventName == "" {
		ah.t.Errorf("expected non-empty event name")
	}
}

// AssertQueryResultNotEmpty asserts that a query result is not empty
func (ah *AssertionHelper) AssertQueryResultNotEmpty(result *query.QueryResult) {
	if result == nil {
		ah.t.Errorf("expected non-nil query result")
		return
	}
	if len(result.Events) == 0 {
		ah.t.Errorf("expected non-empty query result")
	}
}

// AssertQueryResultEmpty asserts that a query result is empty
func (ah *AssertionHelper) AssertQueryResultEmpty(result *query.QueryResult) {
	if result == nil {
		ah.t.Errorf("expected non-nil query result")
		return
	}
	if len(result.Events) > 0 {
		ah.t.Errorf("expected empty query result, but got %d events", len(result.Events))
	}
}


// AssertSliceLength asserts that a slice has the expected length
func (ah *AssertionHelper) AssertSliceLength(slice interface{}, expectedLen int, message string) {
	// Use reflection to get slice length
	switch s := slice.(type) {
	case []interface{}:
		if len(s) != expectedLen {
			ah.t.Errorf("%s: expected length %d, but got %d", message, expectedLen, len(s))
		}
	case []*core.BlockchainEvent:
		if len(s) != expectedLen {
			ah.t.Errorf("%s: expected length %d, but got %d", message, expectedLen, len(s))
		}
	default:
		ah.t.Errorf("%s: unsupported slice type", message)
	}
}

// AssertMapContainsKey asserts that a map contains a key
func (ah *AssertionHelper) AssertMapContainsKey(m map[string]interface{}, key string, message string) {
	if _, exists := m[key]; !exists {
		ah.t.Errorf("%s: expected map to contain key %s", message, key)
	}
}

// AssertMapNotContainsKey asserts that a map does not contain a key
func (ah *AssertionHelper) AssertMapNotContainsKey(m map[string]interface{}, key string, message string) {
	if _, exists := m[key]; exists {
		ah.t.Errorf("%s: expected map to not contain key %s", message, key)
	}
}

// AssertStringNotEmpty asserts that a string is not empty
func (ah *AssertionHelper) AssertStringNotEmpty(s string, message string) {
	if s == "" {
		ah.t.Errorf("%s: expected non-empty string", message)
	}
}

// AssertStringEmpty asserts that a string is empty
func (ah *AssertionHelper) AssertStringEmpty(s string, message string) {
	if s != "" {
		ah.t.Errorf("%s: expected empty string, but got %s", message, s)
	}
}

// AssertStringContains asserts that a string contains a substring
func (ah *AssertionHelper) AssertStringContains(s, substring string, message string) {
	if !contains(s, substring) {
		ah.t.Errorf("%s: expected string to contain %s, but got %s", message, substring, s)
	}
}

// AssertStringNotContains asserts that a string does not contain a substring
func (ah *AssertionHelper) AssertStringNotContains(s, substring string, message string) {
	if contains(s, substring) {
		ah.t.Errorf("%s: expected string to not contain %s, but got %s", message, substring, s)
	}
}

// AssertIntEqual asserts that two integers are equal
func (ah *AssertionHelper) AssertIntEqual(actual, expected int, message string) {
	if actual != expected {
		ah.t.Errorf("%s: expected %d, but got %d", message, expected, actual)
	}
}

// AssertInt64Equal asserts that two int64 values are equal
func (ah *AssertionHelper) AssertInt64Equal(actual, expected int64, message string) {
	if actual != expected {
		ah.t.Errorf("%s: expected %d, but got %d", message, expected, actual)
	}
}

// AssertFloat64Equal asserts that two float64 values are equal within tolerance
func (ah *AssertionHelper) AssertFloat64Equal(actual, expected, tolerance float64, message string) {
	diff := actual - expected
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		ah.t.Errorf("%s: expected %f, but got %f (tolerance: %f)", message, expected, actual, tolerance)
	}
}

// AssertBytesEqual asserts that two byte slices are equal
func (ah *AssertionHelper) AssertBytesEqual(actual, expected []byte, message string) {
	if len(actual) != len(expected) {
		ah.t.Errorf("%s: expected length %d, but got %d", message, len(expected), len(actual))
		return
	}

	for i := range actual {
		if actual[i] != expected[i] {
			ah.t.Errorf("%s: bytes differ at index %d: expected %d, got %d", message, i, expected[i], actual[i])
			return
		}
	}
}

// AssertDurationWithinRange asserts that a duration is within a range
func (ah *AssertionHelper) AssertDurationWithinRange(actual, min, max time.Duration, message string) {
	if actual < min || actual > max {
		ah.t.Errorf("%s: expected duration between %v and %v, but got %v", message, min, max, actual)
	}
}

// AssertPanicRecovered asserts that a function panics
func (ah *AssertionHelper) AssertPanicRecovered(fn func(), message string) {
	defer func() {
		if r := recover(); r == nil {
			ah.t.Errorf("%s: expected panic, but function completed normally", message)
		}
	}()
	fn()
}

// AssertNoPanic asserts that a function does not panic
func (ah *AssertionHelper) AssertNoPanic(fn func(), message string) {
	defer func() {
		if r := recover(); r != nil {
			ah.t.Errorf("%s: expected no panic, but got: %v", message, r)
		}
	}()
	fn()
}

// AssertContextNotCancelled asserts that a context is not cancelled
func (ah *AssertionHelper) AssertContextNotCancelled(ctx context.Context, message string) {
	select {
	case <-ctx.Done():
		ah.t.Errorf("%s: expected context to not be cancelled, but it was", message)
	default:
		// Context is not cancelled
	}
}

// AssertContextCancelled asserts that a context is cancelled
func (ah *AssertionHelper) AssertContextCancelled(ctx context.Context, message string) {
	select {
	case <-ctx.Done():
		// Context is cancelled
	default:
		ah.t.Errorf("%s: expected context to be cancelled, but it wasn't", message)
	}
}

// AssertEventIDsUnique asserts that all event IDs are unique
func (ah *AssertionHelper) AssertEventIDsUnique(events []*core.BlockchainEvent, message string) {
	seen := make(map[string]bool)
	for _, event := range events {
		if seen[event.ID] {
			ah.t.Errorf("%s: found duplicate event ID: %s", message, event.ID)
			return
		}
		seen[event.ID] = true
	}
}

// AssertEventsSorted asserts that events are sorted by block number
func (ah *AssertionHelper) AssertEventsSorted(events []*core.BlockchainEvent, message string) {
	for i := 1; i < len(events); i++ {
		if events[i].BlockNumber < events[i-1].BlockNumber {
			ah.t.Errorf("%s: events not sorted by block number at index %d", message, i)
			return
		}
	}
}

// AssertEventChainIDConsistent asserts that all events have the same chain ID
func (ah *AssertionHelper) AssertEventChainIDConsistent(events []*core.BlockchainEvent, expectedChainID string, message string) {
	for i, event := range events {
		if event.ChainID != expectedChainID {
			ah.t.Errorf("%s: event at index %d has chain ID %s, expected %s", message, i, event.ChainID, expectedChainID)
			return
		}
	}
}

// AssertResponseTimeReasonable asserts that response time is reasonable
func (ah *AssertionHelper) AssertResponseTimeReasonable(result *query.QueryResult, maxMillis int64, message string) {
	if result.ResponseTime > maxMillis {
		ah.t.Errorf("%s: response time %dms exceeds maximum %dms", message, result.ResponseTime, maxMillis)
	}
}

// AssertCacheHitRate asserts that cache hit rate is within expected range
func (ah *AssertionHelper) AssertCacheHitRate(hits, total int64, minRate float64, message string) {
	if total == 0 {
		ah.t.Errorf("%s: total requests is zero", message)
		return
	}

	rate := float64(hits) / float64(total)
	if rate < minRate {
		ah.t.Errorf("%s: cache hit rate %.2f%% is below minimum %.2f%%", message, rate*100, minRate*100)
	}
}

// AssertEventDataNotNil asserts that event data is not nil
func (ah *AssertionHelper) AssertEventDataNotNil(event *core.BlockchainEvent, message string) {
	if event == nil {
		ah.t.Errorf("%s: event is nil", message)
		return
	}
	if event.ID == "" {
		ah.t.Errorf("%s: event ID is empty", message)
	}
	if event.ChainID == "" {
		ah.t.Errorf("%s: event chain ID is empty", message)
	}
}

// AssertQueryResultValid asserts that a query result is valid
func (ah *AssertionHelper) AssertQueryResultValid(result *query.QueryResult, message string) {
	if result == nil {
		ah.t.Errorf("%s: query result is nil", message)
		return
	}
	if result.Total < 0 {
		ah.t.Errorf("%s: query result total is negative", message)
	}
	if result.ResponseTime < 0 {
		ah.t.Errorf("%s: query result response time is negative", message)
	}
}

// AssertNoErrorWithContext asserts that an error is nil with context
func (ah *AssertionHelper) AssertNoErrorWithContext(err error, context string, message string) {
	if err != nil {
		ah.t.Errorf("%s: %s - %v", message, context, err)
	}
}

// AssertErrorWithContext asserts that an error is not nil with context
func (ah *AssertionHelper) AssertErrorWithContext(err error, context string, message string) {
	if err == nil {
		ah.t.Errorf("%s: %s - expected error but got nil", message, context)
	}
}

// AssertErrorType asserts that an error is of a specific type
func (ah *AssertionHelper) AssertErrorType(err error, expectedType string, message string) {
	if err == nil {
		ah.t.Errorf("%s: expected error of type %s, but got nil", message, expectedType)
		return
	}

	actualType := fmt.Sprintf("%T", err)
	if !contains(actualType, expectedType) {
		ah.t.Errorf("%s: expected error type %s, but got %s", message, expectedType, actualType)
	}
}

// AssertSliceNotEmpty asserts that a slice is not empty
func (ah *AssertionHelper) AssertSliceNotEmpty(slice interface{}, message string) {
	switch s := slice.(type) {
	case []interface{}:
		if len(s) == 0 {
			ah.t.Errorf("%s: expected non-empty slice", message)
		}
	case []*core.BlockchainEvent:
		if len(s) == 0 {
			ah.t.Errorf("%s: expected non-empty slice", message)
		}
	default:
		ah.t.Errorf("%s: unsupported slice type", message)
	}
}

// AssertSliceEmpty asserts that a slice is empty
func (ah *AssertionHelper) AssertSliceEmpty(slice interface{}, message string) {
	switch s := slice.(type) {
	case []interface{}:
		if len(s) != 0 {
			ah.t.Errorf("%s: expected empty slice, but got %d elements", message, len(s))
		}
	case []*core.BlockchainEvent:
		if len(s) != 0 {
			ah.t.Errorf("%s: expected empty slice, but got %d elements", message, len(s))
		}
	default:
		ah.t.Errorf("%s: unsupported slice type", message)
	}
}

// AssertMapNotEmpty asserts that a map is not empty
func (ah *AssertionHelper) AssertMapNotEmpty(m map[string]interface{}, message string) {
	if len(m) == 0 {
		ah.t.Errorf("%s: expected non-empty map", message)
	}
}

// AssertMapEmpty asserts that a map is empty
func (ah *AssertionHelper) AssertMapEmpty(m map[string]interface{}, message string) {
	if len(m) != 0 {
		ah.t.Errorf("%s: expected empty map, but got %d elements", message, len(m))
	}
}

// AssertUint64Equal asserts that two uint64 values are equal
func (ah *AssertionHelper) AssertUint64Equal(actual, expected uint64, message string) {
	if actual != expected {
		ah.t.Errorf("%s: expected %d, but got %d", message, expected, actual)
	}
}

// AssertUint64GreaterThan asserts that a uint64 value is greater than expected
func (ah *AssertionHelper) AssertUint64GreaterThan(actual, expected uint64, message string) {
	if actual <= expected {
		ah.t.Errorf("%s: expected greater than %d, but got %d", message, expected, actual)
	}
}

// AssertUint64LessThan asserts that a uint64 value is less than expected
func (ah *AssertionHelper) AssertUint64LessThan(actual, expected uint64, message string) {
	if actual >= expected {
		ah.t.Errorf("%s: expected less than %d, but got %d", message, expected, actual)
	}
}

// AssertTimeWithinRange asserts that a time is within a range
func (ah *AssertionHelper) AssertTimeWithinRange(actual, min, max time.Time, message string) {
	if actual.Before(min) || actual.After(max) {
		ah.t.Errorf("%s: expected time between %v and %v, but got %v", message, min, max, actual)
	}
}

// AssertTimeAfter asserts that a time is after another time
func (ah *AssertionHelper) AssertTimeAfter(actual, expected time.Time, message string) {
	if !actual.After(expected) {
		ah.t.Errorf("%s: expected time after %v, but got %v", message, expected, actual)
	}
}

// AssertTimeBefore asserts that a time is before another time
func (ah *AssertionHelper) AssertTimeBefore(actual, expected time.Time, message string) {
	if !actual.Before(expected) {
		ah.t.Errorf("%s: expected time before %v, but got %v", message, expected, actual)
	}
}
