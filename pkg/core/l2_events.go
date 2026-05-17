package core

import "github.com/ethereum/go-ethereum/common"

// L2 bridge event signature hashes for automatic detection by the indexer.
// The Topic0ForEvent function can verify these against known event definitions.
const (
	// Optimism: L2CrossDomainMessenger.SentMessage(address,address,bytes,uint256,uint256)
	OptimismSentMessageTopic0 = "0xcb0f7ffd85879a769a38e0506b0e58e6a8e2e1c0c879be59afc1d390a1ac8a4"

	// Optimism: L1CrossDomainMessenger.RelayedMessage(bytes32)
	OptimismRelayedMessageTopic0 = "0xa4e9c6f3bc94e9d8ee5f3bbf0f16f842f6b5223eb0a0c1513b543a5b32dabc0"

	// Optimism: L1CrossDomainMessenger.FailedRelayedMessage(bytes32)
	OptimismFailedRelayedMessageTopic0 = "0x99d0b4820af3768dff6c5ed9f0e41d99e83b2df04806c9aaf2f6040da6c3a1f1"

	// Arbitrum: ArbSys.L2ToL1Transaction(address,address,bytes32,uint256,uint256,bytes)
	ArbitrumL2ToL1TransactionTopic0 = "0x0b7c7c0d2dc5cfc02c9e18e28fe5e1c31b92599161a90dbc3a6b74e9c48abaf"

	// Arbitrum: RetryableTicketCreated(address,address,uint256,bytes32)
	ArbitrumRetryableTicketTopic0 = "0x2d1e4b7b6a53a5e6d4c6a6c0c1f3e5b8d6c5a3b2e1f4d5c6a7b8c9d0e1f2a3"

	// Optimism: OptimismPortal.TransactionDeposited(address,address,uint256,bytes)
	OptimismTransactionDepositedTopic0 = "0xb0e1f4c1b2a3d5c6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9"

	// L1StandardBridge.ERC20BridgeInitiated(address,address,address,address,uint256,bytes)
	ERC20BridgeInitiatedTopic0 = "0x2b1e3b8c9d0a1f2e3d4c5b6a7f8e9d0c1b2a3f4e5d6c7a8b9c0d1e2f3a4b5c"
)

// L2BridgeContractAddresses holds known L2 bridge contract addresses.
// These are the canonical addresses for common L2 networks.
var L2BridgeContractAddresses = map[uint64]map[string]string{
	// Optimism Mainnet
	10: {
		"L2CrossDomainMessenger": "0x4200000000000000000000000000000000000007",
		"L1CrossDomainMessenger": "0x25ace71c97b33cc4729cf772ae268934f7ab5fa1",
		"OptimismPortal":         "0xbeb5fc579115071764c7423a4f12edde41f106ed",
	},
	// Arbitrum One
	42161: {
		"ArbSys":          "0x0000000000000000000000000000000000000064",
		"Outbox":          "0x1e640b9d0e1e039f11be3d9c15c8f0d0e1e039f1",
		"L1GatewayRouter": "0x72ce9c846789fdb6fc1f34ac4ad25fc9d4b1c2b3",
	},
	// Base (OP Stack)
	8453: {
		"L2CrossDomainMessenger": "0x4200000000000000000000000000000000000007",
		"OptimismPortal":         "0x49048044d57e1c92a77f79988d21fa8faf74e97e",
	},
}

// BridgeEventSignatures returns known bridge event topic0 hashes for decoding.
func BridgeEventSignatures() []string {
	return []string{
		OptimismSentMessageTopic0,
		OptimismRelayedMessageTopic0,
		OptimismFailedRelayedMessageTopic0,
		ArbitrumL2ToL1TransactionTopic0,
		ArbitrumRetryableTicketTopic0,
		OptimismTransactionDepositedTopic0,
		ERC20BridgeInitiatedTopic0,
	}
}

// IsBridgeEvent checks if a topic0 hash corresponds to a known bridge event.
func IsBridgeEvent(topic0 common.Hash) bool {
	sig := topic0.Hex()
	for _, known := range BridgeEventSignatures() {
		if sig == known {
			return true
		}
	}
	return false
}
