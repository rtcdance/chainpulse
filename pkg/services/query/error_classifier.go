package query

import (
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
)

// ErrorType represents the classification of an error
type ErrorType int

const (
	// ErrorTypeTransient represents temporary errors that may succeed on retry
	ErrorTypeTransient ErrorType = iota
	// ErrorTypePermanent represents errors that won't succeed on retry
	ErrorTypePermanent
	// ErrorTypeCritical represents system-level failures requiring immediate attention
	ErrorTypeCritical
	// ErrorTypeUnknown represents errors that cannot be classified
	ErrorTypeUnknown
)

// String returns the string representation of ErrorType
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeTransient:
		return "transient"
	case ErrorTypePermanent:
		return "permanent"
	case ErrorTypeCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// ErrorClassifier classifies errors into categories
type ErrorClassifier struct {
	// Transient error patterns
	transientPatterns []string
	// Permanent error patterns
	permanentPatterns []string
	// Critical error patterns
	criticalPatterns []string
}

// NewErrorClassifier creates a new error classifier
func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{
		transientPatterns: []string{
			"connection refused",
			"connection reset",
			"connection timeout",
			"i/o timeout",
			"temporary failure",
			"temporarily unavailable",
			"service unavailable",
			"too many connections",
			"connection pool exhausted",
			"deadline exceeded",
			"context deadline exceeded",
			"broken pipe",
			"connection closed",
			"network unreachable",
			"host unreachable",
		},
		permanentPatterns: []string{
			"invalid argument",
			"invalid syntax",
			"constraint violation",
			"unique constraint",
			"foreign key constraint",
			"not found",
			"no such table",
			"no such column",
			"syntax error",
			"permission denied",
			"access denied",
			"authentication failed",
			"invalid credentials",
			"duplicate key",
			"invalid database",
		},
		criticalPatterns: []string{
			"out of memory",
			"disk full",
			"disk quota exceeded",
			"fatal error",
			"panic",
			"segmentation fault",
			"corruption detected",
			"data corruption",
			"unrecoverable error",
		},
	}
}

// ClassifyError classifies an error into a category
func (ec *ErrorClassifier) ClassifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}

	errMsg := strings.ToLower(err.Error())

	// Check for MongoDB specific errors
	if mongoErr := ec.classifyMongoDBError(err); mongoErr != ErrorTypeUnknown {
		return mongoErr
	}

	// Check for PostgreSQL specific errors
	if pgErr := ec.classifyPostgreSQLError(err); pgErr != ErrorTypeUnknown {
		return pgErr
	}

	// Check for critical patterns
	for _, pattern := range ec.criticalPatterns {
		if strings.Contains(errMsg, pattern) {
			return ErrorTypeCritical
		}
	}

	// Check for permanent patterns
	for _, pattern := range ec.permanentPatterns {
		if strings.Contains(errMsg, pattern) {
			return ErrorTypePermanent
		}
	}

	// Check for transient patterns
	for _, pattern := range ec.transientPatterns {
		if strings.Contains(errMsg, pattern) {
			return ErrorTypeTransient
		}
	}

	// Default to unknown
	return ErrorTypeUnknown
}

// classifyMongoDBError classifies MongoDB specific errors
func (ec *ErrorClassifier) classifyMongoDBError(err error) ErrorType {
	// Check for MongoDB command error
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) {
		// Connection errors are transient
		if cmdErr.Code == 13 || cmdErr.Code == 10107 || cmdErr.Code == 10108 {
			return ErrorTypeTransient
		}
		// Duplicate key is permanent
		if cmdErr.Code == 11000 || cmdErr.Code == 11001 {
			return ErrorTypePermanent
		}
		// Authentication errors are permanent
		if cmdErr.Code == 18 {
			return ErrorTypePermanent
		}
		// Timeout is transient
		if cmdErr.Code == 50 {
			return ErrorTypeTransient
		}
	}

	// Check for network error - use generic error checking instead
	// MongoDB driver doesn't export NetworkError, so we check error message
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "network") || strings.Contains(errMsg, "timeout") {
		return ErrorTypeTransient
	}

	// Check for write error
	var writeErr mongo.WriteError
	if errors.As(err, &writeErr) {
		// Duplicate key is permanent
		if writeErr.Code == 11000 || writeErr.Code == 11001 {
			return ErrorTypePermanent
		}
		// Other write errors are usually permanent
		return ErrorTypePermanent
	}

	// Check for write exception
	var writeExc mongo.WriteException
	if errors.As(err, &writeExc) {
		// Check individual write errors
		for _, we := range writeExc.WriteErrors {
			if we.Code == 11000 || we.Code == 11001 {
				return ErrorTypePermanent
			}
		}
		return ErrorTypePermanent
	}

	return ErrorTypeUnknown
}

// classifyPostgreSQLError classifies PostgreSQL specific errors
func (ec *ErrorClassifier) classifyPostgreSQLError(err error) ErrorType {
	errMsg := strings.ToLower(err.Error())

	// Connection errors are transient
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "connection timeout") ||
		strings.Contains(errMsg, "i/o timeout") ||
		strings.Contains(errMsg, "broken pipe") {
		return ErrorTypeTransient
	}

	// Constraint violations are permanent
	if strings.Contains(errMsg, "unique constraint") ||
		strings.Contains(errMsg, "foreign key constraint") ||
		strings.Contains(errMsg, "check constraint") ||
		strings.Contains(errMsg, "not-null constraint") {
		return ErrorTypePermanent
	}

	// Syntax errors are permanent
	if strings.Contains(errMsg, "syntax error") ||
		strings.Contains(errMsg, "invalid syntax") {
		return ErrorTypePermanent
	}

	// Authentication errors are permanent
	if strings.Contains(errMsg, "authentication failed") ||
		strings.Contains(errMsg, "permission denied") ||
		strings.Contains(errMsg, "access denied") {
		return ErrorTypePermanent
	}

	// Table/column not found are permanent
	if strings.Contains(errMsg, "does not exist") ||
		strings.Contains(errMsg, "no such table") ||
		strings.Contains(errMsg, "no such column") {
		return ErrorTypePermanent
	}

	// Disk errors are critical
	if strings.Contains(errMsg, "disk full") ||
		strings.Contains(errMsg, "disk quota exceeded") {
		return ErrorTypeCritical
	}

	// Out of memory is critical
	if strings.Contains(errMsg, "out of memory") {
		return ErrorTypeCritical
	}

	return ErrorTypeUnknown
}

// IsTransient returns true if the error is transient
func (ec *ErrorClassifier) IsTransient(err error) bool {
	return ec.ClassifyError(err) == ErrorTypeTransient
}

// IsPermanent returns true if the error is permanent
func (ec *ErrorClassifier) IsPermanent(err error) bool {
	return ec.ClassifyError(err) == ErrorTypePermanent
}

// IsCritical returns true if the error is critical
func (ec *ErrorClassifier) IsCritical(err error) bool {
	return ec.ClassifyError(err) == ErrorTypeCritical
}

// ClassifyErrorWithContext classifies an error with additional context
func (ec *ErrorClassifier) ClassifyErrorWithContext(err error, operation string) (ErrorType, string) {
	errType := ec.ClassifyError(err)
	context := fmt.Sprintf("operation=%s error_type=%s error=%v", operation, errType.String(), err)
	return errType, context
}
