package core

import "github.com/rtcdance/chainpulse/pkg/logkeys"

// LogKey* constants are defined in pkg/logkeys.
// These forwarding constants preserve existing callers during migration.
const (
	LogKeyComponent        = logkeys.LogKeyComponent
	LogKeyError            = logkeys.LogKeyError
	LogKeyEventID          = logkeys.LogKeyEventID
	LogKeyBlockNumber      = logkeys.LogKeyBlockNumber
	LogKeyNetwork          = logkeys.LogKeyNetwork
	LogKeyChainID          = logkeys.LogKeyChainID
	LogKeyHash             = logkeys.LogKeyHash
	LogKeyKey              = logkeys.LogKeyKey
	LogKeyCount            = logkeys.LogKeyCount
	LogKeyAttempt          = logkeys.LogKeyAttempt
	LogKeyBatchSize        = logkeys.LogKeyBatchSize
	LogKeyProcessed        = logkeys.LogKeyProcessed
	LogKeyTTLMs            = logkeys.LogKeyTTLMs
	LogKeyExpiredEntries   = logkeys.LogKeyExpiredEntries
	LogKeyReorgBlock       = logkeys.LogKeyReorgBlock
	LogKeyCurrentBlock     = logkeys.LogKeyCurrentBlock
	LogKeyBlocksAffected   = logkeys.LogKeyBlocksAffected
	LogKeyBlocksRolledBack = logkeys.LogKeyBlocksRolledBack
	LogKeyEventsRolledBack = logkeys.LogKeyEventsRolledBack
	LogKeyFromBlock        = logkeys.LogKeyFromBlock
	LogKeyToBlock          = logkeys.LogKeyToBlock
	LogKeyInvalidated      = logkeys.LogKeyInvalidated
	LogKeyContractAddress  = logkeys.LogKeyContractAddress
	LogKeyEventName        = logkeys.LogKeyEventName
	LogKeyABIName          = logkeys.LogKeyABIName
	LogKeyOldBlockHash     = logkeys.LogKeyOldBlockHash
	LogKeyNewBlockHash     = logkeys.LogKeyNewBlockHash
	LogKeyAddress          = logkeys.LogKeyAddress
	LogKeyPool             = logkeys.LogKeyPool
	LogKeyToken0           = logkeys.LogKeyToken0
	LogKeyToken1           = logkeys.LogKeyToken1
	LogKeySender           = logkeys.LogKeySender
	LogKeyRecipient        = logkeys.LogKeyRecipient
	LogKeyAmount           = logkeys.LogKeyAmount
	LogKeySwapAmount0In    = logkeys.LogKeySwapAmount0In
	LogKeySwapAmount1In    = logkeys.LogKeySwapAmount1In
	LogKeySwapAmount0Out   = logkeys.LogKeySwapAmount0Out
	LogKeySwapAmount1Out   = logkeys.LogKeySwapAmount1Out
	LogKeyNonce            = logkeys.LogKeyNonce
	LogKeyPaymaster        = logkeys.LogKeyPaymaster
	LogKeyUserOpHash       = logkeys.LogKeyUserOpHash
	LogKeyConnectionID     = logkeys.LogKeyConnectionID
	LogKeyRemoteAddr       = logkeys.LogKeyRemoteAddr
	LogKeyTopic            = logkeys.LogKeyTopic
	LogKeySubscriptionID   = logkeys.LogKeySubscriptionID
	LogKeyConsumerGroup    = logkeys.LogKeyConsumerGroup
	LogKeyPartition        = logkeys.LogKeyPartition
	LogKeyOffset           = logkeys.LogKeyOffset
	LogKeyMessageID        = logkeys.LogKeyMessageID
	LogKeyDuration         = logkeys.LogKeyDuration
	LogKeyDurationMs       = logkeys.LogKeyDurationMs
	LogKeyStatusCode       = logkeys.LogKeyStatusCode
	LogKeyMethod           = logkeys.LogKeyMethod
	LogKeyPath             = logkeys.LogKeyPath
	LogKeyQuery            = logkeys.LogKeyQuery
	LogKeyClientIP         = logkeys.LogKeyClientIP
	LogKeyUserAgent        = logkeys.LogKeyUserAgent
	LogKeyRequestID        = logkeys.LogKeyRequestID
	LogKeyServiceName      = logkeys.LogKeyServiceName
	LogKeyInstanceID       = logkeys.LogKeyInstanceID
	LogKeyEndpoint         = logkeys.LogKeyEndpoint
	LogKeyRetryAfter       = logkeys.LogKeyRetryAfter
	LogKeyMaxRetries       = logkeys.LogKeyMaxRetries
	LogKeyRetryCount       = logkeys.LogKeyRetryCount
	LogKeyDatabaseType     = logkeys.LogKeyDatabaseType
	LogKeyCacheType        = logkeys.LogKeyCacheType
	LogKeyMQType           = logkeys.LogKeyMQType
	LogKeyPullerType       = logkeys.LogKeyPullerType
	LogKeyDeploymentMode   = logkeys.LogKeyDeploymentMode
	LogKeyAdapterProfile   = logkeys.LogKeyAdapterProfile
	LogKeyPort             = logkeys.LogKeyPort
)
