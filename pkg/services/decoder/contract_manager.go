package decoder

import (
	"encoding/json"
	"fmt"
	"sync"

	"chainpulse/pkg/core"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// ContractMetadata holds contract ABI and metadata
type ContractMetadata struct {
	Address common.Address
	ABI     abi.ABI
	Name    string
	Version string
	Events  map[string]abi.Event
	Methods map[string]abi.Method
}

// ContractManager manages contract ABIs and metadata
type ContractManager struct {
	contracts map[string]*ContractMetadata
	mu        sync.RWMutex
	logger    core.Logger
}

// NewContractManager creates a new contract manager
func NewContractManager(logger core.Logger) *ContractManager {
	return &ContractManager{
		contracts: make(map[string]*ContractMetadata),
		logger:    logger,
	}
}

// LoadContractABI loads a contract ABI from JSON
func (cm *ContractManager) LoadContractABI(
	name string,
	abiJSON []byte,
) error {
	if name == "" {
		return fmt.Errorf("contract name is required")
	}

	if len(abiJSON) == 0 {
		return fmt.Errorf("ABI JSON is required")
	}

	var parsedABI abi.ABI
	if err := json.Unmarshal(abiJSON, &parsedABI); err != nil {
		return fmt.Errorf("failed to parse ABI JSON: %w", err)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	metadata := &ContractMetadata{
		ABI:     parsedABI,
		Name:    name,
		Events:  make(map[string]abi.Event),
		Methods: make(map[string]abi.Method),
	}

	// Index events
	for eventName, event := range parsedABI.Events {
		metadata.Events[eventName] = event
	}

	// Index methods
	for methodName, method := range parsedABI.Methods {
		metadata.Methods[methodName] = method
	}

	cm.contracts[name] = metadata

	cm.logger.Info("contract ABI loaded", map[string]interface{}{
		"contract": name,
		"events":   len(metadata.Events),
		"methods":  len(metadata.Methods),
	})

	return nil
}

// LoadContractABIFromString loads a contract ABI from JSON string
func (cm *ContractManager) LoadContractABIFromString(
	name string,
	abiJSON string,
) error {
	return cm.LoadContractABI(name, []byte(abiJSON))
}

// GetABI gets the ABI for a contract
func (cm *ContractManager) GetABI(name string) (abi.ABI, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	metadata, ok := cm.contracts[name]
	if !ok {
		return abi.ABI{}, fmt.Errorf("contract %s not found", name)
	}

	return metadata.ABI, nil
}

// GetEventSignature gets the event signature for a contract and event name
func (cm *ContractManager) GetEventSignature(
	contractName string,
	eventName string,
) (common.Hash, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	metadata, ok := cm.contracts[contractName]
	if !ok {
		return common.Hash{}, fmt.Errorf("contract %s not found", contractName)
	}

	event, ok := metadata.Events[eventName]
	if !ok {
		return common.Hash{}, fmt.Errorf("event %s not found in contract %s", eventName, contractName)
	}

	return event.ID, nil
}

// GetEventSignatures gets all event signatures for a contract
func (cm *ContractManager) GetEventSignatures(contractName string) (map[string]common.Hash, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	metadata, ok := cm.contracts[contractName]
	if !ok {
		return nil, fmt.Errorf("contract %s not found", contractName)
	}

	signatures := make(map[string]common.Hash)
	for eventName, event := range metadata.Events {
		signatures[eventName] = event.ID
	}

	return signatures, nil
}

// GetEvent gets an event from a contract
func (cm *ContractManager) GetEvent(
	contractName string,
	eventName string,
) (abi.Event, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	metadata, ok := cm.contracts[contractName]
	if !ok {
		return abi.Event{}, fmt.Errorf("contract %s not found", contractName)
	}

	event, ok := metadata.Events[eventName]
	if !ok {
		return abi.Event{}, fmt.Errorf("event %s not found in contract %s", eventName, contractName)
	}

	return event, nil
}

// GetMethod gets a method from a contract
func (cm *ContractManager) GetMethod(
	contractName string,
	methodName string,
) (abi.Method, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	metadata, ok := cm.contracts[contractName]
	if !ok {
		return abi.Method{}, fmt.Errorf("contract %s not found", contractName)
	}

	method, ok := metadata.Methods[methodName]
	if !ok {
		return abi.Method{}, fmt.Errorf("method %s not found in contract %s", methodName, contractName)
	}

	return method, nil
}

// HasContract checks if a contract is loaded
func (cm *ContractManager) HasContract(name string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	_, ok := cm.contracts[name]
	return ok
}

// ListContracts lists all loaded contracts
func (cm *ContractManager) ListContracts() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	contracts := make([]string, 0, len(cm.contracts))
	for name := range cm.contracts {
		contracts = append(contracts, name)
	}

	return contracts
}

// RemoveContract removes a contract from the manager
func (cm *ContractManager) RemoveContract(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, ok := cm.contracts[name]; !ok {
		return fmt.Errorf("contract %s not found", name)
	}

	delete(cm.contracts, name)
	cm.logger.Info("contract removed", map[string]interface{}{
		"contract": name,
	})

	return nil
}

// ClearContracts removes all contracts
func (cm *ContractManager) ClearContracts() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.contracts = make(map[string]*ContractMetadata)
	cm.logger.Info("all contracts cleared", nil)
}

// GetContractCount returns the number of loaded contracts
func (cm *ContractManager) GetContractCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return len(cm.contracts)
}
