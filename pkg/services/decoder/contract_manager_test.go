package decoder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chainpulse/pkg/core"
)

// MockLogger implements core.Logger for testing
type MockLogger struct{}

func (ml *MockLogger) Debug(msg string, args ...interface{}) {}
func (ml *MockLogger) Info(msg string, args ...interface{})  {}
func (ml *MockLogger) Warn(msg string, args ...interface{})  {}
func (ml *MockLogger) Error(msg string, args ...interface{}) {}
func (ml *MockLogger) Fatal(msg string, args ...interface{}) {}
func (ml *MockLogger) WithCorrelationID(id string) core.Logger {
	return ml
}

func TestNewContractManager(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	assert.NotNil(t, cm)
	assert.Equal(t, 0, cm.GetContractCount())
}

func TestLoadContractABI(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	// Simple ERC20 ABI
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

	err := cm.LoadContractABI("ERC20", abiJSON)
	require.NoError(t, err)

	assert.Equal(t, 1, cm.GetContractCount())
	assert.True(t, cm.HasContract("ERC20"))
}

func TestLoadContractABIFromString(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	abiJSON := `[
		{
			"type": "event",
			"name": "Transfer",
			"inputs": [
				{"name": "from", "type": "address", "indexed": true},
				{"name": "to", "type": "address", "indexed": true},
				{"name": "value", "type": "uint256", "indexed": false}
			]
		}
	]`

	err := cm.LoadContractABIFromString("ERC20", abiJSON)
	require.NoError(t, err)

	assert.True(t, cm.HasContract("ERC20"))
}

func TestLoadContractABIEmptyName(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	abiJSON := []byte(`[]`)

	err := cm.LoadContractABI("", abiJSON)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "contract name is required")
}

func TestLoadContractABIEmptyJSON(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	err := cm.LoadContractABI("ERC20", []byte{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ABI JSON is required")
}

func TestLoadContractABIInvalidJSON(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	err := cm.LoadContractABI("ERC20", []byte("invalid json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse ABI JSON")
}

func TestGetABI(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

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

	_ = cm.LoadContractABI("ERC20", abiJSON)

	retrievedABI, err := cm.GetABI("ERC20")
	require.NoError(t, err)

	assert.NotNil(t, retrievedABI)
	assert.Greater(t, len(retrievedABI.Events), 0)
}

func TestGetABINotFound(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	_, err := cm.GetABI("NonExistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetEventSignature(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

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

	_ = cm.LoadContractABI("ERC20", abiJSON)

	sig, err := cm.GetEventSignature("ERC20", "Transfer")
	require.NoError(t, err)

	assert.NotNil(t, sig)
	assert.NotEqual(t, [32]byte{}, sig)
}

func TestGetEventSignatureNotFound(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	abiJSON := []byte(`[
		{
			"type": "event",
			"name": "Transfer",
			"inputs": []
		}
	]`)

	_ = cm.LoadContractABI("ERC20", abiJSON)

	_, err := cm.GetEventSignature("ERC20", "NonExistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetEventSignatures(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

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

	_ = cm.LoadContractABI("ERC20", abiJSON)

	sigs, err := cm.GetEventSignatures("ERC20")
	require.NoError(t, err)

	assert.Equal(t, 2, len(sigs))
	assert.Contains(t, sigs, "Transfer")
	assert.Contains(t, sigs, "Approval")
}

func TestGetEvent(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

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

	_ = cm.LoadContractABI("ERC20", abiJSON)

	event, err := cm.GetEvent("ERC20", "Transfer")
	require.NoError(t, err)

	assert.Equal(t, "Transfer", event.Name)
	assert.Equal(t, 3, len(event.Inputs))
}

func TestGetEventNotFound(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	abiJSON := []byte(`[
		{
			"type": "event",
			"name": "Transfer",
			"inputs": []
		}
	]`)

	_ = cm.LoadContractABI("ERC20", abiJSON)

	_, err := cm.GetEvent("ERC20", "NonExistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetMethod(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	abiJSON := []byte(`[
		{
			"type": "function",
			"name": "transfer",
			"inputs": [
				{"name": "to", "type": "address"},
				{"name": "amount", "type": "uint256"}
			],
			"outputs": [
				{"name": "", "type": "bool"}
			]
		}
	]`)

	_ = cm.LoadContractABI("ERC20", abiJSON)

	method, err := cm.GetMethod("ERC20", "transfer")
	require.NoError(t, err)

	assert.Equal(t, "transfer", method.Name)
	assert.Equal(t, 2, len(method.Inputs))
}

func TestHasContract(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	abiJSON := []byte(`[]`)

	_ = cm.LoadContractABI("ERC20", abiJSON)

	assert.True(t, cm.HasContract("ERC20"))
	assert.False(t, cm.HasContract("NonExistent"))
}

func TestListContracts(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	abiJSON := []byte(`[]`)

	_ = cm.LoadContractABI("ERC20", abiJSON)
	_ = cm.LoadContractABI("ERC721", abiJSON)

	contracts := cm.ListContracts()

	assert.Equal(t, 2, len(contracts))
	assert.Contains(t, contracts, "ERC20")
	assert.Contains(t, contracts, "ERC721")
}

func TestRemoveContract(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	abiJSON := []byte(`[]`)

	_ = cm.LoadContractABI("ERC20", abiJSON)
	assert.Equal(t, 1, cm.GetContractCount())

	err := cm.RemoveContract("ERC20")
	require.NoError(t, err)

	assert.Equal(t, 0, cm.GetContractCount())
	assert.False(t, cm.HasContract("ERC20"))
}

func TestRemoveContractNotFound(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	err := cm.RemoveContract("NonExistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestClearContracts(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	abiJSON := []byte(`[]`)

	_ = cm.LoadContractABI("ERC20", abiJSON)
	_ = cm.LoadContractABI("ERC721", abiJSON)

	assert.Equal(t, 2, cm.GetContractCount())

	cm.ClearContracts()

	assert.Equal(t, 0, cm.GetContractCount())
}

func TestGetContractCount(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	abiJSON := []byte(`[]`)

	assert.Equal(t, 0, cm.GetContractCount())

	_ = cm.LoadContractABI("ERC20", abiJSON)
	assert.Equal(t, 1, cm.GetContractCount())

	_ = cm.LoadContractABI("ERC721", abiJSON)
	assert.Equal(t, 2, cm.GetContractCount())
}

func TestMultipleContractABIs(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	erc20ABI := []byte(`[
		{
			"type": "event",
			"name": "Transfer",
			"inputs": []
		}
	]`)

	erc721ABI := []byte(`[
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

	_ = cm.LoadContractABI("ERC20", erc20ABI)
	_ = cm.LoadContractABI("ERC721", erc721ABI)

	erc20Sigs, _ := cm.GetEventSignatures("ERC20")
	erc721Sigs, _ := cm.GetEventSignatures("ERC721")

	assert.Equal(t, 1, len(erc20Sigs))
	assert.Equal(t, 2, len(erc721Sigs))
}

func TestContractManagerConcurrency(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	abiJSON := []byte(`[]`)

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- struct{}{} }()
			contractName := "Contract" + string(rune(id))
			_ = cm.LoadContractABI(contractName, abiJSON)
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 10, cm.GetContractCount())
}

func TestLoadContractABIWithMethods(t *testing.T) {
	logger := &MockLogger{}
	cm := NewContractManager(logger)

	abiJSON := []byte(`[
		{
			"type": "function",
			"name": "transfer",
			"inputs": [
				{"name": "to", "type": "address"},
				{"name": "amount", "type": "uint256"}
			],
			"outputs": [
				{"name": "", "type": "bool"}
			]
		},
		{
			"type": "function",
			"name": "approve",
			"inputs": [
				{"name": "spender", "type": "address"},
				{"name": "amount", "type": "uint256"}
			],
			"outputs": [
				{"name": "", "type": "bool"}
			]
		}
	]`)

	err := cm.LoadContractABI("ERC20", abiJSON)
	require.NoError(t, err)

	abi, _ := cm.GetABI("ERC20")
	assert.Equal(t, 2, len(abi.Methods))
}
