package core

import (
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// knownEventABIs maps event names to their minimal ABI JSON fragments.
// These are used by DecodeEventData() to decode indexed and non-indexed parameters.
var knownEventABIs = map[string]string{
	// ERC-20 / ERC-721 Transfer(address indexed from, address indexed to, uint256 value)
	"Transfer": `[{"type":"event","name":"Transfer","inputs":[{"name":"from","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"value","type":"uint256","indexed":false}]}]`,
	// ERC-20 Approval(address indexed owner, address indexed spender, uint256 value)
	"Approval": `[{"type":"event","name":"Approval","inputs":[{"name":"owner","type":"address","indexed":true},{"name":"spender","type":"address","indexed":true},{"name":"value","type":"uint256","indexed":false}]}]`,
	// ERC-721 ApprovalForAll(address indexed owner, address indexed operator, bool approved)
	"ApprovalForAll": `[{"type":"event","name":"ApprovalForAll","inputs":[{"name":"owner","type":"address","indexed":true},{"name":"operator","type":"address","indexed":true},{"name":"approved","type":"bool","indexed":false}]}]`,
	// ERC-1155 TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value)
	"TransferSingle": `[{"type":"event","name":"TransferSingle","inputs":[{"name":"operator","type":"address","indexed":true},{"name":"from","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"id","type":"uint256","indexed":false},{"name":"value","type":"uint256","indexed":false}]}]`,
	// ERC-1155 TransferBatch(address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values)
	"TransferBatch": `[{"type":"event","name":"TransferBatch","inputs":[{"name":"operator","type":"address","indexed":true},{"name":"from","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"ids","type":"uint256[]","indexed":false},{"name":"values","type":"uint256[]","indexed":false}]}]`,
	// ERC-1155 URI(string value, uint256 indexed id)
	"URI": `[{"type":"event","name":"URI","inputs":[{"name":"value","type":"string","indexed":false},{"name":"id","type":"uint256","indexed":true}]}]`,
	// Uniswap V3 Swap(address indexed sender, int256 amount0, int256 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick)
	"Swap": `[{"type":"event","name":"Swap","inputs":[{"name":"sender","type":"address","indexed":true},{"name":"amount0","type":"int256","indexed":false},{"name":"amount1","type":"int256","indexed":false},{"name":"sqrtPriceX96","type":"uint160","indexed":false},{"name":"liquidity","type":"uint128","indexed":false},{"name":"tick","type":"int24","indexed":false}]}]`,
	// ChainPulse EventEmitter Ping(address sender, uint256 value)
	"Ping": `[{"type":"event","name":"Ping","inputs":[{"name":"sender","type":"address","indexed":false},{"name":"value","type":"uint256","indexed":false}]}]`,

	// --- DeFi: Lending Protocols ---

	// Aave V3 Supply(address indexed reserve, address user, address indexed onBehalfOf, uint256 amount, bool indexed referral)
	"Supply": `[{"type":"event","name":"Supply","inputs":[{"name":"reserve","type":"address","indexed":true},{"name":"user","type":"address","indexed":false},{"name":"onBehalfOf","type":"address","indexed":true},{"name":"amount","type":"uint256","indexed":false},{"name":"referral","type":"bool","indexed":true}]}]`,
	// Aave V3 Withdraw(address indexed reserve, address indexed user, address indexed to, uint256 amount)
	"DeFiWithdraw": `[{"type":"event","name":"Withdraw","inputs":[{"name":"reserve","type":"address","indexed":true},{"name":"user","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"amount","type":"uint256","indexed":false}]}]`,
	// Aave V3 Borrow(address indexed reserve, address user, address indexed onBehalfOf, uint256 amount, uint8 interestRateMode, bool indexed referral)
	"Borrow": `[{"type":"event","name":"Borrow","inputs":[{"name":"reserve","type":"address","indexed":true},{"name":"user","type":"address","indexed":false},{"name":"onBehalfOf","type":"address","indexed":true},{"name":"amount","type":"uint256","indexed":false},{"name":"interestRateMode","type":"uint8","indexed":false},{"name":"referral","type":"bool","indexed":true}]}]`,
	// Aave V3 Repay(address indexed reserve, address indexed user, address indexed repayer, uint256 amount, bool useATokens)
	"Repay": `[{"type":"event","name":"Repay","inputs":[{"name":"reserve","type":"address","indexed":true},{"name":"user","type":"address","indexed":true},{"name":"repayer","type":"address","indexed":true},{"name":"amount","type":"uint256","indexed":false},{"name":"useATokens","type":"bool","indexed":false}]}]`,
	// Aave V3 LiquidationCall(address indexed collateralAsset, address indexed debtAsset, address indexed user, uint256 debtToCover, uint256 liquidatedCollateralAmount, bool receiveAToken)
	"LiquidationCall": `[{"type":"event","name":"LiquidationCall","inputs":[{"name":"collateralAsset","type":"address","indexed":true},{"name":"debtAsset","type":"address","indexed":true},{"name":"user","type":"address","indexed":true},{"name":"debtToCover","type":"uint256","indexed":false},{"name":"liquidatedCollateralAmount","type":"uint256","indexed":false},{"name":"receiveAToken","type":"bool","indexed":false}]}]`,
	// Aave V3 ReserveDataUpdated(address indexed reserve, uint256 liquidityRate, uint256 stableBorrowRate, uint256 variableBorrowRate, uint256 liquidityIndex, uint256 variableBorrowIndex)
	"ReserveDataUpdated": `[{"type":"event","name":"ReserveDataUpdated","inputs":[{"name":"reserve","type":"address","indexed":true},{"name":"liquidityRate","type":"uint256","indexed":false},{"name":"stableBorrowRate","type":"uint256","indexed":false},{"name":"variableBorrowRate","type":"uint256","indexed":false},{"name":"liquidityIndex","type":"uint256","indexed":false},{"name":"variableBorrowIndex","type":"uint256","indexed":false}]}]`,
	// Compound V3 Supply(address indexed from, address indexed to, uint256 amount)
	"CometSupply": `[{"type":"event","name":"Supply","inputs":[{"name":"from","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"amount","type":"uint256","indexed":false}]}]`,
	// Compound V3 Withdraw(address indexed from, address indexed to, uint256 amount)
	"CometWithdraw": `[{"type":"event","name":"Withdraw","inputs":[{"name":"from","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"amount","type":"uint256","indexed":false}]}]`,
	// Compound V3 Borrow(address indexed account, uint256 amount, uint256 index)
	"CometBorrow": `[{"type":"event","name":"Borrow","inputs":[{"name":"account","type":"address","indexed":true},{"name":"amount","type":"uint256","indexed":false},{"name":"index","type":"uint256","indexed":false}]}]`,
	// Compound V3 Repay(address indexed from, address indexed to, uint256 amount)
	"CometRepay": `[{"type":"event","name":"Repay","inputs":[{"name":"from","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"amount","type":"uint256","indexed":false}]}]`,
	// Compound V3 Liquidate(address indexed liquidator, address indexed victim, uint256 amount, address indexed asset, bool isSupply)
	"CometLiquidate": `[{"type":"event","name":"Liquidate","inputs":[{"name":"liquidator","type":"address","indexed":true},{"name":"victim","type":"address","indexed":true},{"name":"amount","type":"uint256","indexed":false},{"name":"asset","type":"address","indexed":true},{"name":"isSupply","type":"bool","indexed":false}]}]`,

	// --- DeFi: DEX Protocols ---

	// Uniswap V2 Swap(address indexed sender, uint256 amount0In, uint256 amount1In, uint256 amount0Out, uint256 amount1Out, address indexed to)
	"UniswapV2Swap": `[{"type":"event","name":"Swap","inputs":[{"name":"sender","type":"address","indexed":true},{"name":"amount0In","type":"uint256","indexed":false},{"name":"amount1In","type":"uint256","indexed":false},{"name":"amount0Out","type":"uint256","indexed":false},{"name":"amount1Out","type":"uint256","indexed":false},{"name":"to","type":"address","indexed":true}]}]`,
	// Uniswap V2 Sync(uint112 reserve0, uint112 reserve1)
	"Sync": `[{"type":"event","name":"Sync","inputs":[{"name":"reserve0","type":"uint112","indexed":false},{"name":"reserve1","type":"uint112","indexed":false}]}]`,
	// Uniswap V2 PairCreated(address indexed token0, address indexed token1, address pair, uint256)
	"PairCreated": `[{"type":"event","name":"PairCreated","inputs":[{"name":"token0","type":"address","indexed":true},{"name":"token1","type":"address","indexed":true},{"name":"pair","type":"address","indexed":false},{"name":"","type":"uint256","indexed":false}]}]`,
	// Curve TokenExchange(address indexed buyer, int128 sold_id, int128 bought_id, uint256 tokens_sold, uint256 tokens_bought)
	"TokenExchange": `[{"type":"event","name":"TokenExchange","inputs":[{"name":"buyer","type":"address","indexed":true},{"name":"sold_id","type":"int128","indexed":false},{"name":"bought_id","type":"int128","indexed":false},{"name":"tokens_sold","type":"uint256","indexed":false},{"name":"tokens_bought","type":"uint256","indexed":false}]}]`,
	// Curve AddLiquidity(address indexed provider, uint256 token_amount, uint256[] token_supply)
	"AddLiquidity": `[{"type":"event","name":"AddLiquidity","inputs":[{"name":"provider","type":"address","indexed":true},{"name":"token_amount","type":"uint256","indexed":false},{"name":"token_supply","type":"uint256[]","indexed":false}]}]`,
	// Balancer V2 Swap(address indexed tokenIn, address indexed tokenOut, uint256 amountIn, uint256 amountOut)
	"BalancerSwap": `[{"type":"event","name":"Swap","inputs":[{"name":"tokenIn","type":"address","indexed":true},{"name":"tokenOut","type":"address","indexed":true},{"name":"amountIn","type":"uint256","indexed":false},{"name":"amountOut","type":"uint256","indexed":false}]}]`,

	// --- Governance ---

	// OpenZeppelin Governor ProposalCreated(uint256 proposalId, address indexed proposer, address[] targets, uint256[] values, string[] signatures, bytes[] calldatas, uint256 voteStart, uint256 voteEnd, string description)
	"ProposalCreated": `[{"type":"event","name":"ProposalCreated","inputs":[{"name":"proposalId","type":"uint256","indexed":false},{"name":"proposer","type":"address","indexed":true},{"name":"targets","type":"address[]","indexed":false},{"name":"values","type":"uint256[]","indexed":false},{"name":"signatures","type":"string[]","indexed":false},{"name":"calldatas","type":"bytes[]","indexed":false},{"name":"voteStart","type":"uint256","indexed":false},{"name":"voteEnd","type":"uint256","indexed":false},{"name":"description","type":"string","indexed":false}]}]`,
	// OpenZeppelin Governor VoteCast(address indexed voter, uint256 proposalId, uint8 support, uint256 weight, string reason)
	"VoteCast": `[{"type":"event","name":"VoteCast","inputs":[{"name":"voter","type":"address","indexed":true},{"name":"proposalId","type":"uint256","indexed":false},{"name":"support","type":"uint8","indexed":false},{"name":"weight","type":"uint256","indexed":false},{"name":"reason","type":"string","indexed":false}]}]`,
	// OpenZeppelin Governor ProposalExecuted(uint256 proposalId)
	"ProposalExecuted": `[{"type":"event","name":"ProposalExecuted","inputs":[{"name":"proposalId","type":"uint256","indexed":false}]}]`,
	// OpenZeppelin Governor ProposalCanceled(uint256 proposalId)
	"ProposalCanceled": `[{"type":"event","name":"ProposalCanceled","inputs":[{"name":"proposalId","type":"uint256","indexed":false}]}]`,

	// --- ERC-4337 Account Abstraction ---

	// EntryPoint UserOperationEvent(address indexed sender, bytes32 userOpHash, uint256 nonce, bool success, uint256 actualGasCost, uint256 actualGasUsed)
	"UserOperationEvent": `[{"type":"event","name":"UserOperationEvent","inputs":[{"name":"sender","type":"address","indexed":true},{"name":"userOpHash","type":"bytes32","indexed":false},{"name":"nonce","type":"uint256","indexed":false},{"name":"success","type":"bool","indexed":false},{"name":"actualGasCost","type":"uint256","indexed":false},{"name":"actualGasUsed","type":"uint256","indexed":false}]}]`,
	// EntryPoint AccountDeployed(address indexed sender, bytes32 userOpHash)
	"AccountDeployed": `[{"type":"event","name":"AccountDeployed","inputs":[{"name":"sender","type":"address","indexed":true},{"name":"userOpHash","type":"bytes32","indexed":false}]}]`,
	// EntryPoint BeforeExecution()
	"BeforeExecution": `[{"type":"event","name":"BeforeExecution","inputs":[]}]`,
	// EntryPoint AfterExecution()
	"AfterExecution": `[{"type":"event","name":"AfterExecution","inputs":[]}]`,

	// --- Cross-Chain Bridge Events ---

	// LayerZero PacketSent(uint16 dstChainId, bytes dstUaAddress, uint64 nonce)
	"PacketSent": `[{"type":"event","name":"PacketSent","inputs":[{"name":"dstChainId","type":"uint16","indexed":false},{"name":"dstUaAddress","type":"bytes","indexed":false},{"name":"nonce","type":"uint64","indexed":false}]}]`,
	// LayerZero PacketDelivered(bytes srcAddress)
	"PacketDelivered": `[{"type":"event","name":"PacketDelivered","inputs":[{"name":"srcAddress","type":"bytes","indexed":false}]}]`,
	// Wormhole LogMessagePublished(address indexed sender, uint64 sequence, uint32 nonce, uint32 consistencyLevel, bytes payload, uint8 guardianSetIndex)
	"LogMessagePublished": `[{"type":"event","name":"LogMessagePublished","inputs":[{"name":"sender","type":"address","indexed":true},{"name":"sequence","type":"uint64","indexed":false},{"name":"nonce","type":"uint32","indexed":false},{"name":"consistencyLevel","type":"uint32","indexed":false},{"name":"payload","type":"bytes","indexed":false},{"name":"guardianSetIndex","type":"uint8","indexed":false}]}]`,
	// Optimism L1CrossDomainMessenger SentMessage(address indexed target, address sender, bytes message, uint256 messageNonce, uint256 gasLimit)
	"SentMessage": `[{"type":"event","name":"SentMessage","inputs":[{"name":"target","type":"address","indexed":true},{"name":"sender","type":"address","indexed":false},{"name":"message","type":"bytes","indexed":false},{"name":"messageNonce","type":"uint256","indexed":false},{"name":"gasLimit","type":"uint256","indexed":false}]}]`,
	// Optimism L1CrossDomainMessenger RelayedMessage(bytes32 indexed msgHash, bool success)
	"RelayedMessage": `[{"type":"event","name":"RelayedMessage","inputs":[{"name":"msgHash","type":"bytes32","indexed":true},{"name":"success","type":"bool","indexed":false}]}]`,
	// Arbitrum Inbox TxToL2(address indexed from, address indexed to, uint256 indexed seqNum, bytes data)
	"TxToL2": `[{"type":"event","name":"TxToL2","inputs":[{"name":"from","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"seqNum","type":"uint256","indexed":true},{"name":"data","type":"bytes","indexed":false}]}]`,
	// Arbitrum NodeInterface L2TxCreated(uint256 indexed ticketNum)
	"L2TxCreated": `[{"type":"event","name":"L2TxCreated","inputs":[{"name":"ticketNum","type":"uint256","indexed":true}]}]`,

	// --- ERC-4337 Stake Events ---

	// StakeLocked(address indexed account, uint256 totalStaked, uint256 unstakeDelaySec)
	"StakeLocked": `[{"type":"event","name":"StakeLocked","inputs":[{"name":"account","type":"address","indexed":true},{"name":"totalStaked","type":"uint256","indexed":false},{"name":"unstakeDelaySec","type":"uint256","indexed":false}]}]`,

	// StakeUnlocked(address indexed account)
	"StakeUnlocked": `[{"type":"event","name":"StakeUnlocked","inputs":[{"name":"account","type":"address","indexed":true}]}]`,

	// StakeWithdrawn(address indexed account, address withdrawTo)
	"StakeWithdrawn": `[{"type":"event","name":"StakeWithdrawn","inputs":[{"name":"account","type":"address","indexed":true},{"name":"withdrawTo","type":"address","indexed":false}]}]`,

	// --- Post-Dencun Events ---

	// EIP-7002 WithdrawalRequested(address indexed source, bytes pubkey, uint256 amount)
	"WithdrawalRequested": `[{"type":"event","name":"WithdrawalRequested","inputs":[{"name":"source","type":"address","indexed":true},{"name":"pubkey","type":"bytes","indexed":false},{"name":"amount","type":"uint256","indexed":false}]}]`,

	// EIP-6110 DepositEvent(bytes pubkey, bytes withdrawal_credentials, bytes amount, bytes signature, bytes index)
	"DepositEvent": `[{"type":"event","name":"DepositEvent","inputs":[{"name":"pubkey","type":"bytes","indexed":false},{"name":"withdrawal_credentials","type":"bytes","indexed":false},{"name":"amount","type":"bytes","indexed":false},{"name":"signature","type":"bytes","indexed":false},{"name":"index","type":"bytes","indexed":false}]}]`,
}

// parsedEventABIs caches the parsed abi.ABI objects, keyed by event name.
var parsedEventABIs map[string]*abi.ABI

// topic0Registry maps keccak256 event signature hashes to event names.
var topic0Registry map[string]string

func init() {
	parsedEventABIs = make(map[string]*abi.ABI, len(knownEventABIs))
	topic0Registry = make(map[string]string, len(knownEventABIs))
	for name, jsonStr := range knownEventABIs {
		parsed, err := abi.JSON(strings.NewReader(jsonStr))
		if err == nil {
			parsedEventABIs[name] = &parsed
			// Build topic0 reverse lookup: event.ID is keccak256 of the event signature
			if event, ok := parsed.Events[name]; ok {
				topic0Registry[event.ID.Hex()] = name
			}
		}
	}
}

// ResolveEventNameByTopic0 looks up an event name by its topic0 hash.
// Returns the event name and true if found, or empty string and false otherwise.
func ResolveEventNameByTopic0(topic0Hex string) (string, bool) {
	name, ok := topic0Registry[topic0Hex]
	return name, ok
}

// RegisterTopic0Mapping adds a custom topic0 → event name mapping.
// This allows runtime registration of event signatures discovered dynamically.
var topic0Mu sync.RWMutex

func RegisterTopic0Mapping(topic0Hex, eventName string) {
	topic0Mu.Lock()
	defer topic0Mu.Unlock()
	topic0Registry[topic0Hex] = eventName
}

// GetABIForEventName returns the pre-parsed ABI for a known event name, or nil.
func GetABIForEventName(name string) *abi.ABI {
	return parsedEventABIs[name]
}

// GetAllTopic0Hashes returns all known topic0 keccak256 hashes from the registry.
// This is used to build the topics filter for eth_getLogs when EventSignatures
// is not explicitly configured but all known events should be filtered.
func GetAllTopic0Hashes() []string {
	topic0Mu.RLock()
	defer topic0Mu.RUnlock()

	hashes := make([]string, 0, len(topic0Registry))
	for hash := range topic0Registry {
		hashes = append(hashes, hash)
	}
	return hashes
}
