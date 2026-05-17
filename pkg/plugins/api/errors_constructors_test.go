package api

import (
	"strings"
	"testing"
)

func TestErrInvalidRequest(t *testing.T) {
	t.Parallel()
	err := ErrInvalidRequest("bad input")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bad input") {
		t.Errorf("Error() = %q", err.Error())
	}
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Code != "INVALID_REQUEST" || apiErr.Status != 400 {
		t.Errorf("unexpected error type: %T %+v", err, apiErr)
	}
}

func TestErrUnauthorized(t *testing.T) {
	t.Parallel()
	err := ErrUnauthorized("invalid token")
	apiErr := err.(*APIError)
	if apiErr.Code != "UNAUTHORIZED" || apiErr.Status != 401 {
		t.Errorf("unexpected: %+v", apiErr)
	}
}

func TestErrForbidden(t *testing.T) {
	t.Parallel()
	err := ErrForbidden("no access")
	apiErr := err.(*APIError)
	if apiErr.Code != "FORBIDDEN" || apiErr.Status != 403 {
		t.Errorf("unexpected: %+v", apiErr)
	}
}

func TestErrNotFound(t *testing.T) {
	t.Parallel()
	err := ErrNotFound("user")
	if !strings.Contains(err.Error(), "user not found") {
		t.Errorf("Error() = %q", err.Error())
	}
	apiErr := err.(*APIError)
	if apiErr.Code != "NOT_FOUND" || apiErr.Status != 404 {
		t.Errorf("unexpected: %+v", apiErr)
	}
}

func TestErrInternalServer(t *testing.T) {
	t.Parallel()
	err := ErrInternalServer("oops")
	apiErr := err.(*APIError)
	if apiErr.Code != "INTERNAL_SERVER_ERROR" || apiErr.Status != 500 {
		t.Errorf("unexpected: %+v", apiErr)
	}
}

func TestErrServiceUnavailable(t *testing.T) {
	t.Parallel()
	err := ErrServiceUnavailable("down")
	apiErr := err.(*APIError)
	if apiErr.Code != "SERVICE_UNAVAILABLE" || apiErr.Status != 503 {
		t.Errorf("unexpected: %+v", apiErr)
	}
}

func TestErrRateLimited(t *testing.T) {
	t.Parallel()
	err := ErrRateLimited()
	apiErr := err.(*APIError)
	if apiErr.Code != "RATE_LIMIT_EXCEEDED" || apiErr.Status != 429 {
		t.Errorf("unexpected: %+v", apiErr)
	}
}

func TestErrInvalidParameter(t *testing.T) {
	t.Parallel()
	err := ErrInvalidParameter("age", "must be positive")
	apiErr := err.(*APIError)
	if apiErr.Code != "INVALID_PARAMETER" {
		t.Errorf("Code = %q", apiErr.Code)
	}
	if !strings.Contains(apiErr.Message, "age") || !strings.Contains(apiErr.Message, "must be positive") {
		t.Errorf("Message = %q", apiErr.Message)
	}
}

func TestErrMissingParameter(t *testing.T) {
	t.Parallel()
	err := ErrMissingParameter("name")
	apiErr := err.(*APIError)
	if apiErr.Code != "MISSING_PARAMETER" || apiErr.Status != 400 {
		t.Errorf("unexpected: %+v", apiErr)
	}
}

func TestErrValidationFailed(t *testing.T) {
	t.Parallel()
	err := ErrValidationFailed("email", "invalid format")
	apiErr := err.(*APIError)
	if apiErr.Code != "VALIDATION_FAILED" {
		t.Errorf("Code = %q", apiErr.Code)
	}
	if !strings.Contains(apiErr.Message, "email") {
		t.Errorf("Message = %q", apiErr.Message)
	}
}
