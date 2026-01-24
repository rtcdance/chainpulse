package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Property 1: Scenario execution is deterministic
func TestScenarioExecutionDeterminism(t *testing.T) {
	iterations := 100

	for iter := 0; iter < iterations; iter++ {
		executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

		executionOrder := []int{}
		scenario := NewScenarioBuilder(fmt.Sprintf("scenario_%d", iter), ScenarioTypeHappyPath).
			AddStep(ScenarioStep{
				Name: "step1",
				Action: func(ctx context.Context) error {
					executionOrder = append(executionOrder, 1)
					return nil
				},
			}).
			AddStep(ScenarioStep{
				Name: "step2",
				Action: func(ctx context.Context) error {
					executionOrder = append(executionOrder, 2)
					return nil
				},
			}).
			AddStep(ScenarioStep{
				Name: "step3",
				Action: func(ctx context.Context) error {
					executionOrder = append(executionOrder, 3)
					return nil
				},
			}).
			Build()

		if err := executor.RegisterScenario(scenario); err != nil {
			t.Fatalf("RegisterScenario failed: %v", err)
		}
		result, _ := executor.ExecuteScenario(context.Background(), scenario.Name)

		// Verify deterministic execution
		if len(executionOrder) != 3 {
			t.Fatalf("Iteration %d: Expected 3 steps, got %d", iter, len(executionOrder))
		}

		if executionOrder[0] != 1 || executionOrder[1] != 2 || executionOrder[2] != 3 {
			t.Fatalf("Iteration %d: Steps executed in wrong order: %v", iter, executionOrder)
		}

		if result.StepsRun != 3 {
			t.Fatalf("Iteration %d: Expected 3 steps run, got %d", iter, result.StepsRun)
		}
	}
}

// Property 2: Setup always runs before steps
func TestScenarioSetupBeforeSteps(t *testing.T) {
	iterations := 100

	for iter := 0; iter < iterations; iter++ {
		executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

		executionOrder := []string{}
		scenario := NewScenarioBuilder(fmt.Sprintf("scenario_%d", iter), ScenarioTypeHappyPath).
			WithSetup(func(ctx context.Context) error {
				executionOrder = append(executionOrder, "setup")
				return nil
			}).
			AddStep(ScenarioStep{
				Name: "step1",
				Action: func(ctx context.Context) error {
					executionOrder = append(executionOrder, "step1")
					return nil
				},
			}).
			Build()

		if err := executor.RegisterScenario(scenario); err != nil {
			t.Fatalf("RegisterScenario failed: %v", err)
		}
		_, _ = executor.ExecuteScenario(context.Background(), scenario.Name)

		// Verify setup runs first
		if len(executionOrder) < 2 || executionOrder[0] != "setup" {
			t.Fatalf("Iteration %d: Setup did not run first: %v", iter, executionOrder)
		}
	}
}

// Property 3: Teardown always runs after steps
func TestScenarioTeardownAfterSteps(t *testing.T) {
	iterations := 100

	for iter := 0; iter < iterations; iter++ {
		executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

		executionOrder := []string{}
		scenario := NewScenarioBuilder(fmt.Sprintf("scenario_%d", iter), ScenarioTypeHappyPath).
			AddStep(ScenarioStep{
				Name: "step1",
				Action: func(ctx context.Context) error {
					executionOrder = append(executionOrder, "step1")
					return nil
				},
			}).
			WithTeardown(func(ctx context.Context) error {
				executionOrder = append(executionOrder, "teardown")
				return nil
			}).
			Build()

		_ = executor.RegisterScenario(scenario)
		_, _ = executor.ExecuteScenario(context.Background(), scenario.Name)

		// Verify teardown runs last
		if len(executionOrder) < 2 || executionOrder[len(executionOrder)-1] != "teardown" {
			t.Fatalf("Iteration %d: Teardown did not run last: %v", iter, executionOrder)
		}
	}
}

// Property 4: Failed steps stop execution
func TestScenarioFailureStopsExecution(t *testing.T) {
	iterations := 100

	for iter := 0; iter < iterations; iter++ {
		executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

		executedSteps := 0
		scenario := NewScenarioBuilder(fmt.Sprintf("scenario_%d", iter), ScenarioTypeHappyPath).
			AddStep(ScenarioStep{
				Name: "step1",
				Action: func(ctx context.Context) error {
					executedSteps++
					return nil
				},
			}).
			AddStep(ScenarioStep{
				Name: "step2",
				Action: func(ctx context.Context) error {
					executedSteps++
					return fmt.Errorf("step2 failed")
				},
			}).
			AddStep(ScenarioStep{
				Name: "step3",
				Action: func(ctx context.Context) error {
					executedSteps++
					return nil
				},
			}).
			Build()

		if err := executor.RegisterScenario(scenario); err != nil {
			t.Fatalf("RegisterScenario failed: %v", err)
		}
		result, _ := executor.ExecuteScenario(context.Background(), scenario.Name)

		// Verify execution stopped at failure
		if executedSteps != 2 {
			t.Fatalf("Iteration %d: Expected 2 steps executed, got %d", iter, executedSteps)
		}

		if result.StepsFailed != 1 {
			t.Fatalf("Iteration %d: Expected 1 failed step, got %d", iter, result.StepsFailed)
		}
	}
}

// Property 5: Validation failures stop execution
func TestScenarioValidationFailureStopsExecution(t *testing.T) {
	iterations := 100

	for iter := 0; iter < iterations; iter++ {
		executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

		executedSteps := 0
		scenario := NewScenarioBuilder(fmt.Sprintf("scenario_%d", iter), ScenarioTypeHappyPath).
			AddStep(ScenarioStep{
				Name: "step1",
				Action: func(ctx context.Context) error {
					executedSteps++
					return nil
				},
				Validate: func(ctx context.Context) error {
					return nil
				},
			}).
			AddStep(ScenarioStep{
				Name: "step2",
				Action: func(ctx context.Context) error {
					executedSteps++
					return nil
				},
				Validate: func(ctx context.Context) error {
					return fmt.Errorf("validation failed")
				},
			}).
			AddStep(ScenarioStep{
				Name: "step3",
				Action: func(ctx context.Context) error {
					executedSteps++
					return nil
				},
			}).
			Build()

		if err := executor.RegisterScenario(scenario); err != nil {
			t.Fatalf("RegisterScenario failed: %v", err)
		}
		result, _ := executor.ExecuteScenario(context.Background(), scenario.Name)

		// Verify execution stopped at validation failure
		if executedSteps != 2 {
			t.Fatalf("Iteration %d: Expected 2 steps executed, got %d", iter, executedSteps)
		}

		if result.Status != "FAILED" {
			t.Fatalf("Iteration %d: Expected FAILED status, got %s", iter, result.Status)
		}
	}
}

// Property 6: Scenario results are recorded
func TestScenarioResultsRecorded(t *testing.T) {
	iterations := 100

	for iter := 0; iter < iterations; iter++ {
		executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

		for i := 0; i < 5; i++ {
			scenario := NewScenarioBuilder(fmt.Sprintf("scenario_%d_%d", iter, i), ScenarioTypeHappyPath).
				AddStep(ScenarioStep{
					Name: "step1",
					Action: func(ctx context.Context) error {
						return nil
					},
				}).
				Build()
			_ = executor.RegisterScenario(scenario)
			_, _ = executor.ExecuteScenario(context.Background(), scenario.Name)
		}

		results := executor.GetResults()
		if len(results) != 5 {
			t.Fatalf("Iteration %d: Expected 5 results, got %d", iter, len(results))
		}

		for _, result := range results {
			if result.Status != "PASSED" {
				t.Fatalf("Iteration %d: Expected PASSED status, got %s", iter, result.Status)
			}
		}
	}
}

// Property 7: Summary statistics are accurate
func TestScenarioSummaryAccuracy(t *testing.T) {
	iterations := 100

	for iter := 0; iter < iterations; iter++ {
		executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

		passCount := 0
		failCount := 0

		// Register passing scenarios
		for i := 0; i < 3; i++ {
			scenario := NewScenarioBuilder(fmt.Sprintf("pass_%d_%d", iter, i), ScenarioTypeHappyPath).
				AddStep(ScenarioStep{
					Name: "step1",
					Action: func(ctx context.Context) error {
						return nil
					},
				}).
				Build()
			_ = executor.RegisterScenario(scenario)
			_, _ = executor.ExecuteScenario(context.Background(), scenario.Name)
			passCount++
		}

		// Register failing scenarios
		for i := 0; i < 2; i++ {
			scenario := NewScenarioBuilder(fmt.Sprintf("fail_%d_%d", iter, i), ScenarioTypeErrorPath).
				AddStep(ScenarioStep{
					Name: "step1",
					Action: func(ctx context.Context) error {
						return fmt.Errorf("intentional failure")
					},
				}).
				Build()
			if err := executor.RegisterScenario(scenario); err != nil {
				t.Fatalf("RegisterScenario failed: %v", err)
			}
			_, _ = executor.ExecuteScenario(context.Background(), scenario.Name)
			failCount++
		}

		summary := executor.GetSummary()

		if summary["total_scenarios"] != passCount+failCount {
			t.Fatalf("Iteration %d: Expected %d total scenarios, got %v", iter, passCount+failCount, summary["total_scenarios"])
		}

		if summary["passed_scenarios"] != passCount {
			t.Fatalf("Iteration %d: Expected %d passed scenarios, got %v", iter, passCount, summary["passed_scenarios"])
		}

		if summary["failed_scenarios"] != failCount {
			t.Fatalf("Iteration %d: Expected %d failed scenarios, got %v", iter, failCount, summary["failed_scenarios"])
		}

		expectedRate := float64(passCount) / float64(passCount+failCount) * 100
		if summary["success_rate"] != expectedRate {
			t.Fatalf("Iteration %d: Expected %.2f%% success rate, got %v", iter, expectedRate, summary["success_rate"])
		}
	}
}

// Property 8: Scenario type filtering works correctly
func TestScenarioTypeFiltering(t *testing.T) {
	iterations := 100

	for iter := 0; iter < iterations; iter++ {
		executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

		types := []ScenarioType{
			ScenarioTypeHappyPath,
			ScenarioTypeErrorPath,
			ScenarioTypePerformance,
			ScenarioTypeMultiChain,
		}

		for typeIdx, scenarioType := range types {
			for i := 0; i < 2; i++ {
				scenario := NewScenarioBuilder(
					fmt.Sprintf("scenario_%d_%d_%d", iter, typeIdx, i),
					scenarioType,
				).
					AddStep(ScenarioStep{
						Name: "step1",
						Action: func(ctx context.Context) error {
							return nil
						},
					}).
					Build()
				if err := executor.RegisterScenario(scenario); err != nil {
					t.Fatalf("RegisterScenario failed: %v", err)
				}
				_, _ = executor.ExecuteScenario(context.Background(), scenario.Name)
			}
		}

		// Test filtering for each type
		for _, scenarioType := range types {
			results, _ := executor.ExecuteScenariosByType(context.Background(), scenarioType)

			for _, result := range results {
				if result.Type != scenarioType {
					t.Fatalf("Iteration %d: Expected type %s, got %s", iter, scenarioType, result.Type)
				}
			}
		}
	}
}

// Property 9: Scenario duration is measured
func TestScenarioDurationMeasurement(t *testing.T) {
	iterations := 100

	for iter := 0; iter < iterations; iter++ {
		executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

		scenario := NewScenarioBuilder(fmt.Sprintf("scenario_%d", iter), ScenarioTypeHappyPath).
			AddStep(ScenarioStep{
				Name: "step1",
				Action: func(ctx context.Context) error {
					time.Sleep(10 * time.Millisecond)
					return nil
				},
			}).
			Build()

		if err := executor.RegisterScenario(scenario); err != nil {
			t.Fatalf("RegisterScenario failed: %v", err)
		}
		result, _ := executor.ExecuteScenario(context.Background(), scenario.Name)

		if result.Duration < 10*time.Millisecond {
			t.Fatalf("Iteration %d: Expected duration >= 10ms, got %v", iter, result.Duration)
		}
	}
}

// Property 10: Metrics are collected during execution
func TestScenarioMetricsCollection(t *testing.T) {
	iterations := 100

	for iter := 0; iter < iterations; iter++ {
		executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

		scenario := NewScenarioBuilder(fmt.Sprintf("scenario_%d", iter), ScenarioTypeHappyPath).
			AddStep(ScenarioStep{
				Name: "step1",
				Action: func(ctx context.Context) error {
					return nil
				},
			}).
			AddStep(ScenarioStep{
				Name: "step2",
				Action: func(ctx context.Context) error {
					return nil
				},
			}).
			Build()

		_ = executor.RegisterScenario(scenario)
		result, _ := executor.ExecuteScenario(context.Background(), scenario.Name)

		if result.Metrics == nil {
			t.Fatalf("Iteration %d: Metrics not collected", iter)
		}

		// Check that the scenario executed successfully
		// Status should be "PASSED" or similar success indicator
		if result.Status != "PASSED" && result.Status != "SUCCESS" && result.Error != nil {
			t.Fatalf("Iteration %d: Expected successful execution, got status: %s, error: %v", iter, result.Status, result.Error)
		}
	}
}

// Property 11: Scenario builder is idempotent
func TestScenarioBuilderIdempotence(t *testing.T) {
	iterations := 100

	for iter := 0; iter < iterations; iter++ {
		builder := NewScenarioBuilder(fmt.Sprintf("scenario_%d", iter), ScenarioTypeHappyPath)

		// Build multiple times
		scenario1 := builder.Build()
		scenario2 := builder.Build()

		if scenario1.Name != scenario2.Name {
			t.Fatalf("Iteration %d: Scenario names differ", iter)
		}

		if scenario1.Type != scenario2.Type {
			t.Fatalf("Iteration %d: Scenario types differ", iter)
		}
	}
}

// Property 12: Concurrent scenario execution is safe
func TestConcurrentScenarioExecution(t *testing.T) {
	iterations := 50

	for iter := 0; iter < iterations; iter++ {
		executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

		// Register scenarios
		for i := 0; i < 10; i++ {
			scenario := NewScenarioBuilder(fmt.Sprintf("scenario_%d_%d", iter, i), ScenarioTypeHappyPath).
				AddStep(ScenarioStep{
					Name: "step1",
					Action: func(ctx context.Context) error {
						return nil
					},
				}).
				Build()
			_ = executor.RegisterScenario(scenario)
		}

		// Execute all scenarios
		results, _ := executor.ExecuteAllScenarios(context.Background())

		if len(results) != 10 {
			t.Fatalf("Iteration %d: Expected 10 results, got %d", iter, len(results))
		}

		for _, result := range results {
			if result.Status != "PASSED" {
				t.Fatalf("Iteration %d: Expected PASSED status, got %s", iter, result.Status)
			}
		}
	}
}
