package evm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestFormatIndexedTopicValue_Address(t *testing.T) {
	t.Parallel()
	addr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	hash := common.BytesToHash(addr.Bytes())
	result := FormatIndexedTopicValue("address", hash)
	expected := addr.Hex()
	if result != expected {
		t.Errorf("expected %s, got %v", expected, result)
	}
}

func TestFormatIndexedTopicValue_Bool_True(t *testing.T) {
	t.Parallel()
	var hash common.Hash
	hash[31] = 1
	result := FormatIndexedTopicValue("bool", hash)
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestFormatIndexedTopicValue_Bool_False(t *testing.T) {
	t.Parallel()
	var hash common.Hash
	result := FormatIndexedTopicValue("bool", hash)
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

func TestFormatIndexedTopicValue_Uint(t *testing.T) {
	t.Parallel()
	val := big.NewInt(1000000)
	var hash common.Hash
	val.FillBytes(hash[:])
	result := FormatIndexedTopicValue("uint256", hash)
	if result != val.String() {
		t.Errorf("expected %s, got %v", val.String(), result)
	}
}

func TestFormatIndexedTopicValue_Uint8(t *testing.T) {
	t.Parallel()
	val := big.NewInt(255)
	var hash common.Hash
	val.FillBytes(hash[:])
	result := FormatIndexedTopicValue("uint8", hash)
	if result != "255" {
		t.Errorf("expected 255, got %v", result)
	}
}

func TestFormatIndexedTopicValue_Int_Negative(t *testing.T) {
	t.Parallel()
	twoTo256 := new(big.Int).Lsh(big.NewInt(1), 256)
	val := new(big.Int).Sub(twoTo256, big.NewInt(42))
	var hash common.Hash
	val.FillBytes(hash[:])
	result := FormatIndexedTopicValue("int256", hash)
	if result != "-42" {
		t.Errorf("expected -42, got %v", result)
	}
}

func TestFormatIndexedTopicValue_Int_Positive(t *testing.T) {
	t.Parallel()
	val := big.NewInt(42)
	var hash common.Hash
	val.FillBytes(hash[:])
	result := FormatIndexedTopicValue("int256", hash)
	if result != "42" {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestFormatIndexedTopicValue_Default(t *testing.T) {
	t.Parallel()
	hash := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	result := FormatIndexedTopicValue("bytes32", hash)
	if result != hash.Hex() {
		t.Errorf("expected %s, got %v", hash.Hex(), result)
	}
}

func TestFormatDecodedValue_BigInt(t *testing.T) {
	t.Parallel()
	result := FormatDecodedValue(big.NewInt(12345))
	if result != "12345" {
		t.Errorf("expected 12345, got %v", result)
	}
}

func TestFormatDecodedValue_Address(t *testing.T) {
	t.Parallel()
	addr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	result := FormatDecodedValue(addr)
	if result != addr.Hex() {
		t.Errorf("expected %s, got %v", addr.Hex(), result)
	}
}

func TestFormatDecodedValue_AddressSlice(t *testing.T) {
	t.Parallel()
	addrs := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
	}
	result := FormatDecodedValue(addrs)
	slice, ok := result.([]string)
	if !ok {
		t.Fatal("expected []string")
	}
	if len(slice) != 2 {
		t.Fatalf("expected 2, got %d", len(slice))
	}
}

func TestFormatDecodedValue_BigIntSlice(t *testing.T) {
	t.Parallel()
	vals := []*big.Int{big.NewInt(1), big.NewInt(2)}
	result := FormatDecodedValue(vals)
	slice, ok := result.([]string)
	if !ok {
		t.Fatal("expected []string")
	}
	if len(slice) != 2 {
		t.Fatalf("expected 2, got %d", len(slice))
	}
}

func TestFormatDecodedValue_BigIntSlice_Nil(t *testing.T) {
	t.Parallel()
	vals := []*big.Int{big.NewInt(1), nil, big.NewInt(3)}
	result := FormatDecodedValue(vals)
	slice, ok := result.([]string)
	if !ok {
		t.Fatal("expected []string")
	}
	if slice[1] != "" {
		t.Errorf("expected empty string for nil, got %s", slice[1])
	}
}

func TestFormatDecodedValue_Bool(t *testing.T) {
	t.Parallel()
	result := FormatDecodedValue(true)
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestFormatDecodedValue_String(t *testing.T) {
	t.Parallel()
	result := FormatDecodedValue("hello")
	if result != "hello" {
		t.Errorf("expected hello, got %v", result)
	}
}

func TestFormatDecodedValue_Bytes(t *testing.T) {
	t.Parallel()
	result := FormatDecodedValue([]byte{0xde, 0xad, 0xbe, 0xef})
	if result != "0xdeadbeef" {
		t.Errorf("expected 0xdeadbeef, got %v", result)
	}
}

func TestFormatDecodedValue_Hash(t *testing.T) {
	t.Parallel()
	hash := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	result := FormatDecodedValue(hash)
	if result != hash.Hex() {
		t.Errorf("expected %s, got %v", hash.Hex(), result)
	}
}

func TestFormatDecodedValue_Bytes32(t *testing.T) {
	t.Parallel()
	var arr [32]byte
	arr[0] = 0xff
	result := FormatDecodedValue(arr)
	expected := common.Hash(arr).Hex()
	if result != expected {
		t.Errorf("expected %s, got %v", expected, result)
	}
}

func TestFormatDecodedValue_Stringer(t *testing.T) {
	t.Parallel()
	result := FormatDecodedValue(big.NewInt(999))
	if result != "999" {
		t.Errorf("expected 999 via Stringer, got %v", result)
	}
}

func TestFormatDecodedValue_DefaultStruct(t *testing.T) {
	t.Parallel()
	type unknown struct{ X int }
	result := FormatDecodedValue(unknown{X: 1})
	if result != "{1}" {
		t.Errorf("expected {1}, got %v", result)
	}
}

func TestDecodeEventData_EmptyEventName_NoTopics(t *testing.T) {
	t.Parallel()
	ResetUnknownEventSignatureCount()
	result := DecodeEventData("", nil, nil)
	if result != nil {
		t.Error("expected nil for empty event name with no topics")
	}
}

func TestDecodeEventData_UnknownWithTopics(t *testing.T) {
	t.Parallel()
	ResetUnknownEventSignatureCount()

	topic0 := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	result := DecodeEventData("NonExistent", []common.Hash{topic0}, nil)
	if result != nil {
		t.Error("expected nil for unknown event")
	}
}

func TestGetUnknownEventSignatureCount_Initial(t *testing.T) {
	ResetUnknownEventSignatureCount()
	if GetUnknownEventSignatureCount() != 0 {
		t.Error("expected 0 after reset")
	}
}
