package core

import "github.com/rtcdance/chainpulse/pkg/core/topics"

// Topic constants re-exported for backward compatibility.
const (
	TopicBlockchainEvents = topics.TopicBlockchainEvents
	TopicReorgDetected    = topics.TopicReorgDetected
	TopicReorgRollback    = topics.TopicReorgRollback
)