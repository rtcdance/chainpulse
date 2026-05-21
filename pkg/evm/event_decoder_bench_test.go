package evm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func BenchmarkDecodeEventDataTransfer(b *testing.B) {
	transferTopic0 := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	from := common.HexToHash("0x000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266")
	to := common.HexToHash("0x00000000000000000000000070997970c51812dc3a010c7d01b50e0d17dc79c8")
	topics := []common.Hash{transferTopic0, from, to}

	// Non-indexed uint256 value = 1000
	data := make([]byte, 32)
	data[31] = 0xe8 // 232
	data[30] = 0x03 // 232 + 3*256 = 1000

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		DecodeEventData("Transfer", topics, data)
	}
}

func BenchmarkDecodeEventDataApproval(b *testing.B) {
	approvalTopic0 := common.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925")
	owner := common.HexToHash("0x000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266")
	spender := common.HexToHash("0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
	topics := []common.Hash{approvalTopic0, owner, spender}

	data := make([]byte, 32)
	data[31] = 0xff

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		DecodeEventData("Approval", topics, data)
	}
}
