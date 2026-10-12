package qerrors

import (
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
)

type Type int

const (
	TypeTransient Type = iota
	TypePermanent
	TypeCritical
	TypeUnknown
)

func (et Type) String() string {
	switch et {
	case TypeTransient:
		return "transient"
	case TypePermanent:
		return "permanent"
	case TypeCritical:
		return "critical"
	default:
		return "unknown"
	}
}

type Classifier struct {
	transientPatterns []string
	permanentPatterns []string
	criticalPatterns  []string
}

func NewClassifier() *Classifier {
	return &Classifier{
		transientPatterns: []string{
			"connection refused", "timeout", "too many connections",
			"deadline exceeded", "context deadline", "no reachable servers",
			"server selection timeout", "EOF",
			// Socket/network level failures — all retryable.
			"connection reset", "connection closed", "broken pipe",
			"network unreachable", "host unreachable",
			// Transient resolver / availability failures.
			"temporary failure", "temporarily unavailable",
			// Connection-pool saturation is backpressure, not corruption.
			"pool exhausted",
		},
		permanentPatterns: []string{
			"not found", "invalid", "permission denied", "unauthorized",
			"forbidden", "already exists", "constraint violation",
			"duplicate key",
			// Schema/statement errors will never succeed on retry.
			"no such table", "no such column", "syntax error",
			// Auth failures are permanent until credentials change.
			"access denied", "authentication failed",
			// "unique constraint" alone (without "violation") is still a
			// constraint error.
			"unique constraint",
		},
		criticalPatterns: []string{
			"panic", "corruption", "data loss", "integrity violation",
			"fatal", "disk full", "out of memory",
			// Storage exhaustion and process-level crashes need operator
			// intervention, not retries.
			"disk quota", "segmentation fault", "unrecoverable",
		},
	}
}

func (ec *Classifier) Classify(err error) Type {
	if err == nil {
		return TypeUnknown
	}
	errStr := strings.ToLower(err.Error())
	if ec.matchesPattern(errStr, ec.criticalPatterns) {
		return TypeCritical
	}
	if ec.matchesPattern(errStr, ec.transientPatterns) {
		return TypeTransient
	}
	if ec.matchesPattern(errStr, ec.permanentPatterns) {
		return TypePermanent
	}
	return TypeUnknown
}

func (ec *Classifier) IsTransient(err error) bool { return ec.Classify(err) == TypeTransient }
func (ec *Classifier) IsPermanent(err error) bool { return ec.Classify(err) == TypePermanent }
func (ec *Classifier) IsCritical(err error) bool  { return ec.Classify(err) == TypeCritical }

func (ec *Classifier) ClassifyWithContext(err error, context string) Type {
	if err == nil {
		return TypeUnknown
	}
	combined := strings.ToLower(err.Error() + " " + context)
	if ec.matchesPattern(combined, ec.criticalPatterns) {
		return TypeCritical
	}
	if ec.matchesPattern(combined, ec.transientPatterns) {
		return TypeTransient
	}
	if ec.matchesPattern(combined, ec.permanentPatterns) {
		return TypePermanent
	}
	return TypeUnknown
}

func (ec *Classifier) IsMongoErrorTransient(err error) bool {
	if err == nil {
		return false
	}
	var mongoErr mongo.ServerError
	if !errorsAs(err, &mongoErr) {
		return ec.IsTransient(err)
	}
	if mongoErr.HasErrorCode(91) || mongoErr.HasErrorCode(134) || mongoErr.HasErrorCode(189) {
		return true
	}
	return false
}

func (ec *Classifier) matchesPattern(s string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func errorsAs(err error, target interface{}) bool {
	for {
		if err == nil {
			return false
		}
		if e, ok := err.(interface{ As(interface{}) bool }); ok {
			if e.As(target) {
				return true
			}
		}
		return false
	}
}
