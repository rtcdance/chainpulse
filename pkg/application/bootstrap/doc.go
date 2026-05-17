// Package bootstrap provides shared wiring utilities used by all chainpulse
// microservices (gateway, api-service, event-processor, puller).
//
// The two primary concerns addressed here are:
//
//   - Security controls: unified auth (JWT + API keys + RBAC) and rate limiting
//     middleware construction via BuildSecurityControls.
//
//   - Graceful shutdown: OS signal handling and goroutine lifecycle coordination
//     via WaitForSignal, ShutdownWithTimeout, and ShutdownWithContext.
//
// All four microservices share the same wiring patterns; this package eliminates
// the ~180 lines of duplicated code that previously existed across their main.go files.
package bootstrap
