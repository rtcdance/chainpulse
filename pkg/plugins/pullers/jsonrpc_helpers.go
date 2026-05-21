package pullers

import (
	"encoding/json"
	"fmt"
)

// JSONRPCError represents a JSON-RPC error.
// Retained for the WebSocket puller's subscription handling.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

// BlockHeader represents a blockchain block header.
// Retained for the WebSocket puller's newHeads subscription parsing.
type BlockHeader struct {
	Number       string   `json:"number"`
	Hash         string   `json:"hash"`
	ParentHash   string   `json:"parentHash"`
	Timestamp    string   `json:"timestamp"`
	Miner        string   `json:"miner"`
	Difficulty   string   `json:"difficulty"`
	GasLimit     string   `json:"gasLimit"`
	GasUsed      string   `json:"gasUsed"`
	Transactions []string `json:"transactions"`
}

// Log represents a blockchain log/event.
// Retained for the WebSocket puller's subscription notification parsing.
type Log struct {
	Address     string   `json:"address"`
	Topics      []string `json:"topics"`
	Data        string   `json:"data"`
	BlockNumber string   `json:"blockNumber"`
	BlockHash   string   `json:"blockHash"`
	TxHash      string   `json:"transactionHash"`
	TxIndex     string   `json:"transactionIndex"`
	LogIndex    string   `json:"logIndex"`
	Removed     bool     `json:"removed"`
}

// hexToUint64 converts a hex string to uint64.
func hexToUint64(hexStr string) uint64 {
	if len(hexStr) < 2 {
		return 0
	}

	if hexStr[:2] == "0x" {
		hexStr = hexStr[2:]
	}

	var result uint64
	if _, err := fmt.Sscanf(hexStr, "%x", &result); err != nil {
		return 0
	}
	return result
}

// uint64ToHex converts uint64 to hex string.
func uint64ToHex(num uint64) string {
	return fmt.Sprintf("0x%x", num)
}

// JSONRPCRequest represents a JSON-RPC request.
// Retained for the WebSocket puller's request/response handling.
type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
	ID      int64  `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC response.
// Retained for the WebSocket puller's request/response handling.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *JSONRPCError   `json:"error"`
	ID      int64           `json:"id"`
}
