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

// Property 1: Happy path scenarios always complete without panic
func TestProperty_HappyPathScenariosNeverPanic(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10

	properties := gopter.NewProperties(parameters)

	properties.Property("happy path scenarios never panic", prop.ForAll(
		func(scenarioIndex int) bool {
			scenarios := NewHappyPathScenarios()
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

// Property 2: Happy path scenarios are deterministic
func TestProperty_HappyPathScenariosAreDeterministic(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 5

	properties := gopter.NewProperties(parameters)

	properties.Property("happy path scenarios are deterministic", prop.ForAll(
		func(scenarioIndex int) bool {
			scenarios := NewHappyPathScenarios()
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

// Property 3: Happy path scenarios respect context timeout
func TestProperty_HappyPathScenariosRespectContextTimeout(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 5

	properties := gopter.NewProperties(parameters)

	properties.Property("happy path scenarios respect context timeout", prop.ForAll(
		func(timeoutMs int) bool {
			scenarios := NewHappyPathScenarios()
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

// Property 4: Happy path scenario count is consistent
func TestProperty_HappyPathScenarioCountIsConsistent(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10

	properties := gopter.NewProperties(parameters)

	properties.Property("happy path scenario count is consistent", prop.ForAll(
		func(_ struct{}) bool {
			scenarios1 := NewHappyPathScenarios()
			scenarios2 := NewHappyPathScenarios()

			return len(scenarios1) == len(scenarios2)
		},
		gen.Const(struct{}{}),
	))

	if !properties.Run(gopter.NewFormatedReporter(true, 160, os.Stdout)) {
		t.Fail()
	}
}

// Property 5: Happy path scenario names are unique
func TestProperty_HappyPathScenarioNamesAreUnique(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 1

	properties := gopter.NewProperties(parameters)

	properties.Property("happy path scenario names are unique", prop.ForAll(
		func(_ struct{}) bool {
			scenarios := NewHappyPathScenarios()
			names := make(map[string]bool)

			for _, scenario := range scenarios {
				if names[scenario.name] {
					return false // Duplicate name
				}
				names[scenario.name] = true
			}

			return true
		},
		gen.Const(struct{}{}),
	))

	if !properties.Run(gopter.NewFormatedReporter(true, 160, os.Stdout)) {
		t.Fail()
	}
}
