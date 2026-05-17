package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
)

// keccak256Hash 计算 Keccak-256 哈希
func keccak256Hash(data []byte) common.Hash {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(data)
	return common.BytesToHash(hasher.Sum(nil))
}

// EventSignature 计算事件签名 (topic0)
func EventSignature(eventName string, paramTypes ...string) common.Hash {
	sig := fmt.Sprintf("%s(%s)", eventName, strings.Join(paramTypes, ","))
	return keccak256Hash([]byte(sig))
}

// 任务 1: 验证 CustomEvent 签名
func TestEventSignature() {
	fmt.Println("=== Task 1: Verify CustomEvent Signature ===")

	// 计算 CustomEvent(bytes32 indexed id, string message, uint256 timestamp) 的签名
	got := EventSignature("CustomEvent", "bytes32", "string", "uint256")
	expected := "0xaf372706c3d37b209e340aba75a7960dbe9c6df4084933559e9200d32a72c0bd"

	fmt.Printf("Computed: %s\n", got.Hex())
	fmt.Printf("Expected: %s\n", expected)

	if got.Hex() == expected {
		fmt.Println("✓ PASS: Signatures match!")
	} else {
		fmt.Println("✗ FAIL: Signatures do not match!")
	}
	fmt.Println()
}

// 任务 2: 提取 indexed bytes32 参数
func TestExtractIndexedParam() {
	fmt.Println("=== Task 2: Extract Indexed bytes32 Parameter ===")

	topicsHex := []string{
		"0xaf372706c3d37b209e340aba75a7960dbe9c6df4084933559e9200d32a72c0bd",
		"0xb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6",
	}

	// topics[0] 是事件签名，topics[1] 是 indexed id
	topic0 := common.HexToHash(topicsHex[0])
	topic1 := common.HexToHash(topicsHex[1])

	fmt.Printf("topic0 (event signature): %s\n", topic0.Hex())
	fmt.Printf("topic1 (indexed id):      %s\n", topic1.Hex())
	fmt.Printf("Expected topic0 match: %v\n", topic0.Hex() == "0xaf372706c3d37b209e340aba75a7960dbe9c6df4084933559e9200d32a72c0bd")
	fmt.Println()
}

// 任务 3: 解码 non-indexed data
func TestDecodeEventData() {
	fmt.Println("=== Task 3: Decode Non-Indexed Data ===")

	dataHex := "0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000001cb000000000000000000000000000000000000000000000000000000000000001968656c6c6f2066726f6d20636861696e70756c73652065326500000000000000"

	data, _ := hex.DecodeString(dataHex[2:])

	// 方法 1: 手动解码
	fmt.Println("--- Manual Decoding ---")

	// data[0:32] = offset of message
	offset := new(big.Int).SetBytes(data[0:32]).Uint64()
	fmt.Printf("Message offset: %d\n", offset)

	// data[32:64] = timestamp
	timestamp := new(big.Int).SetBytes(data[32:64])
	fmt.Printf("Timestamp: %s\n", timestamp.String())

	// data[64:96] = message length
	msgLen := new(big.Int).SetBytes(data[64:96]).Uint64()
	fmt.Printf("Message length: %d\n", msgLen)

	// data[offset+32:offset+32+msgLen] = message content
	// offset 指向 length 字段，实际内容在 length 之后
	messageContent := string(data[offset+32 : offset+32+msgLen])
	fmt.Printf("Message: %q\n", messageContent)

	// 验证
	if messageContent == "hello from chainpulse e2e" && timestamp.Cmp(big.NewInt(0)) > 0 {
		fmt.Println("✓ PASS: Manual decoding correct!")
	} else {
		fmt.Println("✗ FAIL: Manual decoding incorrect!")
	}
	fmt.Println()

	// 方法 2: 使用 go-ethereum ABI 解码
	fmt.Println("--- ABI Decoding ---")

	messageArg, _ := abi.NewType("string", "", nil)
	timestampArg, _ := abi.NewType("uint256", "", nil)

	args := abi.Arguments{
		{Name: "message", Type: messageArg},
		{Name: "timestamp", Type: timestampArg},
	}

	unpacked, err := args.Unpack(data)
	if err != nil {
		fmt.Printf("ABI unpack error: %v\n", err)
		return
	}

	abiMessage := unpacked[0].(string)
	abiTimestamp := unpacked[1].(*big.Int)

	fmt.Printf("Message (ABI): %q\n", abiMessage)
	fmt.Printf("Timestamp (ABI): %s\n", abiTimestamp.String())

	if abiMessage == "hello from chainpulse e2e" && abiTimestamp.Cmp(big.NewInt(0)) > 0 {
		fmt.Println("✓ PASS: ABI decoding correct!")
	} else {
		fmt.Println("✗ FAIL: ABI decoding incorrect!")
	}
}

func main() {
	TestEventSignature()
	TestExtractIndexedParam()
	TestDecodeEventData()

	fmt.Println("=== All tests completed ===")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Review the output above")
	fmt.Println("2. Check pkg/core/eventemitter_binding.go for the production implementation")
	fmt.Println("3. Try the exercises in examples/02_event_signature/")
}
