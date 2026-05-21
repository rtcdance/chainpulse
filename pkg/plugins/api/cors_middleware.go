package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/rtcdance/chainpulse/pkg/core"
)

type CORSMiddleware struct {
	allowedOrigins []string
	allowedMethods []string
	allowedHeaders []string
	maxAge         int
	logger         core.Logger
}

func NewCORSMiddleware(logger core.Logger) *CORSMiddleware {
	return &CORSMiddleware{
		allowedOrigins: []string{"*"},
		allowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		allowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-API-Key",
			"X-Requested-With",
		},
		maxAge: 86400,
		logger: logger,
	}
}

func (m *CORSMiddleware) SetAllowedOrigins(origins []string) {
	m.allowedOrigins = origins
}

func (m *CORSMiddleware) SetAllowedMethods(methods []string) {
	m.allowedMethods = methods
}

func (m *CORSMiddleware) SetAllowedHeaders(headers []string) {
	m.allowedHeaders = headers
}

func (m *CORSMiddleware) SetMaxAge(seconds int) {
	m.maxAge = seconds
}

func (m *CORSMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		if !m.isOriginAllowed(origin) {
			if m.logger != nil {
				m.logger.Warn("CORS: origin not allowed", "origin", origin)
			}
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Vary", "Origin")

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(m.allowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(m.allowedHeaders, ", "))
			w.Header().Set("Access-Control-Max-Age", strconv.Itoa(m.maxAge))
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *CORSMiddleware) isOriginAllowed(origin string) bool {
	for _, allowed := range m.allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}
