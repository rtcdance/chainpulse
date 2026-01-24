package decoder

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventDecoder(t *testing.T) {
	logger := &MockLogger{}
	contractManager := NewContractManager(logger)
	decoder := NewEventDecoder(contractManager, logger)

	assert.NotNil(t, decoder)
	assert.Equal(t, contractManager, decoder.contractManager)
}

func TestDecodeEventNil(t *testing.T) {
	logger := &MockLogger{}
	contractManager := NewContractManager(logger)
	decoder := NewEventDecoder(contractManager, logger)

	abiJSON := []byte(`[]`)
	var contractABI abi.ABI
	_ = json.Unmarshal(abiJSON, &contractABI)

	_, err := decoder.DecodeEvent(nil, contractABI)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "raw event is nil")
}

func TestDecodeEventNoTopics(t *testing.T) {
	logger := &MockLogger{}
	contractManager := NewContractManager(logger)
	decoder := NewEventDecoder(contractManager, logger)

	abiJSON := []byte(`[]`)
	var contractABI abi.ABI
	_ = json.Unmarshal(abiJSON, &contractABI)

	rawEvent := &types.Log{
		Topics: []common.Hash{},
	}

	_, err := decoder.DecodeEvent(rawEvent, contractABI)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no topics")
}

func TestDecodeEventSignatureNotFound(t *testing.T) {
	logger := &MockLogger{}
	contractManager := NewContractManager(logger)
	decoder := NewEventDecoder(contractManager, logger)

	abiJSON := []byte(`[
		{
			"type": "event",
			"name": "Transfer",
			"inputs": []
		}
	]`)
	var contractABI abi.ABI
	_ = json.Unmarshal(abiJSON, &contractABI)

	rawEvent := &types.Log{
		Topics: []common.Hash{
			common.HexToHash("0x9999"), // Non-existent signature
		},
	}

	_, err := decoder.DecodeEvent(rawEvent, contractABI)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event signature not found")
}

func TestDecodeEventBatch(t *testing.T) {
	logger := &MockLogger{}
	contractManager := NewContractManager(logger)
	decoder := NewEventDecoder(contractManager, logger)

	abiJSON := []byte(`[]`)
	var contractABI abi.ABI
	_ = json.Unmarshal(abiJSON, &contractABI)

	rawEvents := []*types.Log{}

	decoded, err := decoder.DecodeEventBatch(rawEvents, contractABI)
	require.NoError(t, err)

	assert.Equal(t, 0, len(decoded))
}

func TestDecodeEventBatchEmpty(t *testing.T) {
	logger := &MockLogger{}
	contractManager := NewContractManager(logger)
	decoder := NewEventDecoder(contractManager, logger)

	abiJSON := []byte(`[]`)
	var contractABI abi.ABI
	_ = json.Unmarshal(abiJSON, &contractABI)

	decoded, err := decoder.DecodeEventBatch([]*types.Log{}, contractABI)
	require.NoError(t, err)

	assert.Equal(t, 0, len(decoded))
}

func TestDecodeEventWithABI(t *testing.T) {
	logger := &MockLogger{}
	contractManager := NewContractManager(logger)
	decoder := NewEventDecoder(contractManager, logger)

	abiJSON := []byte(`[
		{
			"type": "event",
			"name": "Transfer",
			"inputs": []
		}
	]`)

	_ = contractManager.LoadContractABI("ERC20", abiJSON)

	// Create a mock event with the correct signature
	transferEvent, _ := contractManager.GetEvent("ERC20", "Transfer")

	rawEvent := &types.Log{
		Topics: []common.Hash{transferEvent.ID},
	}

	decoded, err := decoder.DecodeEventWithABI("ERC20", rawEvent)
	require.NoError(t, err)

	assert.NotNil(t, decoded)
	assert.Equal(t, "Transfer", decoded.EventName)
}

func TestDecodeEventWithABINotFound(t *testing.T) {
	logger := &MockLogger{}
	contractManager := NewContractManager(logger)
	decoder := NewEventDecoder(contractManager, logger)

	rawEvent := &types.Log{
		Topics: []common.Hash{common.HexToHash("0x1234")},
	}

	_, err := decoder.DecodeEventWithABI("NonExistent", rawEvent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get ABI")
}

func TestDecodeEventBatchWithABI(t *testing.T) {
	logger := &MockLogger{}
	contractManager := NewContractManager(logger)
	decoder := NewEventDecoder(contractManager, logger)

	abiJSON := []byte(`[
		{
			"type": "event",
			"name": "Transfer",
			"inputs": []
		}
	]`)

	_ = contractManager.LoadContractABI("ERC20", abiJSON)

	rawEvents := []*types.Log{}

	decoded, err := decoder.DecodeEventBatchWithABI("ERC20", rawEvents)
	require.NoError(t, err)

	assert.Equal(t, 0, len(decoded))
}

func TestEventDecoderGetEventSignature(t *testing.T) {
	logger := &MockLogger{}
	contractManager := NewContractManager(logger)
	decoder := NewEventDecoder(contractManager, logger)

	abiJSON := []byte(`[
		{
			"type": "event",
			"name": "Transfer",
			"inputs": []
		}
	]`)

	_ = contractManager.LoadContractABI("ERC20", abiJSON)

	sig, err := decoder.GetEventSignature("ERC20", "Transfer")
	require.NoError(t, err)

	assert.NotNil(t, sig)
	assert.NotEqual(t, [32]byte{}, sig)
}

func TestEventDecoderGetEventSignatureNotFound(t *testing.T) {
	logger := &MockLogger{}
	contractManager := NewContractManager(logger)
	decoder := NewEventDecoder(contractManager, logger)

	abiJSON := []byte(`[
		{
			"type": "event",
			"name": "Transfer",
			"inputs": []
		}
	]`)

	_ = contractManager.LoadContractABI("ERC20", abiJSON)

	_, err := decoder.GetEventSignature("ERC20", "NonExistent")
	assert.Error(t, err)
}

func TestEventDecoderGetEventSignatures(t *testing.T) {
	logger := &MockLogger{}
	contractManager := NewContractManager(logger)
	decoder := NewEventDecoder(contractManager, logger)

	abiJSON := []byte(`[
		{
			"type": "event",
			"name": "Transfer",
			"inputs": []
		},
		{
			"type": "event",
			"name": "Approval",
			"inputs": []
		}
	]`)

	_ = contractManager.LoadContractABI("ERC20", abiJSON)

	sigs, err := decoder.GetEventSignatures("ERC20")
	require.NoError(t, err)

	assert.Equal(t, 2, len(sigs))
	assert.Contains(t, sigs, "Transfer")
	assert.Contains(t, sigs, "Approval")
}

func TestDecodedEventStructure(t *testing.T) {
	decoded := &DecodedEvent{
		EventName:  "Transfer",
		Parameters: make(map[string]interface{}),
		Indexed:    make(map[string]interface{}),
		NonIndexed: make(map[string]interface{}),
	}

	decoded.Parameters["from"] = common.HexToAddress("0x1111")
	decoded.Parameters["to"] = common.HexToAddress("0x2222")
	decoded.Parameters["value"] = big.NewInt(1000)

	assert.Equal(t, "Transfer", decoded.EventName)
	assert.Equal(t, 3, len(decoded.Parameters))
}

func TestDecodeEventWithIndexedParameters(t *testing.T) {
	logger := &MockLogger{}
	contractManager := NewContractManager(logger)
	decoder := NewEventDecoder(contractManager, logger)

	abiJSON := []byte(`[
		{
			"type": "event",
			"name": "Transfer",
			"inputs": [
				{"name": "from", "type": "address", "indexed": true},
				{"name": "to", "type": "address", "indexed": true},
				{"name": "value", "type": "uint256", "indexed": false}
			]
		}
	]`)

	_ = contractManager.LoadContractABI("ERC20", abiJSON)

	transferEvent, _ := contractManager.GetEvent("ERC20", "Transfer")

	rawEvent := &types.Log{
		Topics: []common.Hash{
			transferEvent.ID,
			common.HexToHash("0x1111"), // from
			common.HexToHash("0x2222"), // to
		},
		Data: []byte{}, // No non-indexed data
	}

	decoded, err := decoder.DecodeEventWithABI("ERC20", rawEvent)
	require.NoError(t, err)

	assert.NotNil(t, decoded)
	assert.Equal(t, "Transfer", decoded.EventName)
	assert.Equal(t, 2, len(decoded.Indexed))
}

func TestEventDecoderConcurrency(t *testing.T) {
	logger := &MockLogger{}
	contractManager := NewContractManager(logger)
	decoder := NewEventDecoder(contractManager, logger)

	abiJSON := []byte(`[
		{
			"type": "event",
			"name": "Transfer",
			"inputs": []
		}
	]`)

	_ = contractManager.LoadContractABI("ERC20", abiJSON)

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			_, _ = decoder.GetEventSignatures("ERC20")
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestDecodeEventWithNonIndexedData(t *testing.T) {
	logger := &MockLogger{}
	contractManager := NewContractManager(logger)
	decoder := NewEventDecoder(contractManager, logger)

	abiJSON := []byte(`[
		{
			"type": "event",
			"name": "Transfer",
			"inputs": [
				{"name": "from", "type": "address", "indexed": true},
				{"name": "to", "type": "address", "indexed": true},
				{"name": "value", "type": "uint256", "indexed": false}
			]
		}
	]`)

	_ = contractManager.LoadContractABI("ERC20", abiJSON)

	transferEvent, _ := contractManager.GetEvent("ERC20", "Transfer")

	// Create data for non-indexed parameter (value)
	valueBytes := make([]byte, 32)
	copy(valueBytes, big.NewInt(1000).Bytes())

	rawEvent := &types.Log{
		Topics: []common.Hash{
			transferEvent.ID,
			common.HexToHash("0x1111"), // from
			common.HexToHash("0x2222"), // to
		},
		Data: valueBytes,
	}

	decoded, err := decoder.DecodeEventWithABI("ERC20", rawEvent)
	require.NoError(t, err)

	assert.NotNil(t, decoded)
	assert.Equal(t, 2, len(decoded.Indexed))
}
