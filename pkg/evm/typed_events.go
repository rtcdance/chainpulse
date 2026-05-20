package evm

import (
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

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

// ERC20Approval represents an ERC-20 Approval event
// Approval(address indexed owner, address indexed spender, uint256 value)
type ERC20Approval struct {
	Owner   common.Address `json:"owner"`
	Spender common.Address `json:"spender"`
	Value   *big.Int       `json:"value"`
}

func (e *ERC20Approval) EventName() string   { return "Approval" }
func (e *ERC20Approval) Topic0() common.Hash { return topic0ForName("Approval") }

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

// ERC1155URI represents an ERC-1155 URI event
// URI(string value, uint256 indexed id)
type ERC1155URI struct {
	Value string   `json:"value"`
	ID    *big.Int `json:"id"`
}

func (e *ERC1155URI) EventName() string   { return "URI" }
func (e *ERC1155URI) Topic0() common.Hash { return topic0ForName("URI") }

// UniswapV3Swap represents a Uniswap V3 Swap event
// Swap(address indexed sender, int256 amount0, int256 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick)
type UniswapV3Swap struct {
	Sender       common.Address `json:"sender"`
	Amount0      *big.Int       `json:"amount0"`
	Amount1      *big.Int       `json:"amount1"`
	SqrtPriceX96 *big.Int       `json:"sqrtPriceX96"`
	Liquidity    *big.Int       `json:"liquidity"`
	Tick         *big.Int       `json:"tick"`
}

func (e *UniswapV3Swap) EventName() string   { return "Swap" }
func (e *UniswapV3Swap) Topic0() common.Hash { return topic0ForName("Swap") }

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
type typedDecoderFunc func(topics []common.Hash, data []byte) TypedEvent

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

	result := decoder(topics, data)
	if result == nil {
		return nil, false
	}
	return result, true
}

// --- Decoder implementations ---

func decodeERC20Transfer(topics []common.Hash, data []byte) TypedEvent {
	e := &ERC20Transfer{}
	if len(topics) > 1 {
		e.From = common.BytesToAddress(topics[1].Bytes()[12:])
	}
	if len(topics) > 2 {
		e.To = common.BytesToAddress(topics[2].Bytes()[12:])
	}
	if len(data) >= 32 {
		e.Value = new(big.Int).SetBytes(data[:32])
	}
	return e
}

func decodeERC20Approval(topics []common.Hash, data []byte) TypedEvent {
	e := &ERC20Approval{}
	if len(topics) > 1 {
		e.Owner = common.BytesToAddress(topics[1].Bytes()[12:])
	}
	if len(topics) > 2 {
		e.Spender = common.BytesToAddress(topics[2].Bytes()[12:])
	}
	if len(data) >= 32 {
		e.Value = new(big.Int).SetBytes(data[:32])
	}
	return e
}

func decodeERC721ApprovalForAll(topics []common.Hash, data []byte) TypedEvent {
	e := &ERC721ApprovalForAll{}
	if len(topics) > 1 {
		e.Owner = common.BytesToAddress(topics[1].Bytes()[12:])
	}
	if len(topics) > 2 {
		e.Operator = common.BytesToAddress(topics[2].Bytes()[12:])
	}
	if len(data) >= 32 {
		e.Approved = data[31] != 0
	}
	return e
}

func decodeERC1155TransferSingle(topics []common.Hash, data []byte) TypedEvent {
	e := &ERC1155TransferSingle{}
	if len(topics) > 1 {
		e.Operator = common.BytesToAddress(topics[1].Bytes()[12:])
	}
	if len(topics) > 2 {
		e.From = common.BytesToAddress(topics[2].Bytes()[12:])
	}
	if len(topics) > 3 {
		e.To = common.BytesToAddress(topics[3].Bytes()[12:])
	}
	if len(data) >= 64 {
		e.ID = new(big.Int).SetBytes(data[:32])
		e.Value = new(big.Int).SetBytes(data[32:64])
	}
	return e
}

func decodeERC1155TransferBatch(topics []common.Hash, data []byte) TypedEvent {
	e := &ERC1155TransferBatch{}
	if len(topics) > 1 {
		e.Operator = common.BytesToAddress(topics[1].Bytes()[12:])
	}
	if len(topics) > 2 {
		e.From = common.BytesToAddress(topics[2].Bytes()[12:])
	}
	if len(topics) > 3 {
		e.To = common.BytesToAddress(topics[3].Bytes()[12:])
	}
	// ABI-decode dynamic arrays ids[] and values[] from data
	if len(data) > 0 {
		parsedABI := GetABIForEventName("TransferBatch")
		if parsedABI != nil {
			if ev, ok := parsedABI.Events["TransferBatch"]; ok {
				nonIndexed := ev.Inputs.NonIndexed()
				decoded, err := nonIndexed.Unpack(data)
				if err == nil && len(decoded) >= 2 {
					if ids, ok := decoded[0].([]*big.Int); ok {
						e.IDs = ids
					}
					if values, ok := decoded[1].([]*big.Int); ok {
						e.Values = values
					}
				}
			}
		}
	}
	return e
}

func decodeERC1155URI(topics []common.Hash, data []byte) TypedEvent {
	e := &ERC1155URI{}
	if len(topics) > 1 {
		e.ID = new(big.Int).SetBytes(topics[1].Bytes())
	}
	// ABI-decode the dynamic string from data
	if len(data) > 0 {
		parsedABI := GetABIForEventName("URI")
		if parsedABI != nil {
			if ev, ok := parsedABI.Events["URI"]; ok {
				nonIndexed := ev.Inputs.NonIndexed()
				decoded, err := nonIndexed.Unpack(data)
				if err == nil && len(decoded) >= 1 {
					if s, ok := decoded[0].(string); ok {
						e.Value = s
					}
				}
			}
		}
	}
	return e
}

func decodeUniswapV3Swap(topics []common.Hash, data []byte) TypedEvent {
	e := &UniswapV3Swap{}
	if len(topics) > 1 {
		e.Sender = common.BytesToAddress(topics[1].Bytes()[12:])
	}
	if len(data) >= 160 {
		e.Amount0 = new(big.Int).SetBytes(data[:32])
		e.Amount1 = new(big.Int).SetBytes(data[32:64])
		e.SqrtPriceX96 = new(big.Int).SetBytes(data[64:96])
		e.Liquidity = new(big.Int).SetBytes(data[96:128])
		e.Tick = new(big.Int).SetBytes(data[128:160])
	}
	return e
}
