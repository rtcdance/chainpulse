package e2e

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ScenarioType represents the type of scenario being tested
type ScenarioType string

const (
	ScenarioTypeHappyPath   ScenarioType = "happy_path"
	ScenarioTypeErrorPath   ScenarioType = "error_path"
	ScenarioTypePerformance ScenarioType = "performance"
	ScenarioTypeMultiChain  ScenarioType = "multi_chain"
	ScenarioTypeStressTest  ScenarioType = "stress_test"
)

// ScenarioStep represents a single step in a scenario
type ScenarioStep struct {
	Name        string
	Description string
	Action      func(ctx context.Context) error
	Validate    func(ctx context.Context) error
	Timeout     time.Duration
}

// Scenario represents a complete test scenario
type Scenario struct {
	Name        string
	Type        ScenarioType
	Description string
	Steps       []ScenarioStep
	Setup       func(ctx context.Context) error
	Teardown    func(ctx context.Context) error
	Metrics     *MetricsCollector
}

// ScenarioResult represents the result of a scenario execution
type ScenarioResult struct {
	ScenarioName string
	Type         ScenarioType
	Status       string
	Duration     time.Duration
	StepsRun     int
	StepsFailed  int
	Error        error
	Metrics      map[string]interface{}
	Timestamp    time.Time
}

// ScenarioExecutor executes scenarios and collects results
type ScenarioExecutor struct {
	mu        sync.RWMutex
	scenarios map[string]*Scenario
	results   []ScenarioResult
	logger    Logger
	metrics   *MetricsCollector
}

// NewScenarioExecutor creates a new scenario executor
func NewScenarioExecutor(logger Logger, metrics *MetricsCollector) *ScenarioExecutor {
	return &ScenarioExecutor{
		scenarios: make(map[string]*Scenario),
		results:   make([]ScenarioResult, 0),
		logger:    logger,
		metrics:   metrics,
	}
}

// RegisterScenario registers a scenario for execution
func (se *ScenarioExecutor) RegisterScenario(scenario *Scenario) error {
	se.mu.Lock()
	defer se.mu.Unlock()

	if _, exists := se.scenarios[scenario.Name]; exists {
		return fmt.Errorf("scenario %s already registered", scenario.Name)
	}

	scenario.Metrics = NewMetricsCollector(scenario.Name)
	se.scenarios[scenario.Name] = scenario
	return nil
}

// ExecuteScenario executes a single scenario
func (se *ScenarioExecutor) ExecuteScenario(ctx context.Context, scenarioName string) (*ScenarioResult, error) {
	se.mu.RLock()
	scenario, exists := se.scenarios[scenarioName]
	se.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("scenario %s not found", scenarioName)
	}

	result := &ScenarioResult{
		ScenarioName: scenarioName,
		Type:         scenario.Type,
		Status:       "RUNNING",
		Timestamp:    time.Now(),
	}

	startTime := time.Now()

	// Run setup
	if scenario.Setup != nil {
		if err := scenario.Setup(ctx); err != nil {
			result.Status = "FAILED"
			result.Error = fmt.Errorf("setup failed: %w", err)
			result.Duration = time.Since(startTime)
			se.recordResult(result)
			return result, result.Error
		}
	}

	// Run steps
	for i, step := range scenario.Steps {
		stepCtx := ctx
		if step.Timeout > 0 {
			var cancel context.CancelFunc
			stepCtx, cancel = context.WithTimeout(ctx, step.Timeout)
			defer cancel()
		}

		// Execute action
		if err := step.Action(stepCtx); err != nil {
			result.Status = "FAILED"
			result.Error = fmt.Errorf("step %d (%s) action failed: %w", i+1, step.Name, err)
			result.StepsFailed++
			scenario.Metrics.RecordError()
			break
		}

		// Validate
		if step.Validate != nil {
			if err := step.Validate(stepCtx); err != nil {
				result.Status = "FAILED"
				result.Error = fmt.Errorf("step %d (%s) validation failed: %w", i+1, step.Name, err)
				result.StepsFailed++
				scenario.Metrics.RecordError()
				break
			}
		}

		result.StepsRun++
		scenario.Metrics.RecordSuccess()
	}

	// Run teardown
	if scenario.Teardown != nil {
		if err := scenario.Teardown(ctx); err != nil {
			if result.Status != "FAILED" {
				result.Status = "FAILED"
				result.Error = fmt.Errorf("teardown failed: %w", err)
			}
		}
	}

	// Finalize metrics
	scenario.Metrics.SetTestStatus(result.Status)
	scenario.Metrics.Finalize()

	result.Duration = time.Since(startTime)
	if result.Status != "FAILED" {
		result.Status = "PASSED"
	}

	result.Metrics = scenario.Metrics.GetStats()
	se.recordResult(result)

	return result, nil
}

// ExecuteAllScenarios executes all registered scenarios
func (se *ScenarioExecutor) ExecuteAllScenarios(ctx context.Context) ([]ScenarioResult, error) {
	se.mu.RLock()
	scenarioNames := make([]string, 0, len(se.scenarios))
	for name := range se.scenarios {
		scenarioNames = append(scenarioNames, name)
	}
	se.mu.RUnlock()

	results := make([]ScenarioResult, 0, len(scenarioNames))
	for _, name := range scenarioNames {
		result, err := se.ExecuteScenario(ctx, name)
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}

	return results, nil
}

// ExecuteScenariosByType executes all scenarios of a specific type
func (se *ScenarioExecutor) ExecuteScenariosByType(ctx context.Context, scenarioType ScenarioType) ([]ScenarioResult, error) {
	se.mu.RLock()
	scenarioNames := make([]string, 0)
	for name, scenario := range se.scenarios {
		if scenario.Type == scenarioType {
			scenarioNames = append(scenarioNames, name)
		}
	}
	se.mu.RUnlock()

	results := make([]ScenarioResult, 0, len(scenarioNames))
	for _, name := range scenarioNames {
		result, err := se.ExecuteScenario(ctx, name)
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}

	return results, nil
}

// GetResults returns all scenario results
func (se *ScenarioExecutor) GetResults() []ScenarioResult {
	se.mu.RLock()
	defer se.mu.RUnlock()

	results := make([]ScenarioResult, len(se.results))
	copy(results, se.results)
	return results
}

// GetResultsByType returns results for a specific scenario type
func (se *ScenarioExecutor) GetResultsByType(scenarioType ScenarioType) []ScenarioResult {
	se.mu.RLock()
	defer se.mu.RUnlock()

	results := make([]ScenarioResult, 0)
	for _, result := range se.results {
		if result.Type == scenarioType {
			results = append(results, result)
		}
	}
	return results
}

// GetSummary returns a summary of all scenario results
func (se *ScenarioExecutor) GetSummary() map[string]interface{} {
	se.mu.RLock()
	defer se.mu.RUnlock()

	totalScenarios := len(se.results)
	passedScenarios := 0
	failedScenarios := 0
	totalDuration := time.Duration(0)
	totalSteps := 0
	failedSteps := 0

	for _, result := range se.results {
		if result.Status == "PASSED" {
			passedScenarios++
		} else {
			failedScenarios++
		}
		totalDuration += result.Duration
		totalSteps += result.StepsRun
		failedSteps += result.StepsFailed
	}

	successRate := 0.0
	if totalScenarios > 0 {
		successRate = float64(passedScenarios) / float64(totalScenarios) * 100
	}

	return map[string]interface{}{
		"total_scenarios":  totalScenarios,
		"passed_scenarios": passedScenarios,
		"failed_scenarios": failedScenarios,
		"success_rate":     successRate,
		"total_duration":   totalDuration.String(),
		"total_steps":      totalSteps,
		"failed_steps":     failedSteps,
		"average_duration": time.Duration(int64(totalDuration) / int64(totalScenarios)).String(),
	}
}

// recordResult records a scenario result
func (se *ScenarioExecutor) recordResult(result *ScenarioResult) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.results = append(se.results, *result)
}

// ScenarioBuilder helps build scenarios fluently
type ScenarioBuilder struct {
	scenario *Scenario
}

// NewScenarioBuilder creates a new scenario builder
func NewScenarioBuilder(name string, scenarioType ScenarioType) *ScenarioBuilder {
	return &ScenarioBuilder{
		scenario: &Scenario{
			Name:  name,
			Type:  scenarioType,
			Steps: make([]ScenarioStep, 0),
		},
	}
}

// WithDescription sets the scenario description
func (sb *ScenarioBuilder) WithDescription(description string) *ScenarioBuilder {
	sb.scenario.Description = description
	return sb
}

// WithSetup sets the setup function
func (sb *ScenarioBuilder) WithSetup(setup func(ctx context.Context) error) *ScenarioBuilder {
	sb.scenario.Setup = setup
	return sb
}

// WithTeardown sets the teardown function
func (sb *ScenarioBuilder) WithTeardown(teardown func(ctx context.Context) error) *ScenarioBuilder {
	sb.scenario.Teardown = teardown
	return sb
}

// AddStep adds a step to the scenario
func (sb *ScenarioBuilder) AddStep(step ScenarioStep) *ScenarioBuilder {
	sb.scenario.Steps = append(sb.scenario.Steps, step)
	return sb
}

// Build returns the built scenario
func (sb *ScenarioBuilder) Build() *Scenario {
	return sb.scenario
}
