package evm

import (
	"encoding/json"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// jsonBigInt is the JSON form of a *big.Int used by the typed events below.
//
// Why a string and not a number: encoding/json writes *big.Int as a bare decimal
// literal, so a uint256 travels as a 78-digit NUMBER. Measured, for
// 2^256-1 the wire carries
//
//	"value":115792089237316195423570985008687907853269984665640564039457584007913129639935
//
// and any JSON consumer that parses numbers as doubles — every JavaScript
// runtime, and jq — reads that back as
// 115792089237316200000000000000000000000000000000000000000000000000000000000000,
// i.e. the low digits are silently gone. The delivery plan's §4.1 requires
// amounts and large integers to travel as JSON strings for exactly this reason.
//
// A nil value still marshals to null, so the existing shape is preserved.
type jsonBigInt struct {
	value *big.Int
}

// MarshalJSON implements json.Marshaler.
func (j jsonBigInt) MarshalJSON() ([]byte, error) {
	if j.value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(j.value.String())
}

// jsonBigInts converts a slice for the array-valued fields.
func jsonBigInts(values []*big.Int) []jsonBigInt {
	out := make([]jsonBigInt, len(values))
	for i, v := range values {
		out[i] = jsonBigInt{value: v}
	}
	return out
}

// bigIntJSON renders a single value for the shadow structs below.
func bigIntJSON(value *big.Int) jsonBigInt {
	return jsonBigInt{value: value}
}

// TypedEvent represents a type-safe decoded blockchain event.
// Each known event type has its own Go struct with properly typed fields
// (common.Address, *big.Int, etc.) instead of map[string]interface{}.
type TypedEvent interface {
	// EventName returns the canonical event name (e.g., "Transfer", "Approval")
	EventName() string
	// Topic0 returns the keccak256 event signature hash
	Topic0() common.Hash
}

// ERC20Transfer represents an ERC-20/ERC-721 Transfer event
// Transfer(address indexed from, address indexed to, uint256 value)
type ERC20Transfer struct {
	From  common.Address `json:"from"`
	To    common.Address `json:"to"`
	Value *big.Int       `json:"value"`
}

func (e *ERC20Transfer) EventName() string   { return "Transfer" }
func (e *ERC20Transfer) Topic0() common.Hash { return topic0ForName("Transfer") }

// MarshalJSON emits Value as a decimal string. See jsonBigInt.
func (e *ERC20Transfer) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		From  common.Address `json:"from"`
		To    common.Address `json:"to"`
		Value jsonBigInt     `json:"value"`
	}{e.From, e.To, bigIntJSON(e.Value)})
}

// ERC20Approval represents an ERC-20 Approval event
// Approval(address indexed owner, address indexed spender, uint256 value)
type ERC20Approval struct {
	Owner   common.Address `json:"owner"`
	Spender common.Address `json:"spender"`
	Value   *big.Int       `json:"value"`
}

func (e *ERC20Approval) EventName() string   { return "Approval" }
func (e *ERC20Approval) Topic0() common.Hash { return topic0ForName("Approval") }

// MarshalJSON emits Value as a decimal string. See jsonBigInt.
func (e *ERC20Approval) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Owner   common.Address `json:"owner"`
		Spender common.Address `json:"spender"`
		Value   jsonBigInt     `json:"value"`
	}{e.Owner, e.Spender, bigIntJSON(e.Value)})
}

// ERC721ApprovalForAll represents an ERC-721 ApprovalForAll event
// ApprovalForAll(address indexed owner, address indexed operator, bool approved)
type ERC721ApprovalForAll struct {
	Owner    common.Address `json:"owner"`
	Operator common.Address `json:"operator"`
	Approved bool           `json:"approved"`
}

func (e *ERC721ApprovalForAll) EventName() string   { return "ApprovalForAll" }
func (e *ERC721ApprovalForAll) Topic0() common.Hash { return topic0ForName("ApprovalForAll") }

// ERC1155TransferSingle represents an ERC-1155 TransferSingle event
// TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value)
type ERC1155TransferSingle struct {
	Operator common.Address `json:"operator"`
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	ID       *big.Int       `json:"id"`
	Value    *big.Int       `json:"value"`
}

func (e *ERC1155TransferSingle) EventName() string   { return "TransferSingle" }
func (e *ERC1155TransferSingle) Topic0() common.Hash { return topic0ForName("TransferSingle") }

// MarshalJSON emits ID and Value as decimal strings. See jsonBigInt.
func (e *ERC1155TransferSingle) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Operator common.Address `json:"operator"`
		From     common.Address `json:"from"`
		To       common.Address `json:"to"`
		ID       jsonBigInt     `json:"id"`
		Value    jsonBigInt     `json:"value"`
	}{e.Operator, e.From, e.To, bigIntJSON(e.ID), bigIntJSON(e.Value)})
}

// ERC1155TransferBatch represents an ERC-1155 TransferBatch event
// TransferBatch(address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values)
type ERC1155TransferBatch struct {
	Operator common.Address `json:"operator"`
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	IDs      []*big.Int     `json:"ids"`
	Values   []*big.Int     `json:"values"`
}

func (e *ERC1155TransferBatch) EventName() string   { return "TransferBatch" }
func (e *ERC1155TransferBatch) Topic0() common.Hash { return topic0ForName("TransferBatch") }

// MarshalJSON emits IDs and Values as arrays of decimal strings. See jsonBigInt.
func (e *ERC1155TransferBatch) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Operator common.Address `json:"operator"`
		From     common.Address `json:"from"`
		To       common.Address `json:"to"`
		IDs      []jsonBigInt   `json:"ids"`
		Values   []jsonBigInt   `json:"values"`
	}{e.Operator, e.From, e.To, jsonBigInts(e.IDs), jsonBigInts(e.Values)})
}

// ERC1155URI represents an ERC-1155 URI event
// URI(string value, uint256 indexed id)
type ERC1155URI struct {
	Value string   `json:"value"`
	ID    *big.Int `json:"id"`
}

func (e *ERC1155URI) EventName() string   { return "URI" }
func (e *ERC1155URI) Topic0() common.Hash { return topic0ForName("URI") }

// MarshalJSON emits ID as a decimal string. See jsonBigInt.
func (e *ERC1155URI) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Value string     `json:"value"`
		ID    jsonBigInt `json:"id"`
	}{e.Value, bigIntJSON(e.ID)})
}

// UniswapV3Swap represents a Uniswap V3 Swap event.
//
// The real event has TWO indexed parameters, which is what makes its topic0
// 0xc42079f9…:
//
//	Swap(address indexed sender, address indexed recipient, int256 amount0,
//	     int256 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick)
type UniswapV3Swap struct {
	Sender       common.Address `json:"sender"`
	Recipient    common.Address `json:"recipient"`
	Amount0      *big.Int       `json:"amount0"`
	Amount1      *big.Int       `json:"amount1"`
	SqrtPriceX96 *big.Int       `json:"sqrtPriceX96"`
	Liquidity    *big.Int       `json:"liquidity"`
	Tick         *big.Int       `json:"tick"`
}

func (e *UniswapV3Swap) EventName() string   { return "Swap" }
func (e *UniswapV3Swap) Topic0() common.Hash { return topic0ForName("Swap") }

// MarshalJSON emits every numeric field as a decimal string. See jsonBigInt:
// amount0/amount1 are int256, so they can be negative and must not be parsed as
// doubles either.
func (e *UniswapV3Swap) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Sender       common.Address `json:"sender"`
		Recipient    common.Address `json:"recipient"`
		Amount0      jsonBigInt     `json:"amount0"`
		Amount1      jsonBigInt     `json:"amount1"`
		SqrtPriceX96 jsonBigInt     `json:"sqrtPriceX96"`
		Liquidity    jsonBigInt     `json:"liquidity"`
		Tick         jsonBigInt     `json:"tick"`
	}{
		e.Sender, e.Recipient,
		bigIntJSON(e.Amount0), bigIntJSON(e.Amount1),
		bigIntJSON(e.SqrtPriceX96), bigIntJSON(e.Liquidity), bigIntJSON(e.Tick),
	})
}

// topic0ForName looks up the topic0 hash for a known event name.
func topic0ForName(name string) common.Hash {
	abi := GetABIForEventName(name)
	if abi == nil {
		return common.Hash{}
	}
	if event, ok := abi.Events[name]; ok {
		return event.ID
	}
	return common.Hash{}
}

// typedDecoderRegistry maps topic0 hashes to typed decoder functions.
// Each decoder takes (topics, data) and returns a TypedEvent.
// A decoder returns (event, true) only when the input actually satisfies the
// event's shape. Returning a partially-filled event for a truncated log would
// make a malformed log indistinguishable from a real one whose fields happen
// to be zero — so the contract is (nil, false) instead (raw-only fallback).
type typedDecoderFunc func(topics []common.Hash, data []byte) (TypedEvent, bool)

var (
	typedDecoders     map[common.Hash]typedDecoderFunc
	typedDecodersOnce sync.Once
)

// initTypedDecoders lazily builds the topic0 → decoder function map.
func initTypedDecoders() {
	typedDecoders = make(map[common.Hash]typedDecoderFunc)

	// ERC-20 Transfer
	if abi := GetABIForEventName("Transfer"); abi != nil {
		if ev, ok := abi.Events["Transfer"]; ok {
			typedDecoders[ev.ID] = decodeERC20Transfer
		}
	}
	// ERC-20 Approval
	if abi := GetABIForEventName("Approval"); abi != nil {
		if ev, ok := abi.Events["Approval"]; ok {
			typedDecoders[ev.ID] = decodeERC20Approval
		}
	}
	// ERC-721 ApprovalForAll
	if abi := GetABIForEventName("ApprovalForAll"); abi != nil {
		if ev, ok := abi.Events["ApprovalForAll"]; ok {
			typedDecoders[ev.ID] = decodeERC721ApprovalForAll
		}
	}
	// ERC-1155 TransferSingle
	if abi := GetABIForEventName("TransferSingle"); abi != nil {
		if ev, ok := abi.Events["TransferSingle"]; ok {
			typedDecoders[ev.ID] = decodeERC1155TransferSingle
		}
	}
	// ERC-1155 TransferBatch
	if abi := GetABIForEventName("TransferBatch"); abi != nil {
		if ev, ok := abi.Events["TransferBatch"]; ok {
			typedDecoders[ev.ID] = decodeERC1155TransferBatch
		}
	}
	// ERC-1155 URI
	if abi := GetABIForEventName("URI"); abi != nil {
		if ev, ok := abi.Events["URI"]; ok {
			typedDecoders[ev.ID] = decodeERC1155URI
		}
	}
	// Uniswap V3 Swap
	if abi := GetABIForEventName("Swap"); abi != nil {
		if ev, ok := abi.Events["Swap"]; ok {
			typedDecoders[ev.ID] = decodeUniswapV3Swap
		}
	}
}

// DecodeTypedEvent attempts to decode topics+data into a typed event struct.
// Returns (TypedEvent, true) if a type-specific decoder matched, or (nil, false).
func DecodeTypedEvent(topics []common.Hash, data []byte) (TypedEvent, bool) {
	typedDecodersOnce.Do(initTypedDecoders)

	if len(topics) == 0 {
		return nil, false
	}

	decoder, ok := typedDecoders[topics[0]]
	if !ok {
		return nil, false
	}

	return decoder(topics, data)
}

// --- Decoder implementations ---

func decodeERC20Transfer(topics []common.Hash, data []byte) (TypedEvent, bool) {
	// Transfer(address indexed,address indexed,uint256): 3 topics, one word.
	if len(topics) < 3 || len(data) < 32 {
		return nil, false
	}
	return &ERC20Transfer{
		From:  common.BytesToAddress(topics[1].Bytes()[12:]),
		To:    common.BytesToAddress(topics[2].Bytes()[12:]),
		Value: new(big.Int).SetBytes(data[:32]),
	}, true
}

func decodeERC20Approval(topics []common.Hash, data []byte) (TypedEvent, bool) {
	// Approval(address indexed,address indexed,uint256): 3 topics, one word.
	if len(topics) < 3 || len(data) < 32 {
		return nil, false
	}
	return &ERC20Approval{
		Owner:   common.BytesToAddress(topics[1].Bytes()[12:]),
		Spender: common.BytesToAddress(topics[2].Bytes()[12:]),
		Value:   new(big.Int).SetBytes(data[:32]),
	}, true
}

func decodeERC721ApprovalForAll(topics []common.Hash, data []byte) (TypedEvent, bool) {
	// ApprovalForAll(address indexed,address indexed,bool): 3 topics, one word.
	if len(topics) < 3 || len(data) < 32 {
		return nil, false
	}
	return &ERC721ApprovalForAll{
		Owner:    common.BytesToAddress(topics[1].Bytes()[12:]),
		Operator: common.BytesToAddress(topics[2].Bytes()[12:]),
		Approved: data[31] != 0,
	}, true
}

func decodeERC1155TransferSingle(topics []common.Hash, data []byte) (TypedEvent, bool) {
	// TransferSingle(address indexed x3, uint256, uint256): 4 topics, two words.
	if len(topics) < 4 || len(data) < 64 {
		return nil, false
	}
	return &ERC1155TransferSingle{
		Operator: common.BytesToAddress(topics[1].Bytes()[12:]),
		From:     common.BytesToAddress(topics[2].Bytes()[12:]),
		To:       common.BytesToAddress(topics[3].Bytes()[12:]),
		ID:       new(big.Int).SetBytes(data[:32]),
		Value:    new(big.Int).SetBytes(data[32:64]),
	}, true
}

func decodeERC1155TransferBatch(topics []common.Hash, data []byte) (TypedEvent, bool) {
	// TransferBatch(address indexed x3, uint256[], uint256[]): 4 topics, and a
	// payload whose two dynamic arrays must both unpack. If they do not, the log
	// is not decodable and must fall through to raw-only rather than surface as
	// a typed event with empty slices.
	if len(topics) < 4 {
		return nil, false
	}

	parsedABI := GetABIForEventName("TransferBatch")
	if parsedABI == nil {
		return nil, false
	}
	ev, ok := parsedABI.Events["TransferBatch"]
	if !ok {
		return nil, false
	}

	decoded, err := ev.Inputs.NonIndexed().Unpack(data)
	if err != nil || len(decoded) < 2 {
		return nil, false
	}
	ids, okIDs := decoded[0].([]*big.Int)
	values, okValues := decoded[1].([]*big.Int)
	if !okIDs || !okValues {
		return nil, false
	}

	return &ERC1155TransferBatch{
		Operator: common.BytesToAddress(topics[1].Bytes()[12:]),
		From:     common.BytesToAddress(topics[2].Bytes()[12:]),
		To:       common.BytesToAddress(topics[3].Bytes()[12:]),
		IDs:      ids,
		Values:   values,
	}, true
}

func decodeERC1155URI(topics []common.Hash, data []byte) (TypedEvent, bool) {
	// URI(uint256 indexed, string): 2 topics, and the dynamic string must unpack.
	if len(topics) < 2 {
		return nil, false
	}

	parsedABI := GetABIForEventName("URI")
	if parsedABI == nil {
		return nil, false
	}
	ev, ok := parsedABI.Events["URI"]
	if !ok {
		return nil, false
	}

	decoded, err := ev.Inputs.NonIndexed().Unpack(data)
	if err != nil || len(decoded) < 1 {
		return nil, false
	}
	value, ok := decoded[0].(string)
	if !ok {
		return nil, false
	}

	return &ERC1155URI{
		ID:    new(big.Int).SetBytes(topics[1].Bytes()),
		Value: value,
	}, true
}

func decodeUniswapV3Swap(topics []common.Hash, data []byte) (TypedEvent, bool) {
	// Swap(address indexed sender, address indexed recipient, int256, int256,
	// uint160, uint128, int24): 3 topics and five words. A truncated log must
	// not surface as a Swap with zeroed amounts.
	if len(topics) < 3 || len(data) < 160 {
		return nil, false
	}
	return &UniswapV3Swap{
		Sender:       common.BytesToAddress(topics[1].Bytes()[12:]),
		Recipient:    common.BytesToAddress(topics[2].Bytes()[12:]),
		Amount0:      decodeSignedBigInt(data[:32]),       // int256
		Amount1:      decodeSignedBigInt(data[32:64]),     // int256
		SqrtPriceX96: new(big.Int).SetBytes(data[64:96]),  // uint160
		Liquidity:    new(big.Int).SetBytes(data[96:128]), // uint128
		Tick:         decodeSignedBigInt(data[128:160]),   // int24 (sign-extended)
	}, true
}

// decodeSignedBigInt decodes a big-endian two's-complement ABI word as a signed
// integer. ABI signed types (int256, int24, …) are sign-extended to a full
// 32-byte word, so a set high bit in the first byte marks a negative value.
//
// Using new(big.Int).SetBytes here would silently turn -1 into 2^256-1.
func decodeSignedBigInt(word []byte) *big.Int {
	v := new(big.Int).SetBytes(word)
	if len(word) > 0 && word[0]&0x80 != 0 {
		v.Sub(v, new(big.Int).Lsh(big.NewInt(1), uint(8*len(word))))
	}
	return v
}
