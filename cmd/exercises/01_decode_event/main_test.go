package main

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// Test 1: 验证 CustomEvent 事件签名
func TestCustomEventSignature(t *testing.T) {
	got := EventSignature("CustomEvent", "bytes32", "string", "uint256")
	expected := common.HexToHash("0xaf372706c3d37b209e340aba75a7960dbe9c6df4084933559e9200d32a72c0bd")

	if got != expected {
		t.Errorf("CustomEvent signature mismatch:\n  got:      %s\n  expected: %s", got.Hex(), expected.Hex())
	}
}

// Test 2: 验证 Transfer 事件签名 (EIP-20)
func TestTransferEventSignature(t *testing.T) {
	got := EventSignature("Transfer", "address", "address", "uint256")
	expected := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

	if got != expected {
		t.Errorf("Transfer signature mismatch:\n  got:      %s\n  expected: %s", got.Hex(), expected.Hex())
	}
}

// Test 3: 验证 indexed bytes32 参数可以直接从 topic 读取
func TestIndexedBytes32Extraction(t *testing.T) {
	topic1 := common.HexToHash("0xb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6")

	// bytes32 indexed 参数直接存储在 topic 中，无需额外解码
	id := topic1

	if id.Hex() != "0xb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6" {
		t.Errorf("ID extraction failed: got %s", id.Hex())
	}
}

// Test 4: 验证 indexed address 编码为零左填充
func TestIndexedAddressEncoding(t *testing.T) {
	addr := common.HexToAddress("0xAb5801a7D398351b8bE11C439e05C5b3259aeC9B")
	topic := common.BytesToHash(addr.Bytes())

	// 验证前 12 字节为 0
	for i := 0; i < 12; i++ {
		if topic[i] != 0 {
			t.Errorf("Address topic should be zero-padded at byte %d", i)
		}
	}

	// 验证后 20 字节是地址
	if topic.Hex()[26:] != "ab5801a7d398351b8be11c439e05c5b3259aec9b" {
		t.Errorf("Address topic mismatch: got %s", topic.Hex())
	}
}

// Test 5: 手动解码 non-indexed data (string + uint256)
func TestManualDataDecoding(t *testing.T) {
	dataHex := "0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000001cb000000000000000000000000000000000000000000000000000000000000001968656c6c6f2066726f6d20636861696e70756c73652065326500000000000000"
	data, err := hex.DecodeString(dataHex[2:])
	if err != nil {
		t.Fatalf("Failed to decode hex: %v", err)
	}

	// data[0:32] = message offset
	offset := new(big.Int).SetBytes(data[0:32]).Uint64()
	if offset != 64 {
		t.Errorf("Expected offset 64, got %d", offset)
	}

	// data[32:64] = timestamp
	timestamp := new(big.Int).SetBytes(data[32:64])
	expectedTimestamp := big.NewInt(459)
	if timestamp.Cmp(expectedTimestamp) != 0 {
		t.Errorf("Expected timestamp %d, got %d", expectedTimestamp, timestamp)
	}

	// data[64:96] = message length
	msgLen := new(big.Int).SetBytes(data[64:96]).Uint64()
	if msgLen != 25 {
		t.Errorf("Expected message length 25, got %d", msgLen)
	}

	// data[offset+32:offset+32+msgLen] = message content
	// offset 指向 length 字段，内容在 length 之后
	messageStart := offset + 32
	message := string(data[messageStart : messageStart+msgLen])
	if message != "hello from chainpulse e2e" {
		t.Errorf("Expected message %q, got %q", "hello from chainpulse e2e", message)
	}
}

// Test 6: 使用 ABI 解码 non-indexed data
func TestABIDataDecoding(t *testing.T) {
	dataHex := "0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000001cb000000000000000000000000000000000000000000000000000000000000001968656c6c6f2066726f6d20636861696e70756c73652065326500000000000000"
	data, err := hex.DecodeString(dataHex[2:])
	if err != nil {
		t.Fatalf("Failed to decode hex: %v", err)
	}

	messageArg, _ := abi.NewType("string", "", nil)
	timestampArg, _ := abi.NewType("uint256", "", nil)

	args := abi.Arguments{
		{Name: "message", Type: messageArg},
		{Name: "timestamp", Type: timestampArg},
	}

	unpacked, err := args.Unpack(data)
	if err != nil {
		t.Fatalf("ABI unpack failed: %v", err)
	}

	message := unpacked[0].(string)
	timestamp := unpacked[1].(*big.Int)

	if message != "hello from chainpulse e2e" {
		t.Errorf("Expected message %q, got %q", "hello from chainpulse e2e", message)
	}

	expectedTimestamp := big.NewInt(459)
	if timestamp.Cmp(expectedTimestamp) != 0 {
		t.Errorf("Expected timestamp %d, got %d", expectedTimestamp, timestamp)
	}
}

// Test 7: 验证 indexed string 只存储哈希
func TestIndexedStringHashing(t *testing.T) {
	originalString := "hello from chainpulse e2e"
	hash := keccak256Hash([]byte(originalString))

	// indexed string 不存储原始值，只存储哈希
	// 这意味着无法从 topic 恢复原始字符串
	if hash.Hex() == "" || hash == (common.Hash{}) {
		t.Error("String hash should not be empty")
	}

	// 验证同一个字符串总是产生相同的哈希
	hash2 := keccak256Hash([]byte(originalString))
	if hash != hash2 {
		t.Error("Same string should produce same hash")
	}
}

// Test 8: 对比手动解码和 ABI 解码结果一致
func TestManualVsABIDecoding(t *testing.T) {
	dataHex := "0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000001cb000000000000000000000000000000000000000000000000000000000000001968656c6c6f2066726f6d20636861696e70756c73652065326500000000000000"
	data, _ := hex.DecodeString(dataHex[2:])

	// 手动解码
	offset := new(big.Int).SetBytes(data[0:32]).Uint64()
	manualTimestamp := new(big.Int).SetBytes(data[32:64])
	manualMsgLen := new(big.Int).SetBytes(data[64:96]).Uint64()
	messageStart := offset + 32
	manualMessage := string(data[messageStart : messageStart+manualMsgLen])

	// ABI 解码
	messageArg, _ := abi.NewType("string", "", nil)
	timestampArg, _ := abi.NewType("uint256", "", nil)
	args := abi.Arguments{
		{Name: "message", Type: messageArg},
		{Name: "timestamp", Type: timestampArg},
	}
	unpacked, _ := args.Unpack(data)
	abiMessage := unpacked[0].(string)
	abiTimestamp := unpacked[1].(*big.Int)

	if manualMessage != abiMessage {
		t.Errorf("Message mismatch: manual=%q abi=%q", manualMessage, abiMessage)
	}

	if manualTimestamp.Cmp(abiTimestamp) != 0 {
		t.Errorf("Timestamp mismatch: manual=%s abi=%s", manualTimestamp, abiTimestamp)
	}
}
