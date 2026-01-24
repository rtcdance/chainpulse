# Unit Test Standards and Guidelines

## Overview

This document establishes standards for writing unit tests in ChainPulse. All unit tests should follow these patterns and conventions to ensure consistency, maintainability, and comprehensive coverage.

## Test File Organization

### Naming Convention
- Test files: `{source_file}_test.go`
- Property test files: `{source_file}_property_test.go`
- Example: `query_service.go` → `query_service_test.go` and `query_service_property_test.go`

### File Structure
```
pkg/services/query/
├── query_service.go              # Source code
├── query_service_test.go         # Unit tests
├── query_service_property_test.go # Property tests
└── query_service_integration_test.go # Integration tests (if needed)
```

## Test Function Naming

### Pattern: `Test{FunctionName}_{Scenario}`

```go
// Good
func TestQueryService_ParseQuery_ValidInput(t *testing.T) {}
func TestQueryService_ParseQuery_InvalidInput(t *testing.T) {}
func TestQueryService_ParseQuery_EmptyString(t *testing.T) {}

// Avoid
func TestParse(t *testing.T) {}           // Too vague
func TestParseQuery1(t *testing.T) {}     // Unclear scenario
func Test_parse_query(t *testing.T) {}    // Wrong casing
```

## Basic Unit Test Template

### Simple Test
```go
package query

import (
	"testing"
)

func TestQueryService_ParseQuery_ValidInput(t *testing.T) {
	// Arrange
	service := NewQueryService()
	input := "SELECT * FROM events WHERE id = 1"

	// Act
	result, err := service.ParseQuery(input)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected result, got nil")
	}
	if result.Table != "events" {
		t.Errorf("expected table 'events', got '%s'", result.Table)
	}
}
```

### Test with Setup and Cleanup
```go
func TestQueryService_WithDatabase(t *testing.T) {
	// Setup
	db := setupTestDatabase(t)
	defer db.Close()

	service := NewQueryService(db)

	// Test
	result, err := service.QueryEvents(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify
	if len(result) == 0 {
		t.Error("expected events, got none")
	}
}
```

## Table-Driven Tests

### Pattern for Multiple Scenarios
```go
func TestQueryService_ParseQuery_VariousInputs(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTable string
		wantErr   bool
		wantMsg   string
	}{
		{
			name:      "simple select",
			input:     "SELECT * FROM events",
			wantTable: "events",
			wantErr:   false,
		},
		{
			name:      "select with where",
			input:     "SELECT * FROM events WHERE id = 1",
			wantTable: "events",
			wantErr:   false,
		},
		{
			name:      "invalid query",
			input:     "INVALID QUERY",
			wantTable: "",
			wantErr:   true,
			wantMsg:   "invalid syntax",
		},
		{
			name:      "empty input",
			input:     "",
			wantTable: "",
			wantErr:   true,
			wantMsg:   "empty query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewQueryService()
			result, err := service.ParseQuery(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.wantMsg != "" && !strings.Contains(err.Error(), tt.wantMsg) {
					t.Errorf("error should contain '%s', got '%s'", tt.wantMsg, err.Error())
				}
			} else {
				if result == nil {
					t.Error("expected result, got nil")
				}
				if result.Table != tt.wantTable {
					t.Errorf("expected table '%s', got '%s'", tt.wantTable, result.Table)
				}
			}
		})
	}
}
```

## Mocking Patterns

### Using testify/mock
```go
import "github.com/stretchr/testify/mock"

type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) Query(ctx context.Context, sql string) ([]Event, error) {
	args := m.Called(ctx, sql)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Event), args.Error(1)
}

func TestQueryService_WithMockedDatabase(t *testing.T) {
	// Setup mock
	mockDB := new(MockDatabase)
	mockDB.On("Query", mock.Anything, "SELECT * FROM events").
		Return([]Event{{ID: 1, Name: "test"}}, nil).
		Times(1)

	// Create service with mock
	service := NewQueryService(mockDB)

	// Execute
	events, err := service.GetEvents(context.Background())

	// Verify
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}
```

### Using Interface Mocks
```go
type MockCache struct {
	getFunc func(key string) (interface{}, bool)
	setFunc func(key string, value interface{})
}

func (m *MockCache) Get(key string) (interface{}, bool) {
	if m.getFunc != nil {
		return m.getFunc(key)
	}
	return nil, false
}

func (m *MockCache) Set(key string, value interface{}) {
	if m.setFunc != nil {
		m.setFunc(key, value)
	}
}

func TestQueryService_WithMockedCache(t *testing.T) {
	mockCache := &MockCache{
		getFunc: func(key string) (interface{}, bool) {
			if key == "events:1" {
				return []Event{{ID: 1}}, true
			}
			return nil, false
		},
	}

	service := NewQueryService(mockCache)
	events, _ := service.GetEvents(context.Background())

	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}
```

## Error Testing Patterns

### Testing Error Cases
```go
func TestQueryService_ParseQuery_ErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errType error
	}{
		{
			name:    "nil input",
			input:   "",
			wantErr: true,
			errType: ErrEmptyQuery,
		},
		{
			name:    "invalid syntax",
			input:   "INVALID",
			wantErr: true,
			errType: ErrInvalidSyntax,
		},
		{
			name:    "missing table",
			input:   "SELECT * FROM",
			wantErr: true,
			errType: ErrMissingTable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewQueryService()
			_, err := service.ParseQuery(tt.input)

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}

			if tt.wantErr && !errors.Is(err, tt.errType) {
				t.Errorf("expected error type %v, got %v", tt.errType, err)
			}
		})
	}
}
```

## Concurrency Testing

### Testing Concurrent Access
```go
func TestQueryService_ConcurrentAccess(t *testing.T) {
	service := NewQueryService()
	const numGoroutines = 10
	const queriesPerGoroutine = 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*queriesPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < queriesPerGoroutine; j++ {
				query := fmt.Sprintf("SELECT * FROM events WHERE id = %d", id*queriesPerGoroutine+j)
				_, err := service.ParseQuery(query)
				if err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("unexpected error: %v", err)
	}
}
```

### Testing Race Conditions
```bash
# Run tests with race detector
go test -race -v ./pkg/services/query/...
```

## Context and Timeout Testing

### Testing with Context
```go
func TestQueryService_WithContext(t *testing.T) {
	service := NewQueryService()

	// Test with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := service.LongRunningQuery(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("expected context deadline exceeded, got %v", err)
	}
}

func TestQueryService_ContextCancellation(t *testing.T) {
	service := NewQueryService()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := service.LongRunningQuery(ctx)
	if err != context.Canceled {
		t.Errorf("expected context canceled, got %v", err)
	}
}
```

## Assertion Helpers

### Creating Custom Assertions
```go
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("string '%s' does not contain '%s'", s, substr)
	}
}

// Usage
func TestQueryService_ParseQuery(t *testing.T) {
	service := NewQueryService()
	result, err := service.ParseQuery("SELECT * FROM events")

	assertNoError(t, err)
	assertEqual(t, result.Table, "events")
}
```

## Coverage Guidelines

### Minimum Coverage Targets
- **Core packages** (`pkg/core/`): 85%+
- **Service packages** (`pkg/services/`): 80%+
- **Plugin packages** (`pkg/plugins/`): 75%+
- **Integration packages** (`pkg/integrations/`): 70%+

### Measuring Coverage
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./pkg/...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# Check coverage for specific package
go test -coverprofile=coverage.out ./pkg/services/query/...
go tool cover -func=coverage.out | grep query
```

### Coverage Checklist
- [ ] All public functions have at least one test
- [ ] All error paths are tested
- [ ] All edge cases are tested
- [ ] All branches are covered
- [ ] Coverage meets package target

## Best Practices

### 1. Test Isolation
- Each test should be independent
- No shared state between tests
- Use `t.Run()` for sub-tests
- Clean up resources with `defer`

```go
func TestQueryService_Isolation(t *testing.T) {
	t.Run("first test", func(t *testing.T) {
		service := NewQueryService()
		// Test 1
	})

	t.Run("second test", func(t *testing.T) {
		service := NewQueryService()
		// Test 2 - independent from Test 1
	})
}
```

### 2. Descriptive Names
- Test names should describe what they test
- Include the scenario being tested
- Make it clear what the expected behavior is

```go
// Good
func TestQueryService_ParseQuery_WithValidInput_ReturnsCorrectTable(t *testing.T) {}
func TestQueryService_ParseQuery_WithEmptyInput_ReturnsError(t *testing.T) {}

// Avoid
func TestParse(t *testing.T) {}
func TestParseQuery1(t *testing.T) {}
```

### 3. Arrange-Act-Assert Pattern
- Clearly separate setup, execution, and verification
- Use comments to mark each section
- Keep each section focused

```go
func TestQueryService_ParseQuery(t *testing.T) {
	// Arrange
	service := NewQueryService()
	input := "SELECT * FROM events"

	// Act
	result, err := service.ParseQuery(input)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Table != "events" {
		t.Errorf("expected table 'events', got '%s'", result.Table)
	}
}
```

### 4. Avoid Sleep in Tests
- Use channels or WaitGroup for synchronization
- Use context timeouts instead of sleep
- Use test fixtures for setup

```go
// Bad
func TestQueryService_Concurrent(t *testing.T) {
	service := NewQueryService()
	go service.Process()
	time.Sleep(100 * time.Millisecond) // Flaky!
	// Verify
}

// Good
func TestQueryService_Concurrent(t *testing.T) {
	service := NewQueryService()
	done := make(chan bool)
	go func() {
		service.Process()
		done <- true
	}()
	<-done // Wait for completion
	// Verify
}
```

### 5. Test Error Messages
- Verify error messages are helpful
- Test error types with `errors.Is()`
- Test error wrapping with `errors.As()`

```go
func TestQueryService_ErrorMessages(t *testing.T) {
	service := NewQueryService()
	_, err := service.ParseQuery("")

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, ErrEmptyQuery) {
		t.Errorf("expected ErrEmptyQuery, got %v", err)
	}

	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error message should contain 'empty', got '%s'", err.Error())
	}
}
```

### 6. Use Helper Functions
- Mark helpers with `t.Helper()`
- Extract common setup code
- Make tests more readable

```go
func setupTestService(t *testing.T) *QueryService {
	t.Helper()
	service := NewQueryService()
	if service == nil {
		t.Fatal("failed to create service")
	}
	return service
}

func TestQueryService_ParseQuery(t *testing.T) {
	service := setupTestService(t)
	result, err := service.ParseQuery("SELECT * FROM events")
	// ...
}
```

## Running Tests

### Run All Tests
```bash
go test -v ./pkg/...
```

### Run Specific Package
```bash
go test -v ./pkg/services/query/...
```

### Run Specific Test
```bash
go test -v -run TestQueryService_ParseQuery ./pkg/services/query/...
```

### Run with Coverage
```bash
go test -v -coverprofile=coverage.out ./pkg/...
```

### Run with Race Detector
```bash
go test -race -v ./pkg/...
```

### Run with Timeout
```bash
go test -v -timeout 30s ./pkg/...
```

## Debugging Failing Tests

### Verbose Output
```bash
go test -v -run TestQueryService_ParseQuery ./pkg/services/query/...
```

### With Logging
```go
func TestQueryService_WithLogging(t *testing.T) {
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	service := NewQueryService()
	service.SetLogger(logger)

	logger.Println("Starting test")
	result, err := service.ParseQuery("SELECT * FROM events")
	logger.Printf("Result: %v, Error: %v", result, err)
}
```

### With Debugger
```bash
# Run with debugger
dlv test ./pkg/services/query/ -- -test.run TestQueryService_ParseQuery
```

## Common Mistakes to Avoid

1. **Testing Implementation Details**: Test behavior, not implementation
2. **Flaky Tests**: Avoid sleep, use proper synchronization
3. **Shared State**: Each test should be independent
4. **Poor Error Messages**: Make it clear what failed and why
5. **Incomplete Coverage**: Test error paths and edge cases
6. **Slow Tests**: Keep unit tests fast (< 1 second each)
7. **Hard to Read**: Use clear names and structure
8. **Over-Mocking**: Mock only external dependencies

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Table-Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [testify/mock Documentation](https://github.com/stretchr/testify)

