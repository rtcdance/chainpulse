package e2e

import (
	"fmt"
	"math/big"
	"os"
	"os/exec"
)

// TestFixtures provides pre-configured test data and utilities
type TestFixtures struct {
	ERC20Contract  ContractDefinition
	ERC721Contract ContractDefinition
	TestAccounts   []TestAccount
}

// NewTestFixtures creates new test fixtures
func NewTestFixtures() *TestFixtures {
	return &TestFixtures{
		ERC20Contract:  getERC20ContractDefinition(),
		ERC721Contract: getERC721ContractDefinition(),
		TestAccounts:   getTestAccounts(),
	}
}

// getERC20ContractDefinition returns ERC20 contract definition
func getERC20ContractDefinition() ContractDefinition {
	return ContractDefinition{
		Name: "TestToken",
		ABI: `[
			{
				"anonymous": false,
				"inputs": [
					{"indexed": true, "name": "from", "type": "address"},
					{"indexed": true, "name": "to", "type": "address"},
					{"indexed": false, "name": "value", "type": "uint256"}
				],
				"name": "Transfer",
				"type": "event"
			},
			{
				"anonymous": false,
				"inputs": [
					{"indexed": true, "name": "owner", "type": "address"},
					{"indexed": true, "name": "spender", "type": "address"},
					{"indexed": false, "name": "value", "type": "uint256"}
				],
				"name": "Approval",
				"type": "event"
			}
		]`,
		Bytecode: "0x60806040", // Simplified bytecode
		Constructor: []any{
			"Test Token",
			"TEST",
			big.NewInt(1000000),
		},
	}
}

// getERC721ContractDefinition returns ERC721 contract definition
func getERC721ContractDefinition() ContractDefinition {
	return ContractDefinition{
		Name: "TestNFT",
		ABI: `[
			{
				"anonymous": false,
				"inputs": [
					{"indexed": true, "name": "from", "type": "address"},
					{"indexed": true, "name": "to", "type": "address"},
					{"indexed": true, "name": "tokenId", "type": "uint256"}
				],
				"name": "Transfer",
				"type": "event"
			},
			{
				"anonymous": false,
				"inputs": [
					{"indexed": true, "name": "owner", "type": "address"},
					{"indexed": true, "name": "approved", "type": "address"},
					{"indexed": true, "name": "tokenId", "type": "uint256"}
				],
				"name": "Approval",
				"type": "event"
			}
		]`,
		Bytecode: "0x60806040", // Simplified bytecode
		Constructor: []any{
			"Test NFT",
			"TNFT",
		},
	}
}

// getTestAccounts returns test accounts with pre-funded balances
func getTestAccounts() []TestAccount {
	return []TestAccount{
		{
			Address: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
			Balance: "1000000000000000000", // 1 ETH
			Key:     "0xac0974bec39a17e36ba4a6b4d238ff944bacb476caded732d4dff72ef700a0e0",
		},
		{
			Address: "0x70997970C51812e339D9B73b0245ad59cc5ffe89",
			Balance: "1000000000000000000",
			Key:     "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
		},
		{
			Address: "0x3C44CdDdB6a900c6671B362144b7B1caD82ab6FB",
			Balance: "1000000000000000000",
			Key:     "0x5de4111afa1a4b94908f83103db1fb1da08f45dfa6e1e06e57bc7b8f0ad1e4a2",
		},
		{
			Address: "0x90F79bf6EB2c4f870365E785982E1f101E93b906",
			Balance: "1000000000000000000",
			Key:     "0xe9af7928cd6925912261d3b73def4d7cacf0f80c9fd921dc798b3d014df5e53e",
		},
		{
			Address: "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65",
			Balance: "1000000000000000000",
			Key:     "0x877dad265659036145cfb16fc1d64dbe50b708b9fdc516e6918351ba9d52c50e",
		},
	}
}

// CreateERC20TransferEvent creates a mock ERC20 Transfer event
func CreateERC20TransferEvent(from, to string, amount *big.Int) map[string]any {
	return map[string]any{
		"from":  from,
		"to":    to,
		"value": amount,
	}
}

// CreateERC721TransferEvent creates a mock ERC721 Transfer event
func CreateERC721TransferEvent(from, to string, tokenID *big.Int) map[string]any {
	return map[string]any{
		"from":    from,
		"to":      to,
		"tokenId": tokenID,
	}
}

// CreateERC20ApprovalEvent creates a mock ERC20 Approval event
func CreateERC20ApprovalEvent(owner, spender string, amount *big.Int) map[string]any {
	return map[string]any{
		"owner":   owner,
		"spender": spender,
		"value":   amount,
	}
}

// CreateERC721ApprovalEvent creates a mock ERC721 Approval event
func CreateERC721ApprovalEvent(owner, approved string, tokenID *big.Int) map[string]any {
	return map[string]any{
		"owner":    owner,
		"approved": approved,
		"tokenId":  tokenID,
	}
}

// AssertionHelpers provides assertion utilities
type AssertionHelpers struct{}

// NewAssertionHelpers creates new assertion helpers
func NewAssertionHelpers() *AssertionHelpers {
	return &AssertionHelpers{}
}

// AssertEventExists checks if an event exists in a list
func (ah *AssertionHelpers) AssertEventExists(events []*IndexedEvent, txHash string, logIndex uint32) bool {
	for _, event := range events {
		if event.TxHash == txHash && event.LogIndex == logIndex {
			return true
		}
	}
	return false
}

// AssertEventCount checks if the event count matches expected
func (ah *AssertionHelpers) AssertEventCount(events []*IndexedEvent, expected int) bool {
	return len(events) == expected
}

// AssertEventOrdering checks if events are properly ordered
func (ah *AssertionHelpers) AssertEventOrdering(events []*IndexedEvent) bool {
	for i := 1; i < len(events); i++ {
		prev := events[i-1]
		curr := events[i]

		if curr.BlockNumber < prev.BlockNumber {
			return false
		}

		if curr.BlockNumber == prev.BlockNumber && curr.LogIndex < prev.LogIndex {
			return false
		}
	}
	return true
}

// AssertEventDataIntegrity checks if event data is intact
func (ah *AssertionHelpers) AssertEventDataIntegrity(event *IndexedEvent) bool {
	if event == nil {
		return false
	}

	if event.ID == "" || event.ContractAddress == "" || event.EventName == "" {
		return false
	}

	if event.TxHash == "" || event.DecodedData == nil {
		return false
	}

	return true
}

// AssertNoDuplicates checks if there are no duplicate events
func (ah *AssertionHelpers) AssertNoDuplicates(events []*IndexedEvent) bool {
	seen := make(map[string]bool)

	for _, event := range events {
		key := fmt.Sprintf("%s_%d", event.TxHash, event.LogIndex)
		if seen[key] {
			return false
		}
		seen[key] = true
	}

	return true
}

// AssertEventParameter checks if an event has a specific parameter
func (ah *AssertionHelpers) AssertEventParameter(event *IndexedEvent, paramName string) bool {
	if event == nil || event.DecodedData == nil {
		return false
	}

	_, exists := event.DecodedData[paramName]
	return exists
}

// AssertEventParameterValue checks if an event parameter has a specific value
func (ah *AssertionHelpers) AssertEventParameterValue(event *IndexedEvent, paramName string, expectedValue any) bool {
	if event == nil || event.DecodedData == nil {
		return false
	}

	value, exists := event.DecodedData[paramName]
	if !exists {
		return false
	}

	return value == expectedValue
}

// IsAnvilAvailable checks if Anvil is available in the system
func IsAnvilAvailable() bool {
	paths := []string{"anvil", "$HOME/.foundry/bin/anvil", "$HOME/.local/bin/anvil"}
	for _, p := range paths {
		p = os.ExpandEnv(p)
		_, err := exec.LookPath(p)
		if err == nil {
			return true
		}
	}
	return false
}
