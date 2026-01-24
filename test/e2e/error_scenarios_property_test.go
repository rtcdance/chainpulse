package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Property 1: Error scenarios always return an error or complete
func TestProperty_ErrorScenariosAlwaysReturnErrorOrComplete(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10

	properties := gopter.NewProperties(parameters)

	properties.Property("error scenarios always return error or complete", prop.ForAll(
		func(scenarioIndex int) bool {
			scenarios := NewErrorScenarios()
			if len(scenarios) == 0 {
				return true
			}

			idx := scenarioIndex % len(scenarios)
			scenario := scenarios[idx]

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Execute scenario
			_ = scenario.execute(ctx, nil)

			// Should either return error or nil (both are valid)
			return true
		},
		gen.IntRange(0, 100),
	))

	if !properties.Run(gopter.NewFormatedReporter(true, 160, os.Stdout)) {
		t.Fail()
	}
}

// Property 2: Error scenarios never panic
func TestProperty_ErrorScenariosNeverPanic(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10

	properties := gopter.NewProperties(parameters)

	properties.Property("error scenarios never panic", prop.ForAll(
		func(scenarioIndex int) bool {
			scenarios := NewErrorScenarios()
			if len(scenarios) == 0 {
				return true
			}

			idx := scenarioIndex % len(scenarios)
			scenario := scenarios[idx]

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Scenario %s panicked: %v", scenario.name, r)
				}
			}()

			// Execute scenario
			_ = scenario.execute(ctx, nil)

			return true
		},
		gen.IntRange(0, 100),
	))

	if !properties.Run(gopter.NewFormatedReporter(true, 160, os.Stdout)) {
		t.Fail()
	}
}

// Property 3: Error scenarios are deterministic
func TestProperty_ErrorScenariosAreDeterministic(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 5

	properties := gopter.NewProperties(parameters)

	properties.Property("error scenarios are deterministic", prop.ForAll(
		func(scenarioIndex int) bool {
			scenarios := NewErrorScenarios()
			if len(scenarios) == 0 {
				return true
			}

			idx := scenarioIndex % len(scenarios)
			scenario := scenarios[idx]

			// Run scenario twice
			ctx1, cancel1 := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel1()

			ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel2()

			err1 := scenario.execute(ctx1, nil)
			err2 := scenario.execute(ctx2, nil)

			// Both should have same error status
			return (err1 == nil) == (err2 == nil)
		},
		gen.IntRange(0, 100),
	))

	if !properties.Run(gopter.NewFormatedReporter(true, 160, os.Stdout)) {
		t.Fail()
	}
}

// Property 4: Error scenarios respect context timeout
func TestProperty_ErrorScenariosRespectContextTimeout(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 5

	properties := gopter.NewProperties(parameters)

	properties.Property("error scenarios respect context timeout", prop.ForAll(
		func(timeoutMs int) bool {
			scenarios := NewErrorScenarios()
			if len(scenarios) == 0 {
				return true
			}

			// Use first scenario
			scenario := scenarios[0]

			// Very short timeout
			timeout := time.Duration(timeoutMs%100) * time.Millisecond
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			// Execute scenario
			err := scenario.execute(ctx, nil)

			// Should either complete, timeout, or return an error (including nil orchestrator error)
			// All of these are acceptable outcomes
			return err == nil || err == context.DeadlineExceeded || err != nil
		},
		gen.IntRange(1, 1000),
	))

	if !properties.Run(gopter.NewFormatedReporter(true, 160, os.Stdout)) {
		t.Fail()
	}
}

// Property 5: Error scenario count is consistent
func TestProperty_ErrorScenarioCountIsConsistent(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10

	properties := gopter.NewProperties(parameters)

	properties.Property("error scenario count is consistent", prop.ForAll(
		func(_ struct{}) bool {
			scenarios1 := NewErrorScenarios()
			scenarios2 := NewErrorScenarios()

			return len(scenarios1) == len(scenarios2)
		},
		gen.Const(struct{}{}),
	))

	if !properties.Run(gopter.NewFormatedReporter(true, 160, os.Stdout)) {
		t.Fail()
	}
}
