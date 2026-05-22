package api

import "github.com/rtcdance/chainpulse/pkg/plugins/api/apierrors"

// Backward-compatible type aliases for types moved to api/apierrors.
type (
	APIError = apierrors.APIError
)

// Backward-compatible function re-exports.
var (
	ErrInvalidRequest     = apierrors.ErrInvalidRequest
	ErrUnauthorized       = apierrors.ErrUnauthorized
	ErrForbidden          = apierrors.ErrForbidden
	ErrNotFound           = apierrors.ErrNotFound
	ErrInternalServer     = apierrors.ErrInternalServer
	ErrServiceUnavailable = apierrors.ErrServiceUnavailable
	ErrRateLimited        = apierrors.ErrRateLimited
	ErrInvalidParameter   = apierrors.ErrInvalidParameter
	ErrMissingParameter   = apierrors.ErrMissingParameter
	ErrValidationFailed   = apierrors.ErrValidationFailed
	MapErrorToAPIError    = apierrors.MapErrorToAPIError
	IsAPIError            = apierrors.IsAPIError
)
