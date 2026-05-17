package core_test

import (
	"context"
	"fmt"

	"chainpulse/pkg/core"
)

// ExampleNewEventBus demonstrates basic publish/subscribe with PublishSync.
func ExampleNewEventBus() {
	logger := core.NewSlogLogger(core.LogLevelInfo, "json")
	eb := core.NewEventBus(logger)

	// Subscribe to a topic
	_, _ = eb.Subscribe(context.Background(), "greetings", func(payload any) {
		msg, ok := payload.(string)
		if ok {
			fmt.Println("received:", msg)
		}
	})

	// PublishSync runs handlers synchronously — output is deterministic
	_ = eb.PublishSync(context.Background(), "greetings", "hello, world!")

	// Output:
	// received: hello, world!
}

// ExampleNewSystemError demonstrates creating and classifying errors.
func ExampleNewSystemError() {
	// Transient error — retryable (e.g., network timeout)
	err := core.NewSystemError(
		core.ErrorTypeTransient,
		core.ErrorCodeTimeout,
		"connection timed out",
		nil,
	)
	fmt.Println(err.Error())

	// Permanent error — not retryable (e.g., invalid input)
	err2 := core.NewSystemError(
		core.ErrorTypePermanent,
		core.ErrorCodeValidation,
		"invalid block number",
		nil,
	)
	fmt.Println(err2.Error())

	// Output:
	// [transient] TIMEOUT: connection timed out
	// [permanent] VALIDATION_ERROR: invalid block number
}
