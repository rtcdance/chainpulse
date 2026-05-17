package grpc

import (
	"context"
	"strings"

	api "github.com/rtcdance/chainpulse/pkg/plugins/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryAuthInterceptor returns a grpc.UnaryServerInterceptor that validates
// JWT Bearer tokens or API Keys. If validator is nil, all requests are allowed.
func UnaryAuthInterceptor(validator *api.TokenValidator) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if validator == nil {
			return handler(ctx, req)
		}

		if err := validateGRPCAuth(ctx, validator); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

// StreamAuthInterceptor returns a grpc.StreamServerInterceptor that validates
// JWT Bearer tokens or API Keys. If validator is nil, all requests are allowed.
func StreamAuthInterceptor(validator *api.TokenValidator) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if validator == nil {
			return handler(srv, ss)
		}

		if err := validateGRPCAuth(ss.Context(), validator); err != nil {
			return err
		}

		return handler(srv, ss)
	}
}

// validateGRPCAuth checks Authorization or X-API-Key in gRPC metadata
func validateGRPCAuth(ctx context.Context, validator *api.TokenValidator) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing metadata")
	}

	// Check API Key first
	if apiKeys := md.Get("x-api-key"); len(apiKeys) > 0 {
		result := validator.ValidateAPIKey(ctx, apiKeys[0])
		if result.Valid {
			return nil
		}
		return status.Error(codes.Unauthenticated, "invalid API key")
	}

	// Check Bearer token
	if authHeaders := md.Get("authorization"); len(authHeaders) > 0 {
		token := strings.TrimPrefix(authHeaders[0], "Bearer ")
		if token == authHeaders[0] || token == "" {
			return status.Error(codes.Unauthenticated, "invalid authorization header format")
		}
		result := validator.ValidateJWT(token)
		if result.Valid {
			return nil
		}
		return status.Error(codes.Unauthenticated, "invalid token")
	}

	return status.Error(codes.Unauthenticated, "no credentials provided")
}
