package core

import "github.com/rtcdance/chainpulse/pkg/evm"

// Forwarding aliases for types moved to domain packages.
// These are temporary shims — update callers to import the domain packages directly.
// Deprecated: Import from pkg/evm directly.
type ConfirmationTracker = evm.ConfirmationTracker