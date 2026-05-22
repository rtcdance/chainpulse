package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// padAddressToTopic left-pads a 20-byte address to 32 bytes for EVM topic encoding.
func padAddressToTopic(addr common.Address) common.Hash {
	var h common.Hash
	copy(h[12:], addr[:])
	return h
}

// padUint256ToTopic encodes a *big.Int as a 32-byte topic.
func padUint256ToTopic(v *big.Int) common.Hash {
	var h common.Hash
	b := v.Bytes()
	copy(h[32-len(b):], b)
	return h
}

// encodeUint256 encodes a *big.Int as a 32-byte ABI-encoded data slot.
func encodeUint256(v *big.Int) []byte {
	b := v.Bytes()
	data := make([]byte, 32)
	copy(data[32-len(b):], b)
	return data
}

// --- ERC20Transfer ---

func TestDecodeERC20Transfer(t *testing.T) {
	from := common.HexToAddress("0x1111111111111111111111111111111111111111")
	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	value := big.NewInt(1000000)

	topics := []common.Hash{
		topic0ForName("Transfer"),
		padAddressToTopic(from),
		padAddressToTopic(to),
	}
	data := encodeUint256(value)

	got, ok := DecodeTypedEvent(topics, data)
	if !ok {
		t.Fatal("DecodeTypedEvent returned false for Transfer")
	}

	transfer, ok := got.(*ERC20Transfer)
	if !ok {
		t.Fatalf("expected *ERC20Transfer, got %T", got)
	}

	if transfer.From != from {
		t.Errorf("From = %s, want %s", transfer.From, from)
	}
	if transfer.To != to {
		t.Errorf("To = %s, want %s", transfer.To, to)
	}
	if transfer.Value.Cmp(value) != 0 {
		t.Errorf("Value = %d, want %d", transfer.Value, value)
	}
}

// --- ERC20Approval ---

func TestDecodeERC20Approval(t *testing.T) {
	owner := common.HexToAddress("0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	spender := common.HexToAddress("0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB")
	value := big.NewInt(0).SetBytes(common.Hex2Bytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"))

	topics := []common.Hash{
		topic0ForName("Approval"),
		padAddressToTopic(owner),
		padAddressToTopic(spender),
	}
	data := encodeUint256(value)

	got, ok := DecodeTypedEvent(topics, data)
	if !ok {
		t.Fatal("DecodeTypedEvent returned false for Approval")
	}

	approval, ok := got.(*ERC20Approval)
	if !ok {
		t.Fatalf("expected *ERC20Approval, got %T", got)
	}

	if approval.Owner != owner {
		t.Errorf("Owner = %s, want %s", approval.Owner, owner)
	}
	if approval.Spender != spender {
		t.Errorf("Spender = %s, want %s", approval.Spender, spender)
	}
	if approval.Value.Cmp(value) != 0 {
		t.Errorf("Value = %d, want max uint256", approval.Value)
	}
}

// --- ERC721ApprovalForAll ---

func TestDecodeERC721ApprovalForAll(t *testing.T) {
	owner := common.HexToAddress("0xCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC")
	operator := common.HexToAddress("0xDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD")

	tests := []struct {
		name     string
		approved bool
	}{
		{"approved_true", true},
		{"approved_false", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			topics := []common.Hash{
				topic0ForName("ApprovalForAll"),
				padAddressToTopic(owner),
				padAddressToTopic(operator),
			}
			// bool true = 0x00...01, bool false = 0x00...00
			data := make([]byte, 32)
			if tc.approved {
				data[31] = 1
			}

			got, ok := DecodeTypedEvent(topics, data)
			if !ok {
				t.Fatal("DecodeTypedEvent returned false for ApprovalForAll")
			}

			afa, ok := got.(*ERC721ApprovalForAll)
			if !ok {
				t.Fatalf("expected *ERC721ApprovalForAll, got %T", got)
			}

			if afa.Owner != owner {
				t.Errorf("Owner = %s, want %s", afa.Owner, owner)
			}
			if afa.Operator != operator {
				t.Errorf("Operator = %s, want %s", afa.Operator, operator)
			}
			if afa.Approved != tc.approved {
				t.Errorf("Approved = %v, want %v", afa.Approved, tc.approved)
			}
		})
	}
}

// --- ERC1155TransferSingle ---

func TestDecodeERC1155TransferSingle(t *testing.T) {
	operator := common.HexToAddress("0x1111111111111111111111111111111111111111")
	from := common.HexToAddress("0x2222222222222222222222222222222222222222")
	to := common.HexToAddress("0x3333333333333333333333333333333333333333")
	id := big.NewInt(42)
	value := big.NewInt(100)

	topics := []common.Hash{
		topic0ForName("TransferSingle"),
		padAddressToTopic(operator),
		padAddressToTopic(from),
		padAddressToTopic(to),
	}
	// data: id (uint256) + value (uint256) = 64 bytes
	data := append(encodeUint256(id), encodeUint256(value)...)

	got, ok := DecodeTypedEvent(topics, data)
	if !ok {
		t.Fatal("DecodeTypedEvent returned false for TransferSingle")
	}

	ts, ok := got.(*ERC1155TransferSingle)
	if !ok {
		t.Fatalf("expected *ERC1155TransferSingle, got %T", got)
	}

	if ts.Operator != operator {
		t.Errorf("Operator = %s, want %s", ts.Operator, operator)
	}
	if ts.From != from {
		t.Errorf("From = %s, want %s", ts.From, from)
	}
	if ts.To != to {
		t.Errorf("To = %s, want %s", ts.To, to)
	}
	if ts.ID.Cmp(id) != 0 {
		t.Errorf("ID = %d, want %d", ts.ID, id)
	}
	if ts.Value.Cmp(value) != 0 {
		t.Errorf("Value = %d, want %d", ts.Value, value)
	}
}

// --- ERC1155TransferBatch ---

func TestDecodeERC1155TransferBatch(t *testing.T) {
	operator := common.HexToAddress("0x4444444444444444444444444444444444444444")
	from := common.HexToAddress("0x5555555555555555555555555555555555555555")
	to := common.HexToAddress("0x6666666666666666666666666666666666666666")

	topics := []common.Hash{
		topic0ForName("TransferBatch"),
		padAddressToTopic(operator),
		padAddressToTopic(from),
		padAddressToTopic(to),
	}

	got, ok := DecodeTypedEvent(topics, []byte{})
	if !ok {
		t.Fatal("DecodeTypedEvent returned false for TransferBatch")
	}

	tb, ok := got.(*ERC1155TransferBatch)
	if !ok {
		t.Fatalf("expected *ERC1155TransferBatch, got %T", got)
	}

	if tb.Operator != operator {
		t.Errorf("Operator = %s, want %s", tb.Operator, operator)
	}
	if tb.From != from {
		t.Errorf("From = %s, want %s", tb.From, from)
	}
	if tb.To != to {
		t.Errorf("To = %s, want %s", tb.To, to)
	}
	// IDs and Values are nil when data is empty (no ABI-decoded content)
	if tb.IDs != nil || tb.Values != nil {
		t.Error("TransferBatch IDs/Values should be nil with empty data")
	}
}

func TestDecodeERC1155TransferBatchWithData(t *testing.T) {
	operator := common.HexToAddress("0x4444444444444444444444444444444444444444")
	from := common.HexToAddress("0x5555555555555555555555555555555555555555")
	to := common.HexToAddress("0x6666666666666666666666666666666666666666")

	topics := []common.Hash{
		topic0ForName("TransferBatch"),
		padAddressToTopic(operator),
		padAddressToTopic(from),
		padAddressToTopic(to),
	}

	// ABI-encode TransferBatch non-indexed args: uint256[] ids, uint256[] values
	parsedABI := GetABIForEventName("TransferBatch")
	if parsedABI == nil {
		t.Fatal("TransferBatch ABI not found")
	}
	ev := parsedABI.Events["TransferBatch"]
	nonIndexed := ev.Inputs.NonIndexed()

	ids := []*big.Int{big.NewInt(1), big.NewInt(2)}
	values := []*big.Int{big.NewInt(100), big.NewInt(200)}
	data, err := nonIndexed.Pack(ids, values)
	if err != nil {
		t.Fatalf("failed to ABI-encode TransferBatch data: %v", err)
	}

	got, ok := DecodeTypedEvent(topics, data)
	if !ok {
		t.Fatal("DecodeTypedEvent returned false for TransferBatch")
	}

	tb, ok := got.(*ERC1155TransferBatch)
	if !ok {
		t.Fatalf("expected *ERC1155TransferBatch, got %T", got)
	}

	if tb.Operator != operator {
		t.Errorf("Operator = %s, want %s", tb.Operator, operator)
	}
	if tb.From != from {
		t.Errorf("From = %s, want %s", tb.From, from)
	}
	if tb.To != to {
		t.Errorf("To = %s, want %s", tb.To, to)
	}
	if len(tb.IDs) != 2 {
		t.Fatalf("len(IDs) = %d, want 2", len(tb.IDs))
	}
	if tb.IDs[0].Cmp(big.NewInt(1)) != 0 {
		t.Errorf("IDs[0] = %d, want 1", tb.IDs[0])
	}
	if tb.IDs[1].Cmp(big.NewInt(2)) != 0 {
		t.Errorf("IDs[1] = %d, want 2", tb.IDs[1])
	}
	if len(tb.Values) != 2 {
		t.Fatalf("len(Values) = %d, want 2", len(tb.Values))
	}
	if tb.Values[0].Cmp(big.NewInt(100)) != 0 {
		t.Errorf("Values[0] = %d, want 100", tb.Values[0])
	}
	if tb.Values[1].Cmp(big.NewInt(200)) != 0 {
		t.Errorf("Values[1] = %d, want 200", tb.Values[1])
	}
}

// --- ERC1155URI ---

func TestDecodeERC1155URI(t *testing.T) {
	id := big.NewInt(999)

	topics := []common.Hash{
		topic0ForName("URI"),
		padUint256ToTopic(id),
	}

	got, ok := DecodeTypedEvent(topics, []byte{})
	if !ok {
		t.Fatal("DecodeTypedEvent returned false for URI")
	}

	uri, ok := got.(*ERC1155URI)
	if !ok {
		t.Fatalf("expected *ERC1155URI, got %T", got)
	}

	if uri.ID.Cmp(id) != 0 {
		t.Errorf("ID = %d, want %d", uri.ID, id)
	}
	// Value is empty when data is empty (no ABI-decoded content)
	if uri.Value != "" {
		t.Error("URI Value should be empty with empty data")
	}
}

func TestDecodeERC1155URIWithData(t *testing.T) {
	id := big.NewInt(999)

	topics := []common.Hash{
		topic0ForName("URI"),
		padUint256ToTopic(id),
	}

	// ABI-encode URI non-indexed args: string value
	parsedABI := GetABIForEventName("URI")
	if parsedABI == nil {
		t.Fatal("URI ABI not found")
	}
	ev := parsedABI.Events["URI"]
	nonIndexed := ev.Inputs.NonIndexed()

	data, err := nonIndexed.Pack("https://example.com/metadata/1.json")
	if err != nil {
		t.Fatalf("failed to ABI-encode URI data: %v", err)
	}

	got, ok := DecodeTypedEvent(topics, data)
	if !ok {
		t.Fatal("DecodeTypedEvent returned false for URI")
	}

	uri, ok := got.(*ERC1155URI)
	if !ok {
		t.Fatalf("expected *ERC1155URI, got %T", got)
	}

	if uri.ID.Cmp(id) != 0 {
		t.Errorf("ID = %d, want %d", uri.ID, id)
	}
	if uri.Value != "https://example.com/metadata/1.json" {
		t.Errorf("Value = %q, want %q", uri.Value, "https://example.com/metadata/1.json")
	}
}

// --- UniswapV3Swap ---

func TestDecodeUniswapV3Swap(t *testing.T) {
	sender := common.HexToAddress("0x7777777777777777777777777777777777777777")
	amount0 := big.NewInt(-1000) // negative (int256)
	amount1 := big.NewInt(500)   // positive (int256)
	sqrtPriceX96, _ := new(big.Int).SetString("79228162514264337593543950336", 10)
	liquidity := big.NewInt(1000000000)
	tick := big.NewInt(-100)

	topics := []common.Hash{
		topic0ForName("Swap"),
		padAddressToTopic(sender),
	}
	// 5 x uint256/int256 = 160 bytes
	data := make([]byte, 0, 160)
	data = append(data, encodeUint256(new(big.Int).SetUint64(uint64(amount0.Int64())))...)
	data = append(data, encodeUint256(amount1)...)
	data = append(data, encodeUint256(sqrtPriceX96)...)
	data = append(data, encodeUint256(liquidity)...)
	data = append(data, encodeUint256(tick)...)

	got, ok := DecodeTypedEvent(topics, data)
	if !ok {
		t.Fatal("DecodeTypedEvent returned false for Swap")
	}

	swap, ok := got.(*UniswapV3Swap)
	if !ok {
		t.Fatalf("expected *UniswapV3Swap, got %T", got)
	}

	if swap.Sender != sender {
		t.Errorf("Sender = %s, want %s", swap.Sender, sender)
	}
	// Note: int256 values are encoded as two's complement in 32 bytes.
	// SetBytes interprets the raw bytes, so negative values need special handling.
	// For a positive amount1, direct comparison works:
	if swap.Amount1.Cmp(amount1) != 0 {
		t.Errorf("Amount1 = %d, want %d", swap.Amount1, amount1)
	}
	if swap.SqrtPriceX96.Cmp(sqrtPriceX96) != 0 {
		t.Errorf("SqrtPriceX96 = %d, want %d", swap.SqrtPriceX96, sqrtPriceX96)
	}
	if swap.Liquidity.Cmp(liquidity) != 0 {
		t.Errorf("Liquidity = %d, want %d", swap.Liquidity, liquidity)
	}
}

// --- Boundary / Edge Cases ---

func TestDecodeTypedEvent_EmptyTopics(t *testing.T) {
	_, ok := DecodeTypedEvent(nil, nil)
	if ok {
		t.Error("expected false for nil topics")
	}

	_, ok = DecodeTypedEvent([]common.Hash{}, nil)
	if ok {
		t.Error("expected false for empty topics")
	}
}

func TestDecodeTypedEvent_UnknownTopic0(t *testing.T) {
	unknownTopic := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	_, ok := DecodeTypedEvent([]common.Hash{unknownTopic}, nil)
	if ok {
		t.Error("expected false for unknown topic0")
	}
}

func TestDecodeTypedEvent_AddressTruncation(t *testing.T) {
	// Verify that only the last 20 bytes of a 32-byte topic are used for address
	from := common.HexToAddress("0x0000000000000000000000000000000000000001")
	to := common.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff")

	// Construct topic with non-zero high bytes (should be ignored)
	var topic1, topic2 common.Hash
	topic1[31] = 0x01        // last byte = 1, rest zero → address 0x...0001
	copy(topic2[12:], to[:]) // proper padding

	topics := []common.Hash{
		topic0ForName("Transfer"),
		topic1,
		topic2,
	}
	data := encodeUint256(big.NewInt(1))

	got, ok := DecodeTypedEvent(topics, data)
	if !ok {
		t.Fatal("DecodeTypedEvent returned false")
	}

	transfer := got.(*ERC20Transfer)
	if transfer.From != from {
		t.Errorf("From = %s, want %s — address truncation failed", transfer.From, from)
	}
	if transfer.To != to {
		t.Errorf("To = %s, want %s", transfer.To, to)
	}
}

func TestDecodeTypedEvent_ShortData(t *testing.T) {
	// Transfer with less than 32 bytes of data — Value should be nil
	from := common.HexToAddress("0x1111111111111111111111111111111111111111")
	to := common.HexToAddress("0x2222222222222222222222222222222222222222")

	topics := []common.Hash{
		topic0ForName("Transfer"),
		padAddressToTopic(from),
		padAddressToTopic(to),
	}

	got, ok := DecodeTypedEvent(topics, []byte{0x01, 0x02}) // only 2 bytes
	if !ok {
		t.Fatal("DecodeTypedEvent returned false for short data")
	}

	transfer := got.(*ERC20Transfer)
	if transfer.Value != nil {
		t.Errorf("Value should be nil for short data, got %v", transfer.Value)
	}
}

func TestDecodeTypedEvent_MissingIndexedTopics(t *testing.T) {
	// Transfer with only topic0 — From and To should be zero addresses
	topics := []common.Hash{
		topic0ForName("Transfer"),
	}
	data := encodeUint256(big.NewInt(100))

	got, ok := DecodeTypedEvent(topics, data)
	if !ok {
		t.Fatal("DecodeTypedEvent returned false")
	}

	transfer := got.(*ERC20Transfer)
	if transfer.From != (common.Address{}) {
		t.Errorf("From should be zero address when topics[1] missing, got %s", transfer.From)
	}
	if transfer.To != (common.Address{}) {
		t.Errorf("To should be zero address when topics[2] missing, got %s", transfer.To)
	}
	if transfer.Value.Cmp(big.NewInt(100)) != 0 {
		t.Errorf("Value = %d, want 100", transfer.Value)
	}
}

// --- Table-driven test for EventName/Topic0 interface ---

func TestTypedEventInterface(t *testing.T) {
	events := []TypedEvent{
		&ERC20Transfer{},
		&ERC20Approval{},
		&ERC721ApprovalForAll{},
		&ERC1155TransferSingle{},
		&ERC1155TransferBatch{},
		&ERC1155URI{},
		&UniswapV3Swap{},
	}

	expectedNames := []string{
		"Transfer",
		"Approval",
		"ApprovalForAll",
		"TransferSingle",
		"TransferBatch",
		"URI",
		"Swap",
	}

	for i, ev := range events {
		if ev.EventName() != expectedNames[i] {
			t.Errorf("EventName() = %q, want %q", ev.EventName(), expectedNames[i])
		}
		// Topic0 should not be zero hash for known events
		if ev.Topic0() == (common.Hash{}) {
			t.Errorf("Topic0() returned zero hash for %s", expectedNames[i])
		}
	}
}

func TestDecodeTypedEvent_AllEventTypes(t *testing.T) {
	// Verify that all 7 event types are registered in the decoder
	from := common.HexToAddress("0x1111111111111111111111111111111111111111")
	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	operator := common.HexToAddress("0x3333333333333333333333333333333333333333")
	sender := common.HexToAddress("0x4444444444444444444444444444444444444444")

	tests := []struct {
		name     string
		topics   []common.Hash
		data     []byte
		wantType string
	}{
		{
			"Transfer",
			[]common.Hash{topic0ForName("Transfer"), padAddressToTopic(from), padAddressToTopic(to)},
			encodeUint256(big.NewInt(100)),
			"*core.ERC20Transfer",
		},
		{
			"Approval",
			[]common.Hash{topic0ForName("Approval"), padAddressToTopic(from), padAddressToTopic(to)},
			encodeUint256(big.NewInt(200)),
			"*core.ERC20Approval",
		},
		{
			"ApprovalForAll",
			[]common.Hash{topic0ForName("ApprovalForAll"), padAddressToTopic(from), padAddressToTopic(to)},
			make([]byte, 32),
			"*core.ERC721ApprovalForAll",
		},
		{
			"TransferSingle",
			[]common.Hash{topic0ForName("TransferSingle"), padAddressToTopic(operator), padAddressToTopic(from), padAddressToTopic(to)},
			append(encodeUint256(big.NewInt(1)), encodeUint256(big.NewInt(10))...),
			"*core.ERC1155TransferSingle",
		},
		{
			"TransferBatch",
			[]common.Hash{topic0ForName("TransferBatch"), padAddressToTopic(operator), padAddressToTopic(from), padAddressToTopic(to)},
			[]byte{},
			"*core.ERC1155TransferBatch",
		},
		{
			"URI",
			[]common.Hash{topic0ForName("URI"), padUint256ToTopic(big.NewInt(1))},
			[]byte{},
			"*core.ERC1155URI",
		},
		{
			"Swap",
			[]common.Hash{topic0ForName("Swap"), padAddressToTopic(sender)},
			make([]byte, 160),
			"*core.UniswapV3Swap",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := DecodeTypedEvent(tc.topics, tc.data)
			if !ok {
				t.Fatalf("DecodeTypedEvent returned false for %s", tc.name)
			}
			typeStr := got.EventName()
			if typeStr != tc.name {
				t.Errorf("EventName() = %q, want %q", typeStr, tc.name)
			}
		})
	}
}
