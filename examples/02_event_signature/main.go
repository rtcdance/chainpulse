package main

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
)

// EventSignature 计算 Solidity 事件的 keccak256 签名哈希
// 对应生产代码: pkg/core/event_signature.go
//
// Solidity 中: keccak256("EventName(type1,type2,...)")
// 例如: keccak256("Transfer(address,address,uint256)")
func EventSignature(eventName string, paramTypes ...string) common.Hash {
	sig := fmt.Sprintf("%s(%s)", eventName, strings.Join(paramTypes, ","))
	return keccak256Hash([]byte(sig))
}

// keccak256Hash 计算输入的 Keccak-256 哈希
func keccak256Hash(data []byte) common.Hash {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(data)
	return common.BytesToHash(hasher.Sum(nil))
}

// EncodeIndexedParam 将 indexed 参数编码为 32 字节的 topic
// 对应生产代码: pkg/core/event_signature.go 的 EncodeIndexedParam
func EncodeAddressToTopic(addr common.Address) common.Hash {
	return common.BytesToHash(addr.Bytes())
}

func main() {
	// 1. 计算 ERC-20 Transfer 事件签名
	transferSig := EventSignature("Transfer", "address", "address", "uint256")
	fmt.Printf("Transfer event signature:\n")
	fmt.Printf("  %s\n\n", transferSig.Hex())

	// 2. 验证已知哈希 (来自 EIP-20)
	expected := "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	fmt.Printf("Expected:\n")
	fmt.Printf("  %s\n\n", expected)

	if transferSig.Hex() == expected {
		fmt.Println("✓ Transfer signature matches!")
	} else {
		fmt.Println("✗ Transfer signature mismatch!")
	}

	// 3. 演示 indexed 参数编码
	alice := common.HexToAddress("0xAb5801a7D398351b8bE11C439e05C5b3259aeC9B")
	bob := common.HexToAddress("0x5B38Da6a701c568545dCfcB03FcB875f56beddC4")

	topic1 := EncodeAddressToTopic(alice)
	topic2 := EncodeAddressToTopic(bob)

	fmt.Printf("\nIndexed parameters encoding:\n")
	fmt.Printf("  From (topic[1]): %s\n", topic1.Hex())
	fmt.Printf("  To   (topic[2]): %s\n", topic2.Hex())

	// 4. 自定义事件签名
	customSig := EventSignature("CustomEvent", "bytes32", "string", "uint256")
	fmt.Printf("\nCustomEvent signature:\n")
	fmt.Printf("  %s\n", customSig.Hex())
}
