package core

import "github.com/rtcdance/chainpulse/pkg/ports"

// Deprecated: Type aliases for backward compatibility. New code should import
// pkg/ports directly. These aliases will be removed in a future major version.
type (
	IdempotencyInvalidator = ports.IdempotencyInvalidator
	CheckpointStore        = ports.CheckpointStore
)
