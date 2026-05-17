package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWSSubscriptionCloseOnce verifies that closing a subscription's done channel
// via closeOnce.Do is idempotent and does not panic on double close
func TestWSSubscriptionCloseOnce(t *testing.T) {
	t.Parallel()
	sub := &WSSubscription{
		id:    "test-sub",
		topic: "test-topic",
		done:  make(chan struct{}),
	}

	// First close should succeed
	sub.closeOnce.Do(func() { close(sub.done) })
	assert.True(t, isClosed(sub.done), "done channel should be closed after first close")

	// Second close should NOT panic — this is the core fix
	assert.NotPanics(t, func() {
		sub.closeOnce.Do(func() { close(sub.done) })
	}, "double close via sync.Once should not panic")
}

// TestWSSubscriptionCloseOnceConcurrent verifies that concurrent close attempts
// don't cause a panic
func TestWSSubscriptionCloseOnceConcurrent(t *testing.T) {
	t.Parallel()
	sub := &WSSubscription{
		id:    "concurrent-sub",
		topic: "test-topic",
		done:  make(chan struct{}),
	}

	// Launch multiple goroutines trying to close the same subscription
	done := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func() {
			sub.closeOnce.Do(func() { close(sub.done) })
			done <- struct{}{}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.True(t, isClosed(sub.done), "done channel should be closed after concurrent closes")
}

// isClosed checks if a channel is closed without blocking
func isClosed(ch chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}
