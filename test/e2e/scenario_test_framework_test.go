package e2e

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestScenarioExecutorRegisterScenario(t *testing.T) {
	executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))
	scenario := &Scenario{
		Name: "test_scenario",
		Type: ScenarioTypeHappyPath,
	}

	err := executor.RegisterScenario(scenario)
	if err != nil {
		t.Fatalf("RegisterScenario failed: %v", err)
	}

	// Verify scenario was registered
	executor.mu.RLock()
	_, exists := executor.scenarios["test_scenario"]
	executor.mu.RUnlock()

	if !exists {
		t.Error("Scenario not registered")
	}
}

func TestScenarioExecutorRegisterDuplicate(t *testing.T) {
	executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))
	scenario := &Scenario{
		Name: "test_scenario",
		Type: ScenarioTypeHappyPath,
	}

	_ = executor.RegisterScenario(scenario)
	err := executor.RegisterScenario(scenario)

	if err == nil {
		t.Error("Expected error when registering duplicate scenario")
	}
}

func TestScenarioExecutorExecuteScenario(t *testing.T) {
	executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

	scenario := NewScenarioBuilder("test_scenario", ScenarioTypeHappyPath).
		AddStep(ScenarioStep{
			Name: "step1",
			Action: func(ctx context.Context) error {
				return nil
			},
			Validate: func(ctx context.Context) error {
				return nil
			},
		}).
		Build()

	_ = executor.RegisterScenario(scenario)

	result, err := executor.ExecuteScenario(context.Background(), "test_scenario")
	if err != nil {
		t.Fatalf("ExecuteScenario failed: %v", err)
	}

	if result.Status != "PASSED" {
		t.Errorf("Expected status PASSED, got %s", result.Status)
	}

	if result.StepsRun != 1 {
		t.Errorf("Expected 1 step run, got %d", result.StepsRun)
	}
}

func TestScenarioExecutorExecuteScenarioNotFound(t *testing.T) {
	executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

	_, err := executor.ExecuteScenario(context.Background(), "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent scenario")
	}
}

func TestScenarioExecutorExecuteScenarioWithSetup(t *testing.T) {
	executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

	setupCalled := false
	scenario := NewScenarioBuilder("test_scenario", ScenarioTypeHappyPath).
		WithSetup(func(ctx context.Context) error {
			setupCalled = true
			return nil
		}).
		AddStep(ScenarioStep{
			Name: "step1",
			Action: func(ctx context.Context) error {
				return nil
			},
		}).
		Build()

	_ = executor.RegisterScenario(scenario)
	_, _ = executor.ExecuteScenario(context.Background(), "test_scenario")

	if !setupCalled {
		t.Error("Setup was not called")
	}
}

func TestScenarioExecutorExecuteScenarioWithTeardown(t *testing.T) {
	executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

	teardownCalled := false
	scenario := NewScenarioBuilder("test_scenario", ScenarioTypeHappyPath).
		AddStep(ScenarioStep{
			Name: "step1",
			Action: func(ctx context.Context) error {
				return nil
			},
		}).
		WithTeardown(func(ctx context.Context) error {
			teardownCalled = true
			return nil
		}).
		Build()

	_ = executor.RegisterScenario(scenario)
	_, _ = executor.ExecuteScenario(context.Background(), "test_scenario")

	if !teardownCalled {
		t.Error("Teardown was not called")
	}
}

func TestScenarioExecutorExecuteScenarioActionFails(t *testing.T) {
	executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

	scenario := NewScenarioBuilder("test_scenario", ScenarioTypeHappyPath).
		AddStep(ScenarioStep{
			Name: "step1",
			Action: func(ctx context.Context) error {
				return errors.New("action failed")
			},
		}).
		Build()

	_ = executor.RegisterScenario(scenario)
	result, _ := executor.ExecuteScenario(context.Background(), "test_scenario")

	if result.Status != "FAILED" {
		t.Errorf("Expected status FAILED, got %s", result.Status)
	}

	if result.StepsFailed != 1 {
		t.Errorf("Expected 1 failed step, got %d", result.StepsFailed)
	}
}

func TestScenarioExecutorExecuteScenarioValidationFails(t *testing.T) {
	executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

	scenario := NewScenarioBuilder("test_scenario", ScenarioTypeHappyPath).
		AddStep(ScenarioStep{
			Name: "step1",
			Action: func(ctx context.Context) error {
				return nil
			},
			Validate: func(ctx context.Context) error {
				return errors.New("validation failed")
			},
		}).
		Build()

	_ = executor.RegisterScenario(scenario)
	result, _ := executor.ExecuteScenario(context.Background(), "test_scenario")

	if result.Status != "FAILED" {
		t.Errorf("Expected status FAILED, got %s", result.Status)
	}
}

func TestScenarioExecutorExecuteAllScenarios(t *testing.T) {
	executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

	for i := 0; i < 3; i++ {
		scenario := NewScenarioBuilder("scenario_"+string(rune(i)), ScenarioTypeHappyPath).
			AddStep(ScenarioStep{
				Name: "step1",
				Action: func(ctx context.Context) error {
					return nil
				},
			}).
			Build()
		_ = executor.RegisterScenario(scenario)
	}

	results, err := executor.ExecuteAllScenarios(context.Background())
	if err != nil {
		t.Fatalf("ExecuteAllScenarios failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
}

func TestScenarioExecutorExecuteScenariosByType(t *testing.T) {
	executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

	// Register scenarios of different types
	for i := 0; i < 2; i++ {
		scenario := NewScenarioBuilder("happy_"+string(rune(i)), ScenarioTypeHappyPath).
			AddStep(ScenarioStep{
				Name: "step1",
				Action: func(ctx context.Context) error {
					return nil
				},
			}).
			Build()
		_ = executor.RegisterScenario(scenario)
	}

	for i := 0; i < 2; i++ {
		scenario := NewScenarioBuilder("error_"+string(rune(i)), ScenarioTypeErrorPath).
			AddStep(ScenarioStep{
				Name: "step1",
				Action: func(ctx context.Context) error {
					return nil
				},
			}).
			Build()
		_ = executor.RegisterScenario(scenario)
	}

	results, err := executor.ExecuteScenariosByType(context.Background(), ScenarioTypeHappyPath)
	if err != nil {
		t.Fatalf("ExecuteScenariosByType failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 happy path results, got %d", len(results))
	}

	for _, result := range results {
		if result.Type != ScenarioTypeHappyPath {
			t.Errorf("Expected type HappyPath, got %s", result.Type)
		}
	}
}

func TestScenarioExecutorGetResults(t *testing.T) {
	executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

	scenario := NewScenarioBuilder("test_scenario", ScenarioTypeHappyPath).
		AddStep(ScenarioStep{
			Name: "step1",
			Action: func(ctx context.Context) error {
				return nil
			},
		}).
		Build()

	_ = executor.RegisterScenario(scenario)
	_, _ = executor.ExecuteScenario(context.Background(), "test_scenario")

	results := executor.GetResults()
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestScenarioExecutorGetSummary(t *testing.T) {
	executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

	scenario := NewScenarioBuilder("test_scenario", ScenarioTypeHappyPath).
		AddStep(ScenarioStep{
			Name: "step1",
			Action: func(ctx context.Context) error {
				return nil
			},
		}).
		Build()

	_ = executor.RegisterScenario(scenario)
	_, _ = executor.ExecuteScenario(context.Background(), "test_scenario")

	summary := executor.GetSummary()

	if summary["total_scenarios"] != 1 {
		t.Errorf("Expected 1 total scenario, got %v", summary["total_scenarios"])
	}

	if summary["passed_scenarios"] != 1 {
		t.Errorf("Expected 1 passed scenario, got %v", summary["passed_scenarios"])
	}

	if summary["success_rate"] != 100.0 {
		t.Errorf("Expected 100%% success rate, got %v", summary["success_rate"])
	}
}

func TestScenarioBuilderFluent(t *testing.T) {
	scenario := NewScenarioBuilder("test", ScenarioTypeHappyPath).
		WithDescription("Test scenario").
		WithSetup(func(ctx context.Context) error { return nil }).
		WithTeardown(func(ctx context.Context) error { return nil }).
		AddStep(ScenarioStep{
			Name: "step1",
			Action: func(ctx context.Context) error {
				return nil
			},
		}).
		Build()

	if scenario.Name != "test" {
		t.Errorf("Expected name 'test', got %s", scenario.Name)
	}

	if scenario.Description != "Test scenario" {
		t.Errorf("Expected description 'Test scenario', got %s", scenario.Description)
	}

	if scenario.Setup == nil {
		t.Error("Setup function not set")
	}

	if scenario.Teardown == nil {
		t.Error("Teardown function not set")
	}

	if len(scenario.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(scenario.Steps))
	}
}

func TestScenarioExecutorStepTimeout(t *testing.T) {
	executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

	scenario := NewScenarioBuilder("test_scenario", ScenarioTypeHappyPath).
		AddStep(ScenarioStep{
			Name:    "slow_step",
			Timeout: 100 * time.Millisecond,
			Action: func(ctx context.Context) error {
				select {
				case <-time.After(500 * time.Millisecond):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		}).
		Build()

	_ = executor.RegisterScenario(scenario)
	result, _ := executor.ExecuteScenario(context.Background(), "test_scenario")

	if result.Status != "FAILED" {
		t.Errorf("Expected status FAILED due to timeout, got %s", result.Status)
	}
}

func TestScenarioExecutorMultipleSteps(t *testing.T) {
	executor := NewScenarioExecutor(NewTestLogger(), NewMetricsCollector("test"))

	stepOrder := []string{}
	scenario := NewScenarioBuilder("test_scenario", ScenarioTypeHappyPath).
		AddStep(ScenarioStep{
			Name: "step1",
			Action: func(ctx context.Context) error {
				stepOrder = append(stepOrder, "step1")
				return nil
			},
		}).
		AddStep(ScenarioStep{
			Name: "step2",
			Action: func(ctx context.Context) error {
				stepOrder = append(stepOrder, "step2")
				return nil
			},
		}).
		AddStep(ScenarioStep{
			Name: "step3",
			Action: func(ctx context.Context) error {
				stepOrder = append(stepOrder, "step3")
				return nil
			},
		}).
		Build()

	_ = executor.RegisterScenario(scenario)
	result, _ := executor.ExecuteScenario(context.Background(), "test_scenario")

	if result.StepsRun != 3 {
		t.Errorf("Expected 3 steps run, got %d", result.StepsRun)
	}

	if len(stepOrder) != 3 || stepOrder[0] != "step1" || stepOrder[1] != "step2" || stepOrder[2] != "step3" {
		t.Errorf("Steps executed in wrong order: %v", stepOrder)
	}
}
