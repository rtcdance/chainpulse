package database

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestErrorMongoClientNotInitialized tests MongoDB client not initialized error
func TestErrorMongoClientNotInitialized(t *testing.T) {
	assert.NotNil(t, ErrMongoClientNotInitialized)
	assert.Equal(t, "MongoDB client not initialized", ErrMongoClientNotInitialized.Error())
}

// TestErrorPostgresDBNotInitialized tests PostgreSQL DB not initialized error
func TestErrorPostgresDBNotInitialized(t *testing.T) {
	assert.NotNil(t, ErrPostgresDBNotInitialized)
	assert.Equal(t, "PostgreSQL database not initialized", ErrPostgresDBNotInitialized.Error())
}

// TestErrorDatabaseManagerNotInitialized tests database manager not initialized error
func TestErrorDatabaseManagerNotInitialized(t *testing.T) {
	assert.NotNil(t, ErrDatabaseManagerNotInitialized)
	assert.Equal(t, "database manager not initialized", ErrDatabaseManagerNotInitialized.Error())
}

// TestErrorDatabaseManagerAlreadyClosed tests database manager already closed error
func TestErrorDatabaseManagerAlreadyClosed(t *testing.T) {
	assert.NotNil(t, ErrDatabaseManagerAlreadyClosed)
	assert.Equal(t, "database manager already closed", ErrDatabaseManagerAlreadyClosed.Error())
}

// TestErrorsAreErrors tests that all errors implement error interface
func TestErrorsAreErrors(t *testing.T) {
	errors := []error{
		ErrMongoClientNotInitialized,
		ErrPostgresDBNotInitialized,
		ErrDatabaseManagerNotInitialized,
		ErrDatabaseManagerAlreadyClosed,
	}

	for _, err := range errors {
		assert.NotNil(t, err)
		assert.NotEmpty(t, err.Error())
	}
}

// TestErrorsAreUnique tests that all errors are unique
func TestErrorsAreUnique(t *testing.T) {
	errors := []error{
		ErrMongoClientNotInitialized,
		ErrPostgresDBNotInitialized,
		ErrDatabaseManagerNotInitialized,
		ErrDatabaseManagerAlreadyClosed,
	}

	errorStrings := make(map[string]bool)
	for _, err := range errors {
		errStr := err.Error()
		assert.False(t, errorStrings[errStr], "duplicate error message: %s", errStr)
		errorStrings[errStr] = true
	}
}

// TestErrorMongoClientNotInitializedComparison tests error comparison
func TestErrorMongoClientNotInitializedComparison(t *testing.T) {
	err := ErrMongoClientNotInitialized
	assert.Equal(t, ErrMongoClientNotInitialized, err)
	assert.NotEqual(t, ErrPostgresDBNotInitialized, err)
}

// TestErrorPostgresDBNotInitializedComparison tests error comparison
func TestErrorPostgresDBNotInitializedComparison(t *testing.T) {
	err := ErrPostgresDBNotInitialized
	assert.Equal(t, ErrPostgresDBNotInitialized, err)
	assert.NotEqual(t, ErrMongoClientNotInitialized, err)
}

// TestErrorDatabaseManagerNotInitializedComparison tests error comparison
func TestErrorDatabaseManagerNotInitializedComparison(t *testing.T) {
	err := ErrDatabaseManagerNotInitialized
	assert.Equal(t, ErrDatabaseManagerNotInitialized, err)
	assert.NotEqual(t, ErrDatabaseManagerAlreadyClosed, err)
}

// TestErrorDatabaseManagerAlreadyClosedComparison tests error comparison
func TestErrorDatabaseManagerAlreadyClosedComparison(t *testing.T) {
	err := ErrDatabaseManagerAlreadyClosed
	assert.Equal(t, ErrDatabaseManagerAlreadyClosed, err)
	assert.NotEqual(t, ErrDatabaseManagerNotInitialized, err)
}

// TestErrorWrapping tests error wrapping
func TestErrorWrapping(t *testing.T) {
	wrappedErr := fmt.Errorf("failed to initialize: %w", ErrMongoClientNotInitialized)

	assert.NotNil(t, wrappedErr)
	assert.Contains(t, wrappedErr.Error(), "MongoDB client not initialized")
}

// TestErrorMessages tests error messages are descriptive
func TestErrorMessages(t *testing.T) {
	testCases := []struct {
		err     error
		message string
	}{
		{ErrMongoClientNotInitialized, "MongoDB client not initialized"},
		{ErrPostgresDBNotInitialized, "PostgreSQL database not initialized"},
		{ErrDatabaseManagerNotInitialized, "database manager not initialized"},
		{ErrDatabaseManagerAlreadyClosed, "database manager already closed"},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.message, tc.err.Error())
	}
}

// TestErrorsNotNil tests that all errors are not nil
func TestErrorsNotNil(t *testing.T) {
	assert.NotNil(t, ErrMongoClientNotInitialized)
	assert.NotNil(t, ErrPostgresDBNotInitialized)
	assert.NotNil(t, ErrDatabaseManagerNotInitialized)
	assert.NotNil(t, ErrDatabaseManagerAlreadyClosed)
}

// TestErrorsCanBeUsedInConditions tests errors can be used in conditions
func TestErrorsCanBeUsedInConditions(t *testing.T) {
	var err error

	err = ErrMongoClientNotInitialized
	assert.NotNil(t, err)

	err = ErrPostgresDBNotInitialized
	assert.NotNil(t, err)

	err = ErrDatabaseManagerNotInitialized
	assert.NotNil(t, err)

	err = ErrDatabaseManagerAlreadyClosed
	assert.NotNil(t, err)
}

// TestErrorsInSlice tests errors in slice
func TestErrorsInSlice(t *testing.T) {
	errors := []error{
		ErrMongoClientNotInitialized,
		ErrPostgresDBNotInitialized,
		ErrDatabaseManagerNotInitialized,
		ErrDatabaseManagerAlreadyClosed,
	}

	assert.Equal(t, 4, len(errors))
	for _, err := range errors {
		assert.NotNil(t, err)
	}
}

// TestErrorsInMap tests errors in map
func TestErrorsInMap(t *testing.T) {
	errorMap := map[string]error{
		"mongo":    ErrMongoClientNotInitialized,
		"postgres": ErrPostgresDBNotInitialized,
		"manager":  ErrDatabaseManagerNotInitialized,
		"closed":   ErrDatabaseManagerAlreadyClosed,
	}

	assert.Equal(t, 4, len(errorMap))
	assert.NotNil(t, errorMap["mongo"])
	assert.NotNil(t, errorMap["postgres"])
	assert.NotNil(t, errorMap["manager"])
	assert.NotNil(t, errorMap["closed"])
}

// TestErrorStringRepresentation tests error string representation
func TestErrorStringRepresentation(t *testing.T) {
	err := ErrMongoClientNotInitialized
	errStr := err.Error()

	assert.NotEmpty(t, errStr)
	assert.IsType(t, "", errStr)
}

// TestErrorTypeAssertion tests error type assertion
func TestErrorTypeAssertion(t *testing.T) {
	err := ErrMongoClientNotInitialized

	assert.NotNil(t, err)
}

// TestErrorComparison tests error comparison with nil
func TestErrorComparison(t *testing.T) {
	assert.NotEqual(t, nil, ErrMongoClientNotInitialized)
	assert.NotEqual(t, nil, ErrPostgresDBNotInitialized)
	assert.NotEqual(t, nil, ErrDatabaseManagerNotInitialized)
	assert.NotEqual(t, nil, ErrDatabaseManagerAlreadyClosed)
}

// TestErrorsCanBeReturned tests errors can be returned from functions
func TestErrorsCanBeReturned(t *testing.T) {
	testFunc := func() error {
		return ErrMongoClientNotInitialized
	}

	err := testFunc()
	assert.Equal(t, ErrMongoClientNotInitialized, err)
}

// TestErrorsCanBeChecked tests errors can be checked
func TestErrorsCanBeChecked(t *testing.T) {
	err := ErrMongoClientNotInitialized

	if err != nil {
		assert.Equal(t, ErrMongoClientNotInitialized, err)
	} else {
		t.Fail()
	}
}

// TestErrorsCanBeFormatted tests errors can be formatted
func TestErrorsCanBeFormatted(t *testing.T) {
	err := ErrMongoClientNotInitialized
	formatted := fmt.Sprintf("Error: %v", err)

	assert.Contains(t, formatted, "MongoDB client not initialized")
}

// TestErrorsCanBeLogged tests errors can be logged
func TestErrorsCanBeLogged(t *testing.T) {
	err := ErrMongoClientNotInitialized
	logMessage := fmt.Sprintf("Database error: %s", err.Error())

	assert.NotEmpty(t, logMessage)
	assert.Contains(t, logMessage, "MongoDB client not initialized")
}
