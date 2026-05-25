package evm

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func TestDecodeWithABI(t *testing.T) {
	t.Parallel()

	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	addressType, err := abi.NewType("address", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	eventABI := abi.NewEvent("Transfer", "Transfer", false, abi.Arguments{
		{Name: "from", Type: addressType, Indexed: true},
		{Name: "to", Type: addressType, Indexed: true},
		{Name: "value", Type: uint256Type, Indexed: false},
	})

	fromAddr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	toAddr := common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")

	topics := []common.Hash{
		eventABI.ID,
		common.BytesToHash(fromAddr.Bytes()),
		common.BytesToHash(toAddr.Bytes()),
	}

	nonIndexedArgs := abi.Arguments{{Name: "value", Type: uint256Type}}
	value := big.NewInt(1000000000000000000)
	data, err := nonIndexedArgs.Pack(value)
	if err != nil {
		t.Fatal(err)
	}

	result := decodeWithABI(eventABI, topics, data)

	if result["from"] != fromAddr.Hex() {
		t.Errorf("from = %v, want %s", result["from"], fromAddr.Hex())
	}
	if result["to"] != toAddr.Hex() {
		t.Errorf("to = %v, want %s", result["to"], toAddr.Hex())
	}
	if result["value"] != value.String() {
		t.Errorf("value = %v, want %s", result["value"], value.String())
	}
}

func TestDecodeWithABIAddressType(t *testing.T) {
	t.Parallel()

	addressType, err := abi.NewType("address", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	eventABI := abi.NewEvent("OwnerChanged", "OwnerChanged", false, abi.Arguments{
		{Name: "oldOwner", Type: addressType, Indexed: true},
		{Name: "newOwner", Type: addressType, Indexed: false},
	})

	oldOwner := common.HexToAddress("0x1111111111111111111111111111111111111111")
	newOwner := common.HexToAddress("0x2222222222222222222222222222222222222222")

	topics := []common.Hash{
		eventABI.ID,
		common.BytesToHash(oldOwner.Bytes()),
	}

	nonIndexedArgs := abi.Arguments{{Name: "newOwner", Type: addressType}}
	data, err := nonIndexedArgs.Pack(newOwner)
	if err != nil {
		t.Fatal(err)
	}

	result := decodeWithABI(eventABI, topics, data)
	if result["oldOwner"] != oldOwner.Hex() {
		t.Errorf("oldOwner = %v", result["oldOwner"])
	}
	if result["newOwner"] != newOwner.Hex() {
		t.Errorf("newOwner = %v", result["newOwner"])
	}
}

func TestDecodeWithABIBoolType(t *testing.T) {
	t.Parallel()

	boolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	eventABI := abi.NewEvent("Paused", "Paused", false, abi.Arguments{
		{Name: "isPaused", Type: boolType, Indexed: true},
	})

	var boolTrue common.Hash
	boolTrue[31] = 1

	topics := []common.Hash{eventABI.ID, boolTrue}
	result := decodeWithABI(eventABI, topics, nil)

	if result["isPaused"] != true {
		t.Errorf("isPaused = %v, want true", result["isPaused"])
	}
}

func TestDecodeWithABIBytes32Type(t *testing.T) {
	t.Parallel()

	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	eventABI := abi.NewEvent("HashSet", "HashSet", false, abi.Arguments{
		{Name: "hash", Type: bytes32Type, Indexed: true},
	})

	testHash := common.HexToHash("0xabcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	topics := []common.Hash{eventABI.ID, testHash}
	result := decodeWithABI(eventABI, topics, nil)

	got := result["hash"].(string)
	if !strings.HasPrefix(got, "0x") {
		t.Errorf("hash should start with 0x, got %s", got)
	}
}

func TestDecodeWithABINoNonIndexedParams(t *testing.T) {
	t.Parallel()

	boolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	eventABI := abi.NewEvent("AllIndexed", "AllIndexed", false, abi.Arguments{
		{Name: "flag", Type: boolType, Indexed: true},
	})

	var boolFalse common.Hash
	topics := []common.Hash{eventABI.ID, boolFalse}
	result := decodeWithABI(eventABI, topics, nil)

	if result["flag"] != false {
		t.Errorf("flag = %v, want false", result["flag"])
	}
}

func TestDecodeWithABIEmptyData(t *testing.T) {
	t.Parallel()

	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	eventABI := abi.NewEvent("Empty", "Empty", false, abi.Arguments{
		{Name: "amount", Type: uint256Type, Indexed: false},
	})

	result := decodeWithABI(eventABI, []common.Hash{eventABI.ID}, nil)
	if result["amount"] != nil {
		t.Errorf("amount should be nil for empty data, got %v", result["amount"])
	}
}

func TestDecodeWithABIIntType(t *testing.T) {
	t.Parallel()

	int256Type, err := abi.NewType("int256", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	eventABI := abi.NewEvent("Signed", "Signed", false, abi.Arguments{
		{Name: "val", Type: int256Type, Indexed: true},
	})

	val := big.NewInt(42)
	var hash common.Hash
	val.FillBytes(hash[:])

	topics := []common.Hash{eventABI.ID, hash}
	result := decodeWithABI(eventABI, topics, nil)

	if result["val"] != "42" {
		t.Errorf("val = %v, want 42", result["val"])
	}
}

func TestChainedDecoderResolveEventName(t *testing.T) {
	t.Parallel()

	d := NewChainedDecoder()
	name, ok := d.ResolveEventName(common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"))
	if !ok || name != "Transfer" {
		t.Errorf("expected Transfer, got %s (%v)", name, ok)
	}
}

func TestChainedDecoderResolveEventNameNotFound(t *testing.T) {
	t.Parallel()

	d := NewChainedDecoder()
	_, ok := d.ResolveEventName(common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"))
	if ok {
		t.Error("expected not found")
	}
}

func TestChainedDecoderRegisterABIError(t *testing.T) {
	t.Parallel()

	d := NewChainedDecoder()
	if err := d.RegisterABI("", nil); err == nil {
		t.Error("expected error for empty name")
	}
	if err := d.RegisterABI("Test", nil); err == nil {
		t.Error("expected error for nil ABI")
	}
}

func TestChainedDecoderDecodeWithRegistered(t *testing.T) {
	t.Parallel()

	parsedABI, err := abi.JSON(strings.NewReader(`[{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`))
	if err != nil {
		t.Fatal(err)
	}

	d := NewChainedDecoder()
	if err := d.RegisterABI("Token", &parsedABI); err != nil {
		t.Fatal(err)
	}

	fromAddr := common.HexToAddress("0xaaaa0000000000000000000000000000000000")
	toAddr := common.HexToAddress("0xbbbb0000000000000000000000000000000000")

	eventTopic := parsedABI.Events["Transfer"].ID
	topics := []common.Hash{
		eventTopic,
		common.BytesToHash(fromAddr.Bytes()),
		common.BytesToHash(toAddr.Bytes()),
	}

	uint256Type, _ := abi.NewType("uint256", "", nil)
	args := abi.Arguments{{Name: "value", Type: uint256Type}}
	data, _ := args.Pack(big.NewInt(500))

	result := d.Decode("Transfer", topics, data)
	if result["from"] != fromAddr.Hex() {
		t.Errorf("from = %v", result["from"])
	}
	if result["value"] != "500" {
		t.Errorf("value = %v", result["value"])
	}
}

func TestChainedDecoderDecodeWithTypedEvent(t *testing.T) {
	t.Parallel()

	d := NewChainedDecoder()
	transferTopic0 := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	from := common.HexToHash("0x000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266")
	to := common.HexToHash("0x00000000000000000000000070997970c51812dc3a010c7d01b50e0d17dc79c8")

	mapResult, typed := d.DecodeWithTypedEvent("Transfer", []common.Hash{transferTopic0, from, to}, nil)
	if mapResult == nil {
		t.Error("mapResult should not be nil")
	}
	if typed == nil {
		t.Error("typed should not be nil for known event")
	}
}

func TestChainedDecoderDecodeRawFallback(t *testing.T) {
	t.Parallel()

	d := NewChainedDecoder()
	unknownTopic := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	result := d.Decode("Unknown", []common.Hash{unknownTopic}, []byte{1, 2, 3})
	if result == nil {
		t.Error("expected fallback result")
	}
	if _, ok := result["_raw"]; !ok {
		t.Error("expected _raw marker")
	}
}

func TestChainedDecoderDecodeNoName(t *testing.T) {
	t.Parallel()

	d := NewChainedDecoder()
	unknownTopic := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	result := d.Decode("", []common.Hash{unknownTopic}, nil)
	if result == nil {
		t.Error("expected fallback result for empty name")
	}
	if label, ok := result["_eventLabel"]; ok {
		if label.(string) != "Unknown_0xdeadbeef" {
			t.Errorf("expected Unknown_ label, got %v", label)
		}
	}
}

func TestDecodeWithABIRawHexFallback(t *testing.T) {
	t.Parallel()

	d := NewChainedDecoder()
	topics := []common.Hash{
		common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
	}
	data := hex.EncodeToString([]byte{1, 2, 3})
	rawHex := d.rawHexFallback("CustomEvent", topics, common.FromHex("0x"+data))

	if rawHex["_eventLabel"] != "CustomEvent" {
		t.Errorf("eventLabel = %v", rawHex["_eventLabel"])
	}
	if topicsRaw, ok := rawHex["_topics"].([]string); !ok || len(topicsRaw) != 2 {
		t.Error("expected 2 topics")
	}
}
