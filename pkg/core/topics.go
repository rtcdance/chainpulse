package core

// Topic constants for EventBus pub/sub.
// Centralized to prevent typos and enable IDE "Find All References".
const (
	TopicBlockchainEvents = "blockchain-events"
	TopicReorgDetected    = "reorg-detected"
	TopicReorgRollback    = "reorg-rollback"
)
