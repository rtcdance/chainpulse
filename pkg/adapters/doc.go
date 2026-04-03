// Package adapters contains implementations for external systems.
//
// Examples: RPC clients, database connectors, cache adapters, API transports.
//
// Constraints:
// - Implements domain/application contracts.
// - Must not contain core business rules.
// - Can vary by deployment mode (in-memory vs production) behind same interface.
package adapters
