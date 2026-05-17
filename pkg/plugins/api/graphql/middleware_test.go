package graphql

import (
	"context"
	"testing"
)

// mockTokenValidator implements TokenValidator for testing
type mockTokenValidator struct {
	tokenResult  ValidationResult
	apiKeyResult ValidationResult
}

func (m *mockTokenValidator) ValidateToken(ctx context.Context, authHeader string) ValidationResult {
	return m.tokenResult
}

func (m *mockTokenValidator) ValidateAPIKey(ctx context.Context, apiKey string) ValidationResult {
	return m.apiKeyResult
}

func TestAuthenticate_MissingToken(t *testing.T) {
	t.Parallel()
	am := NewAuthMiddleware(nil, nil)
	_, err := am.Authenticate(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestAuthenticate_TokenTooShort(t *testing.T) {
	t.Parallel()
	am := NewAuthMiddleware(nil, nil)
	am.SetRequireAuth(true)
	_, err := am.Authenticate(context.Background(), "ab")
	if err == nil {
		t.Fatal("expected error for short token")
	}
}

func TestAuthenticate_NoValidatorRejection(t *testing.T) {
	t.Parallel()
	am := NewAuthMiddleware(nil, nil)
	am.SetRequireAuth(true)
	_, err := am.Authenticate(context.Background(), "some-valid-token")
	if err == nil {
		t.Fatal("expected error when no validator configured")
	}
}

func TestAuthenticate_ValidJWT(t *testing.T) {
	t.Parallel()
	am := NewAuthMiddleware(nil, nil)
	am.SetRequireAuth(true)
	am.SetTokenValidator(&mockTokenValidator{
		tokenResult: ValidationResult{
			Valid:       true,
			UserID:      "test-user",
			Roles:       []string{"admin"},
			Permissions: []string{"read:events", "manage:cache"},
		},
	})

	authCtx, err := am.Authenticate(context.Background(), "Bearer valid-jwt-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if authCtx.UserID != "test-user" {
		t.Fatalf("expected UserID 'test-user', got '%s'", authCtx.UserID)
	}
	if len(authCtx.Scopes) != 2 {
		t.Fatalf("expected 2 scopes, got %d", len(authCtx.Scopes))
	}
}

func TestAuthenticate_ValidAPIKey(t *testing.T) {
	t.Parallel()
	am := NewAuthMiddleware(nil, nil)
	am.SetRequireAuth(true)
	am.SetTokenValidator(&mockTokenValidator{
		tokenResult: ValidationResult{Valid: false, Error: "invalid JWT"},
		apiKeyResult: ValidationResult{
			Valid:       true,
			UserID:      "api-user",
			Roles:       []string{"user"},
			Permissions: []string{"read:events"},
		},
	})

	authCtx, err := am.Authenticate(context.Background(), "valid-api-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if authCtx.UserID != "api-user" {
		t.Fatalf("expected UserID 'api-user', got '%s'", authCtx.UserID)
	}
}

func TestAuthenticate_InvalidCredentials(t *testing.T) {
	t.Parallel()
	am := NewAuthMiddleware(nil, nil)
	am.SetRequireAuth(true)
	am.SetTokenValidator(&mockTokenValidator{
		tokenResult:  ValidationResult{Valid: false, Error: "bad token"},
		apiKeyResult: ValidationResult{Valid: false, Error: "bad key"},
	})

	_, err := am.Authenticate(context.Background(), "invalid-credentials")
	if err == nil {
		t.Fatal("expected error for invalid credentials")
	}
}

func TestAuthenticate_AuthDisabled(t *testing.T) {
	t.Parallel()
	am := NewAuthMiddleware(nil, nil)
	am.SetRequireAuth(false)

	authCtx, err := am.Authenticate(context.Background(), "any-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if authCtx.UserID != "anonymous" {
		t.Fatalf("expected UserID 'anonymous' when auth disabled, got '%s'", authCtx.UserID)
	}
}
