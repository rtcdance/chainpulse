package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Claims represents the JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// AuthMiddleware handles JWT token authentication
type AuthMiddleware struct {
	JWTSecret string
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		JWTSecret: jwtSecret,
	}
}

// Middleware is the actual middleware function
func (am *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			// If the prefix wasn't found, try "Token " prefix
			tokenString = strings.TrimPrefix(authHeader, "Token ")
			if tokenString == authHeader {
				// Neither prefix found, return error
				http.Error(w, "Authorization header must be in the form 'Bearer {token}' or 'Token {token}'", http.StatusUnauthorized)
				return
			}
		}

		// Validate the token
		claims, err := am.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
			return
		}

		// Add claims to the request context
		ctx := context.WithValue(r.Context(), "user", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GenerateToken creates a new JWT token
func (am *AuthMiddleware) GenerateToken(userID, role string) (string, error) {
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "chainpulse",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(am.JWTSecret))
}

// ValidateToken validates a JWT token and returns the claims
func (am *AuthMiddleware) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(am.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// GetUserFromContext retrieves user claims from the request context
func GetUserFromContext(ctx context.Context) *Claims {
	if user, ok := ctx.Value("user").(*Claims); ok {
		return user
	}
	return nil
}

// RequireRole creates a middleware that checks if the user has the required role
func (am *AuthMiddleware) RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				http.Error(w, "User not authenticated", http.StatusUnauthorized)
				return
			}

			if user.Role != requiredRole && requiredRole != "admin" && user.Role != "admin" {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}