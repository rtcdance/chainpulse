// Package topics provides EventBus topic constants.
//
// Centralized to prevent typos and enable IDE "Find All References".
// These constants were originally defined in pkg/core and are re-exported
// there for backward compatibility.
package topics

const (
	TopicBlockchainEvents = "blockchain-events"
	TopicReorgDetected    = "reorg-detected"
	TopicReorgRollback    = "reorg-rollback"
)