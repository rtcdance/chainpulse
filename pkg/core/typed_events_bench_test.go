package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func BenchmarkDecodeTypedEvent_ERC20Transfer(b *testing.B) {
	from := common.HexToAddress("0x1111111111111111111111111111111111111111")
	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	value := big.NewInt(1000000)

	topics := []common.Hash{
		topic0ForName("Transfer"),
		padAddressToTopic(from),
		padAddressToTopic(to),
	}
	data := encodeUint256(value)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result, ok := DecodeTypedEvent(topics, data)
		_ = result
		_ = ok
	}
}

func BenchmarkDecodeTypedEvent_ERC1155TransferSingle(b *testing.B) {
	operator := common.HexToAddress("0x1111111111111111111111111111111111111111")
	from := common.HexToAddress("0x2222222222222222222222222222222222222222")
	to := common.HexToAddress("0x3333333333333333333333333333333333333333")

	topics := []common.Hash{
		topic0ForName("TransferSingle"),
		padAddressToTopic(operator),
		padAddressToTopic(from),
		padAddressToTopic(to),
	}
	data := append(encodeUint256(big.NewInt(42)), encodeUint256(big.NewInt(100))...)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result, ok := DecodeTypedEvent(topics, data)
		_ = result
		_ = ok
	}
}

func BenchmarkDecodeTypedEvent_UniswapV3Swap(b *testing.B) {
	sender := common.HexToAddress("0x7777777777777777777777777777777777777777")

	topics := []common.Hash{
		topic0ForName("Swap"),
		padAddressToTopic(sender),
	}
	data := make([]byte, 160) // 5 x 32 bytes

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result, ok := DecodeTypedEvent(topics, data)
		_ = result
		_ = ok
	}
}

func BenchmarkDecodeTypedEvent_UnknownTopic(b *testing.B) {
	unknownTopic := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	topics := []common.Hash{unknownTopic}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result, ok := DecodeTypedEvent(topics, nil)
		_ = result
		_ = ok
	}
}
