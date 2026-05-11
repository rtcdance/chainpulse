package core

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func makeSwapEvent(txHash string, blockNum uint64, txIndex uint64, contractAddr string, eventName string) BlockchainEvent {
	return BlockchainEvent{
		TransactionHash:  common.HexToHash(txHash),
		BlockNumber:      blockNum,
		TransactionIndex: txIndex,
		ContractAddress:  common.HexToAddress(contractAddr),
		EventName:        eventName,
		ChainID:          "ethereum",
	}
}

func TestDetectSandwichAttackDetected(t *testing.T) {
	tokenAddr := "0xabcdef1234567890abcdef1234567890abcdef12"
	events := []BlockchainEvent{
		makeSwapEvent("0x01", 100, 1, tokenAddr, "Swap"),
		makeSwapEvent("0x02", 100, 2, tokenAddr, "Swap"), // victim
		makeSwapEvent("0x03", 100, 3, tokenAddr, "Swap"),
	}

	detections := DetectSandwichAttack(events)
	if len(detections) == 0 {
		t.Fatal("expected at least one sandwich detection")
	}

	d := detections[0]
	if d.VictimTxHash != common.HexToHash("0x02") {
		t.Errorf("victim tx hash mismatch: got %s", d.VictimTxHash.Hex())
	}
	if d.FrontrunTxHash != common.HexToHash("0x01") {
		t.Errorf("frontrun tx hash mismatch: got %s", d.FrontrunTxHash.Hex())
	}
	if d.BackrunTxHash != common.HexToHash("0x03") {
		t.Errorf("backrun tx hash mismatch: got %s", d.BackrunTxHash.Hex())
	}
	if d.TokenAddress != common.HexToAddress(tokenAddr) {
		t.Errorf("token address mismatch: got %s", d.TokenAddress.Hex())
	}
	if d.Confidence < 0.3 {
		t.Errorf("confidence too low: %f", d.Confidence)
	}
}

func TestDetectSandwichAttackNoSandwichSingleSwap(t *testing.T) {
	events := []BlockchainEvent{
		makeSwapEvent("0x01", 100, 1, "0xaaa", "Swap"),
	}

	detections := DetectSandwichAttack(events)
	if len(detections) != 0 {
		t.Errorf("expected 0 detections for single swap, got %d", len(detections))
	}
}

func TestDetectSandwichAttackEmptyBlock(t *testing.T) {
	detections := DetectSandwichAttack(nil)
	if len(detections) != 0 {
		t.Errorf("expected 0 detections for nil events, got %d", len(detections))
	}
}

func TestDetectSandwichAttackDifferentTokens(t *testing.T) {
	events := []BlockchainEvent{
		makeSwapEvent("0x01", 100, 1, "0xaaa", "Swap"),
		makeSwapEvent("0x02", 100, 2, "0xbbb", "Swap"), // different token — not a sandwich target
		makeSwapEvent("0x03", 100, 3, "0xaaa", "Swap"),
	}

	detections := DetectSandwichAttack(events)
	// Should not detect sandwich because the victim interacts with a different token
	if len(detections) != 0 {
		t.Errorf("expected 0 detections for different-token scenario, got %d", len(detections))
	}
}

func TestDetectSandwichAttackDifferentBlocks(t *testing.T) {
	tokenAddr := "0xabcdef1234567890abcdef1234567890abcdef12"
	events := []BlockchainEvent{
		makeSwapEvent("0x01", 99, 1, tokenAddr, "Swap"),
		makeSwapEvent("0x02", 100, 1, tokenAddr, "Swap"), // different block
		makeSwapEvent("0x03", 101, 1, tokenAddr, "Swap"), // different block
	}

	detections := DetectSandwichAttack(events)
	if len(detections) != 0 {
		t.Errorf("expected 0 detections across different blocks, got %d", len(detections))
	}
}
