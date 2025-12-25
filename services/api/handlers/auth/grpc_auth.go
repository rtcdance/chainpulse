package auth

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPCAuthUnaryInterceptor adds authentication to gRPC unary calls
func (am *AuthMiddleware) GRPCAuthUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Skip authentication for specific methods (like health checks or login)
	if am.isPublicMethod(info.FullMethod) {
		return handler(ctx, req)
	}

	// Extract token from context
	tokenString, err := am.extractTokenFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	// Validate the token
	claims, err := am.ValidateToken(tokenString)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	// Add claims to the context
	newCtx := context.WithValue(ctx, "user", claims)

	// Continue with the authenticated context
	return handler(newCtx, req)
}

// GRPCAuthStreamInterceptor adds authentication to gRPC stream calls
func (am *AuthMiddleware) GRPCAuthStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// Skip authentication for specific methods (like health checks)
	if am.isPublicMethod(info.FullMethod) {
		return handler(srv, ss)
	}

	// Extract token from context
	tokenString, err := am.extractTokenFromContext(ss.Context())
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	// Validate the token
	claims, err := am.ValidateToken(tokenString)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	// Create a new context with the claims
	newCtx := context.WithValue(ss.Context(), "user", claims)

	// Wrap the stream with the new context
	wrappedStream := &wrappedStream{
		ServerStream: ss,
		newCtx:       newCtx,
	}
	
	return handler(srv, wrappedStream)
}

// extractTokenFromContext extracts the JWT token from the gRPC context
func (am *AuthMiddleware) extractTokenFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("no metadata in context")
	}

	// Check for "authorization" key (lowercase)
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		// Check for "Authorization" key (capitalized)
		authHeaders = md.Get("Authorization")
		if len(authHeaders) == 0 {
			return "", fmt.Errorf("authorization header not found")
		}
	}

	authHeader := authHeaders[0]
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is empty")
	}

	// Extract token from "Bearer {token}" format
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		// If the prefix wasn't found, try "Token " prefix
		tokenString = strings.TrimPrefix(authHeader, "Token ")
		if tokenString == authHeader {
			// Neither prefix found, return error
			return "", fmt.Errorf("authorization header must be in the form 'Bearer {token}' or 'Token {token}'")
		}
	}

	return strings.TrimSpace(tokenString), nil
}

// isPublicMethod checks if the method is public (doesn't require authentication)
func (am *AuthMiddleware) isPublicMethod(fullMethod string) bool {
	// Add any public methods here that don't require authentication
	// For example, health check methods or login methods
	publicMethods := []string{
		// "/grpc.health.v1.Health/Check",
		// "/grpc.health.v1.Health/Watch",
	}
	
	for _, method := range publicMethods {
		if fullMethod == method {
			return true
		}
	}
	
	return false
}

// wrappedStream wraps the gRPC stream to use the new context
type wrappedStream struct {
	grpc.ServerStream
	newCtx context.Context
}

// Context returns the new context
func (w *wrappedStream) Context() context.Context {
	return w.newCtx
}

// GetGRPCAuthInterceptors returns both unary and stream interceptors
func (am *AuthMiddleware) GetGRPCAuthInterceptors() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return am.GRPCAuthUnaryInterceptor, am.GRPCAuthStreamInterceptor
}