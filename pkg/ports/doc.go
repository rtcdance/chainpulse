// Package ports defines the primary interfaces (ports) for the ChainPulse
// hexagonal architecture.
//
// This package is the SINGLE authoritative source for all port interfaces.
// All external actors (plugins, services, infrastructure) depend on these
// interfaces, never on concrete implementations.
//
// Type aliases in pkg/core/ are maintained for backward compatibility only.
// New code MUST import this package directly.
//
// Migration note: pkg/core provides compatibility aliases for these definitions.
// Those aliases are deprecated; this package owns the interfaces.
package ports
