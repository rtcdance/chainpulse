package main

import (
	"log"
	"net/http"
	"sync"
	"time"
)

// middlewareChain demonstrates Go's classic HTTP middleware pattern.
// Each middleware wraps the next handler in the chain, forming a "Russian doll"
// model (also known as the onion pattern).
//
// This is the same pattern used by gin, echo, chi, and the standard library —
// learning it unlocks the entire Go web ecosystem.
type middlewareChain struct {
	middlewares []func(http.Handler) http.Handler
}

func newMiddlewareChain() *middlewareChain {
	return &middlewareChain{}
}

func (c *middlewareChain) Use(mw func(http.Handler) http.Handler) {
	c.middlewares = append(c.middlewares, mw)
}

func (c *middlewareChain) Then(handler http.Handler) http.Handler {
	// Apply middlewares in reverse order so the first registered
	// runs outermost (closest to the client).
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}
	return handler
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[middleware] %s %s — %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[middleware] panic recovered: %v", rec)
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string][]time.Time
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		visitors: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		rl.mu.Lock()
		now := time.Now()
		cutoff := now.Add(-rl.window)

		times := rl.visitors[ip]
		valid := times[:0]
		for _, t := range times {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		rl.visitors[ip] = valid

		if len(valid) >= rl.limit {
			rl.mu.Unlock()
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		rl.visitors[ip] = append(valid, now)
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
