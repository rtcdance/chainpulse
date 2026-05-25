package evm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestNewChainedDecoder(t *testing.T) {
	t.Parallel()
	d := NewChainedDecoder()
	if d == nil {
		t.Fatal("expected non-nil decoder")
	}
	if d.contracts == nil {
		t.Error("expected non-nil contracts map")
	}
}

func TestChainedDecoder_RegisterABI_EmptyName(t *testing.T) {
	t.Parallel()
	d := NewChainedDecoder()
	err := d.RegisterABI("", nil)
	if err == nil {
		t.Error("expected error for empty contract name")
	}
}

func TestChainedDecoder_RegisterABI_NilABI(t *testing.T) {
	t.Parallel()
	d := NewChainedDecoder()
	err := d.RegisterABI("test", nil)
	if err == nil {
		t.Error("expected error for nil ABI")
	}
}

func TestChainedDecoder_ResolveEventName_Unknown(t *testing.T) {
	t.Parallel()
	d := NewChainedDecoder()
	hash := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	name, ok := d.ResolveEventName(hash)
	if ok {
		t.Errorf("expected not found, got %s", name)
	}
	if name != "" {
		t.Errorf("expected empty name, got %s", name)
	}
}

func TestChainedDecoder_RawHexFallback_Unknown(t *testing.T) {
	t.Parallel()
	d := NewChainedDecoder()
	topic0 := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	result := d.Decode("Unknown", []common.Hash{topic0}, []byte{0x01, 0x02})
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result["_raw"] != true {
		t.Error("expected _raw marker")
	}
	if result["_data"] == nil {
		t.Error("expected _data field")
	}
	label, _ := result["_eventLabel"].(string)
	if label == "" {
		t.Error("expected _eventLabel")
	}
}

func TestChainedDecoder_RawHexFallback_EmptyEventName(t *testing.T) {
	t.Parallel()
	d := NewChainedDecoder()
	topic0 := common.HexToHash("0xcafebabecafebabecafebabecafebabecafebabecafebabecafebabecafebabe")

	result := d.Decode("", []common.Hash{topic0}, nil)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	label, _ := result["_eventLabel"].(string)
	if label == "" {
		t.Error("expected _eventLabel")
	}
}

func TestTypedEvent_ERC20Transfer(t *testing.T) {
	t.Parallel()
	e := &ERC20Transfer{From: common.HexToAddress("0x1"), To: common.HexToAddress("0x2"), Value: big.NewInt(100)}
	if e.EventName() != "Transfer" {
		t.Errorf("expected Transfer, got %s", e.EventName())
	}
	topic := e.Topic0()
	if topic == (common.Hash{}) {
		t.Error("expected non-zero topic0 for Transfer")
	}
}

func TestTypedEvent_ERC20Approval(t *testing.T) {
	t.Parallel()
	e := &ERC20Approval{Owner: common.HexToAddress("0x1"), Spender: common.HexToAddress("0x2"), Value: big.NewInt(200)}
	if e.EventName() != "Approval" {
		t.Errorf("expected Approval, got %s", e.EventName())
	}
	topic := e.Topic0()
	if topic == (common.Hash{}) {
		t.Error("expected non-zero topic0 for Approval")
	}
}

func TestTypedEvent_ERC721ApprovalForAll(t *testing.T) {
	t.Parallel()
	e := &ERC721ApprovalForAll{Owner: common.HexToAddress("0x1"), Operator: common.HexToAddress("0x2"), Approved: true}
	if e.EventName() != "ApprovalForAll" {
		t.Errorf("expected ApprovalForAll, got %s", e.EventName())
	}
	topic := e.Topic0()
	if topic == (common.Hash{}) {
		t.Error("expected non-zero topic0 for ApprovalForAll")
	}
}

func TestTypedEvent_ERC1155TransferSingle(t *testing.T) {
	t.Parallel()
	e := &ERC1155TransferSingle{
		Operator: common.HexToAddress("0x1"),
		From:     common.HexToAddress("0x2"),
		To:       common.HexToAddress("0x3"),
		ID:       big.NewInt(42),
		Value:    big.NewInt(10),
	}
	if e.EventName() != "TransferSingle" {
		t.Errorf("expected TransferSingle, got %s", e.EventName())
	}
	topic := e.Topic0()
	if topic == (common.Hash{}) {
		t.Error("expected non-zero topic0 for TransferSingle")
	}
}

func TestTypedEvent_ERC1155TransferBatch(t *testing.T) {
	t.Parallel()
	e := &ERC1155TransferBatch{
		Operator: common.HexToAddress("0x1"),
		From:     common.HexToAddress("0x2"),
		To:       common.HexToAddress("0x3"),
		IDs:      []*big.Int{big.NewInt(1), big.NewInt(2)},
		Values:   []*big.Int{big.NewInt(10), big.NewInt(20)},
	}
	if e.EventName() != "TransferBatch" {
		t.Errorf("expected TransferBatch, got %s", e.EventName())
	}
	topic := e.Topic0()
	if topic == (common.Hash{}) {
		t.Error("expected non-zero topic0 for TransferBatch")
	}
}

func TestTypedEvent_ERC1155URI(t *testing.T) {
	t.Parallel()
	e := &ERC1155URI{Value: "https://token.com/1", ID: big.NewInt(1)}
	if e.EventName() != "URI" {
		t.Errorf("expected URI, got %s", e.EventName())
	}
	topic := e.Topic0()
	if topic == (common.Hash{}) {
		t.Error("expected non-zero topic0 for URI")
	}
}

func TestTypedEvent_UniswapV3Swap(t *testing.T) {
	t.Parallel()
	e := &UniswapV3Swap{
		Sender:       common.HexToAddress("0x1"),
		Amount0:      big.NewInt(1000),
		Amount1:      big.NewInt(2000),
		SqrtPriceX96: big.NewInt(12345),
		Liquidity:    big.NewInt(67890),
		Tick:         big.NewInt(-100),
	}
	if e.EventName() != "Swap" {
		t.Errorf("expected Swap, got %s", e.EventName())
	}
	topic := e.Topic0()
	if topic == (common.Hash{}) {
		t.Error("expected non-zero topic0 for Swap")
	}
}

func TestDecodeTypedEvent_NoTopics(t *testing.T) {
	t.Parallel()
	result, ok := DecodeTypedEvent(nil, nil)
	if ok {
		t.Error("expected false for no topics")
	}
	if result != nil {
		t.Error("expected nil result")
	}
}

func TestDecodeTypedEvent_EmptyTopics(t *testing.T) {
	t.Parallel()
	result, ok := DecodeTypedEvent([]common.Hash{}, nil)
	if ok {
		t.Error("expected false for empty topics")
	}
	if result != nil {
		t.Error("expected nil result")
	}
}

func TestDecodeTypedEvent_UnknownTopic0(t *testing.T) {
	t.Parallel()
	unknownTopic := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	result, ok := DecodeTypedEvent([]common.Hash{unknownTopic}, nil)
	if ok {
		t.Error("expected false for unknown topic0")
	}
	if result != nil {
		t.Error("expected nil result")
	}
}

func TestDecodeTypedEvent_Transfer(t *testing.T) {
	t.Parallel()
	transferTopic := topic0ForName("Transfer")
	if transferTopic == (common.Hash{}) {
		t.Skip("ABI not available for Transfer")
	}

	from := common.HexToHash("0x000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266")
	to := common.HexToHash("0x00000000000000000000000070997970c51812dc3a010c7d01b50e0d17dc79c8")
	val := make([]byte, 32)
	big.NewInt(1000).FillBytes(val)

	result, ok := DecodeTypedEvent([]common.Hash{transferTopic, from, to}, val)
	if !ok {
		t.Fatal("expected true for Transfer event")
	}
	transfer, ok := result.(*ERC20Transfer)
	if !ok {
		t.Fatalf("expected *ERC20Transfer, got %T", result)
	}
	if transfer.Value.Cmp(big.NewInt(1000)) != 0 {
		t.Errorf("expected value 1000, got %s", transfer.Value)
	}
}

func TestDecodeTypedEvent_Approval(t *testing.T) {
	t.Parallel()
	topic := topic0ForName("Approval")
	if topic == (common.Hash{}) {
		t.Skip("ABI not available for Approval")
	}
	owner := common.HexToHash("0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	spender := common.HexToHash("0x000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	val := make([]byte, 32)
	big.NewInt(500).FillBytes(val)

	result, ok := DecodeTypedEvent([]common.Hash{topic, owner, spender}, val)
	if !ok {
		t.Fatal("expected true for Approval event")
	}
	approval, ok := result.(*ERC20Approval)
	if !ok {
		t.Fatalf("expected *ERC20Approval, got %T", result)
	}
	if approval.Value.Cmp(big.NewInt(500)) != 0 {
		t.Errorf("expected value 500, got %s", approval.Value)
	}
}

func TestChainedDecoder_Decode_UnknownEvent(t *testing.T) {
	t.Parallel()
	d := NewChainedDecoder()
	topic0 := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	result := d.Decode("RandomEvent", []common.Hash{topic0}, nil)
	if result == nil {
		t.Fatal("chained decoder should always return a map (raw fallback)")
	}
	if result["_raw"] != true {
		t.Error("expected _raw marker for unknown event")
	}
}

func TestChainedDecoder_DecodeWithTypedEvent(t *testing.T) {
	t.Parallel()
	d := NewChainedDecoder()
	topic0 := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	mapResult, typedResult := d.DecodeWithTypedEvent("Event", []common.Hash{topic0}, nil)
	if mapResult == nil {
		t.Fatal("expected non-nil map result")
	}
	if typedResult != nil {
		t.Error("expected nil typed result for unknown event")
	}
}

func TestDecodeEvent(t *testing.T) {
	t.Parallel()
	topic0 := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	result := DecodeEvent("Test", []common.Hash{topic0}, nil)
	if result == nil {
		t.Fatal("DecodeEvent should always return a map")
	}
}

func TestDecodeEventWithTyped(t *testing.T) {
	t.Parallel()
	topic0 := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	mapResult, typedResult := DecodeEventWithTyped("Test", []common.Hash{topic0}, nil)
	if mapResult == nil {
		t.Fatal("DecodeEventWithTyped should return a map")
	}
	if typedResult != nil {
		t.Error("expected nil typed result for unknown event")
	}
}

func TestTopic0ForName_Unknown(t *testing.T) {
	t.Parallel()
	result := topic0ForName("NonExistentEventName")
	if result != (common.Hash{}) {
		t.Error("expected zero hash for unknown event")
	}
}

func TestDecodeTypedEvent_ApprovalForAll(t *testing.T) {
	t.Parallel()
	topic := topic0ForName("ApprovalForAll")
	if topic == (common.Hash{}) {
		t.Skip("ABI not available for ApprovalForAll")
	}
	owner := common.HexToHash("0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	operator := common.HexToHash("0x000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	approved := make([]byte, 32)
	approved[31] = 1

	result, ok := DecodeTypedEvent([]common.Hash{topic, owner, operator}, approved)
	if !ok {
		t.Fatal("expected true for ApprovalForAll event")
	}
	e, ok := result.(*ERC721ApprovalForAll)
	if !ok {
		t.Fatalf("expected *ERC721ApprovalForAll, got %T", result)
	}
	if !e.Approved {
		t.Error("expected approved=true")
	}
}

func TestDecodeTypedEvent_TransferSingle(t *testing.T) {
	t.Parallel()
	topic := topic0ForName("TransferSingle")
	if topic == (common.Hash{}) {
		t.Skip("ABI not available for TransferSingle")
	}
	op := common.HexToHash("0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	from := common.HexToHash("0x000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	to := common.HexToHash("0x000000000000000000000000cccccccccccccccccccccccccccccccccccccccc")
	data := make([]byte, 64)
	big.NewInt(42).FillBytes(data[:32])
	big.NewInt(10).FillBytes(data[32:64])

	result, ok := DecodeTypedEvent([]common.Hash{topic, op, from, to}, data)
	if !ok {
		t.Fatal("expected true for TransferSingle event")
	}
	e, ok := result.(*ERC1155TransferSingle)
	if !ok {
		t.Fatalf("expected *ERC1155TransferSingle, got %T", result)
	}
	if e.ID.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("expected ID=42, got %s", e.ID)
	}
}

func TestDecodeTypedEvent_Swap(t *testing.T) {
	t.Parallel()
	topic := topic0ForName("Swap")
	if topic == (common.Hash{}) {
		t.Skip("ABI not available for Swap")
	}
	sender := common.HexToHash("0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	data := make([]byte, 160)
	big.NewInt(1000).FillBytes(data[:32])
	big.NewInt(2000).FillBytes(data[32:64])
	big.NewInt(12345).FillBytes(data[64:96])
	big.NewInt(67890).FillBytes(data[96:128])
	big.NewInt(100).FillBytes(data[128:160])

	result, ok := DecodeTypedEvent([]common.Hash{topic, sender}, data)
	if !ok {
		t.Fatal("expected true for Swap event")
	}
	e, ok := result.(*UniswapV3Swap)
	if !ok {
		t.Fatalf("expected *UniswapV3Swap, got %T", result)
	}
	if e.Amount0.Cmp(big.NewInt(1000)) != 0 {
		t.Errorf("expected Amount0=1000, got %s", e.Amount0)
	}
}

func TestChainedDecoder_DefaultDecoder(t *testing.T) {
	t.Parallel()
	if DefaultDecoder == nil {
		t.Fatal("expected non-nil DefaultDecoder")
	}
}

func TestDecodeTypedEvent_TransferBatch(t *testing.T) {
	t.Parallel()
	topic := topic0ForName("TransferBatch")
	if topic == (common.Hash{}) {
		t.Skip("ABI not available for TransferBatch")
	}

	op := common.HexToHash("0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	from := common.HexToHash("0x000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	to := common.HexToHash("0x000000000000000000000000cccccccccccccccccccccccccccccccccccccccc")

	abi := GetABIForEventName("TransferBatch")
	ev := abi.Events["TransferBatch"]
	nonIndexed := ev.Inputs.NonIndexed()

	ids := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
	values := []*big.Int{big.NewInt(100), big.NewInt(200), big.NewInt(300)}
	data, err := nonIndexed.Pack(ids, values)
	if err != nil {
		t.Fatalf("failed to pack TransferBatch data: %v", err)
	}

	result, ok := DecodeTypedEvent([]common.Hash{topic, op, from, to}, data)
	if !ok {
		t.Fatal("expected true for TransferBatch event")
	}

	e, ok := result.(*ERC1155TransferBatch)
	if !ok {
		t.Fatalf("expected *ERC1155TransferBatch, got %T", result)
	}

	if e.Operator != common.HexToAddress("0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa") {
		t.Errorf("unexpected operator")
	}
	if e.From != common.HexToAddress("0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB") {
		t.Errorf("unexpected from")
	}
	if e.To != common.HexToAddress("0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccCCc") {
		t.Errorf("unexpected to")
	}
	if len(e.IDs) != 3 || e.IDs[0].Cmp(big.NewInt(1)) != 0 {
		t.Errorf("expected IDs [1,2,3], got %v", e.IDs)
	}
	if len(e.Values) != 3 || e.Values[0].Cmp(big.NewInt(100)) != 0 {
		t.Errorf("expected Values [100,200,300], got %v", e.Values)
	}
}

func TestDecodeTypedEvent_TransferBatch_NoData(t *testing.T) {
	t.Parallel()
	topic := topic0ForName("TransferBatch")
	if topic == (common.Hash{}) {
		t.Skip("ABI not available for TransferBatch")
	}

	op := common.HexToHash("0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	from := common.HexToHash("0x000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	to := common.HexToHash("0x000000000000000000000000cccccccccccccccccccccccccccccccccccccccc")

	result, ok := DecodeTypedEvent([]common.Hash{topic, op, from, to}, nil)
	if !ok {
		t.Fatal("expected true for TransferBatch event with no data")
	}

	e, ok := result.(*ERC1155TransferBatch)
	if !ok {
		t.Fatalf("expected *ERC1155TransferBatch, got %T", result)
	}

	if e.Operator != common.HexToAddress("0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa") {
		t.Errorf("unexpected operator")
	}
	if len(e.IDs) != 0 {
		t.Errorf("expected empty IDs, got %v", e.IDs)
	}
	if len(e.Values) != 0 {
		t.Errorf("expected empty Values, got %v", e.Values)
	}
}

func TestDecodeTypedEvent_URI(t *testing.T) {
	t.Parallel()
	topic := topic0ForName("URI")
	if topic == (common.Hash{}) {
		t.Skip("ABI not available for URI")
	}

	id := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")

	abi := GetABIForEventName("URI")
	ev := abi.Events["URI"]
	nonIndexed := ev.Inputs.NonIndexed()

	data, err := nonIndexed.Pack("https://example.com/token/1")
	if err != nil {
		t.Fatalf("failed to pack URI data: %v", err)
	}

	result, ok := DecodeTypedEvent([]common.Hash{topic, id}, data)
	if !ok {
		t.Fatal("expected true for URI event")
	}

	e, ok := result.(*ERC1155URI)
	if !ok {
		t.Fatalf("expected *ERC1155URI, got %T", result)
	}
	if e.Value != "https://example.com/token/1" {
		t.Errorf("expected URI, got %s", e.Value)
	}
	if e.ID.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("expected ID=1, got %s", e.ID)
	}
}

func TestDecodeTypedEvent_URI_NoData(t *testing.T) {
	t.Parallel()
	topic := topic0ForName("URI")
	if topic == (common.Hash{}) {
		t.Skip("ABI not available for URI")
	}

	id := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")

	result, ok := DecodeTypedEvent([]common.Hash{topic, id}, nil)
	if !ok {
		t.Fatal("expected true for URI event with no data")
	}

	e, ok := result.(*ERC1155URI)
	if !ok {
		t.Fatalf("expected *ERC1155URI, got %T", result)
	}
	if e.Value != "" {
		t.Errorf("expected empty URI, got %s", e.Value)
	}
	if e.ID.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("expected ID=1, got %s", e.ID)
	}
}
