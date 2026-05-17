package core

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// LiquidationEvent represents a decoded Aave V3 LiquidationCall event.
// This is the most critical DeFi risk event — it signals that a borrower's
// position was forcibly closed due to falling below the health factor threshold.
type LiquidationEvent struct {
	CollateralAsset            common.Address `json:"collateral_asset"`
	DebtAsset                  common.Address `json:"debt_asset"`
	Debtor                     common.Address `json:"debtor"`
	Liquidator                 common.Address `json:"liquidator"`
	DebtToCover                *big.Int       `json:"debt_to_cover"`
	LiquidatedCollateralAmount *big.Int       `json:"liquidated_collateral_amount"`
	ReceiveAToken              bool           `json:"receive_a_token"`
}

// IsLiquidationTopic0 checks if the given topic0 matches Aave V3 LiquidationCall.
func IsLiquidationTopic0(topic0 string) bool {
	return topic0 == "0x3a84f64446e8eada995aa9da2ddbfcd9b5d5d650503b19f024096d04c05ef2a9"
}

// DEXSwapEvent represents a generic DEX swap event decoded across protocols.
// Different DEX protocols emit different swap events, but they all share
// the concept of tokens flowing in and out.
type DEXSwapEvent struct {
	Protocol  string         `json:"protocol"` // "uniswap_v2", "uniswap_v3", "curve", "balancer"
	Sender    common.Address `json:"sender"`
	To        common.Address `json:"to"`
	TokenIn   common.Address `json:"token_in"`   // zero if not available
	TokenOut  common.Address `json:"token_out"`  // zero if not available
	AmountIn  *big.Int       `json:"amount_in"`  // V2: amount0In+amount1In, V3: amount0 or amount1
	AmountOut *big.Int       `json:"amount_out"` // V2: amount0Out+amount1Out, V3: the other amount
	Pool      common.Address `json:"pool"`       // contract address emitting the event
}

// LendingSupplyEvent represents a supply/deposit event from a lending protocol.
type LendingSupplyEvent struct {
	Protocol   string         `json:"protocol"` // "aave_v3", "compound_v3"
	Reserve    common.Address `json:"reserve"`
	User       common.Address `json:"user"`
	OnBehalfOf common.Address `json:"on_behalf_of"`
	Amount     *big.Int       `json:"amount"`
}

// LendingBorrowEvent represents a borrow event from a lending protocol.
type LendingBorrowEvent struct {
	Protocol         string         `json:"protocol"`
	Reserve          common.Address `json:"reserve"`
	User             common.Address `json:"user"`
	OnBehalfOf       common.Address `json:"on_behalf_of"`
	Amount           *big.Int       `json:"amount"`
	InterestRateMode uint8          `json:"interest_rate_mode"` // 1=stable, 2=variable
}

// ReserveUpdateEvent represents an Aave V3 ReserveDataUpdated event.
// This is emitted every time a reserve's interest rates change and is
// critical for calculating health factors and liquidation thresholds.
type ReserveUpdateEvent struct {
	Reserve             common.Address `json:"reserve"`
	LiquidityRate       *big.Int       `json:"liquidity_rate"`
	StableBorrowRate    *big.Int       `json:"stable_borrow_rate"`
	VariableBorrowRate  *big.Int       `json:"variable_borrow_rate"`
	LiquidityIndex      *big.Int       `json:"liquidity_index"`
	VariableBorrowIndex *big.Int       `json:"variable_borrow_index"`
}

// GovernanceProposalEvent represents a governance proposal lifecycle event.
type GovernanceProposalEvent struct {
	ProposalID *big.Int       `json:"proposal_id"`
	Proposer   common.Address `json:"proposer"`
	State      string         `json:"state"` // "created", "executed", "canceled"
}

// GovernanceVoteEvent represents a governance vote cast event.
type GovernanceVoteEvent struct {
	Voter      common.Address `json:"voter"`
	ProposalID *big.Int       `json:"proposal_id"`
	Support    uint8          `json:"support"` // 0=against, 1=for, 2=abstain
	Weight     *big.Int       `json:"weight"`
	Reason     string         `json:"reason"`
}

// HealthFactor calculates the health factor for a lending position.
// HF = (collateral * liquidationThreshold) / totalDebt
// HF > 1: safe, HF = 1: liquidation threshold, HF < 1: eligible for liquidation
func HealthFactor(collateralValue, liquidationThreshold, totalDebt *big.Int) *big.Int {
	if totalDebt == nil || totalDebt.Sign() == 0 {
		// No debt = infinite health factor
		return nil
	}
	if collateralValue == nil || liquidationThreshold == nil {
		return big.NewInt(0)
	}

	// HF = (collateral * threshold) / debt
	// Using big.Int for precision: multiply first, then divide
	numerator := new(big.Int).Mul(collateralValue, liquidationThreshold)
	// Scale by 1e18 for precision (like Aave does)
	numerator.Mul(numerator, big.NewInt(1e18))
	healthFactor := new(big.Int).Div(numerator, totalDebt)

	return healthFactor
}

// IsLiquidatable checks if a health factor indicates a liquidatable position.
func IsLiquidatable(healthFactor *big.Int) bool {
	if healthFactor == nil {
		return false // No debt = not liquidatable
	}
	// Health factor < 1e18 (1.0 in ray) means liquidatable
	return healthFactor.Cmp(big.NewInt(1e18)) < 0
}

// ProtocolName returns a human-readable protocol name from the protocol string.
func ProtocolName(protocol string) string {
	names := map[string]string{
		"aave_v3":     "Aave V3",
		"compound_v3": "Compound V3",
		"uniswap_v2":  "Uniswap V2",
		"uniswap_v3":  "Uniswap V3",
		"curve":       "Curve",
		"balancer":    "Balancer V2",
	}
	if name, ok := names[protocol]; ok {
		return name
	}
	return protocol
}

// --- ERC-4337 Account Abstraction Events ---

// UserOperationEvent represents a parsed ERC-4337 UserOperationEvent
// emitted by the EntryPoint contract after a UserOperation is processed.
type UserOperationEvent struct {
	Sender        common.Address `json:"sender"`
	UserOpHash    common.Hash    `json:"user_op_hash"`
	Nonce         *big.Int       `json:"nonce"`
	Success       bool           `json:"success"`
	ActualGasCost *big.Int       `json:"actual_gas_cost"`
	ActualGasUsed uint64         `json:"actual_gas_used"`
}

// IsSuccessful checks if the UserOperation succeeded.
func (e *UserOperationEvent) IsSuccessful() bool {
	return e.Success
}

// GasEfficiency computes the gas efficiency ratio (used/limit).
// Returns 0 if actualGasUsed is 0.
func (e *UserOperationEvent) GasEfficiency(gasLimit uint64) float64 {
	if gasLimit == 0 || e.ActualGasUsed == 0 {
		return 0
	}
	return float64(e.ActualGasUsed) / float64(gasLimit)
}

// AccountDeployedEvent represents a parsed ERC-4337 AccountDeployed event
// emitted when a smart account is first deployed via initCode.
type AccountDeployedEvent struct {
	Sender     common.Address `json:"sender"`
	UserOpHash common.Hash    `json:"user_op_hash"`
}

// --- Cross-Chain Bridge Events ---

// BridgeTransferEvent represents a cross-chain bridge transfer, correlating
// events from protocols like LayerZero, Wormhole, Optimism, and Arbitrum.
type BridgeTransferEvent struct {
	Protocol    string         `json:"protocol"`     // "layerzero", "wormhole", "optimism", "arbitrum"
	SourceChain uint64         `json:"source_chain"` // EVM chain ID or LayerZero chain ID
	DestChain   uint64         `json:"dest_chain"`
	Sender      common.Address `json:"sender"`
	Receiver    common.Address `json:"receiver"`
	Token       common.Address `json:"token,omitempty"`
	Amount      *big.Int       `json:"amount,omitempty"`
	MessageID   common.Hash    `json:"message_id"`
	Status      string         `json:"status"` // "sent", "delivered", "relayed"
}

// IsCrossChain checks if the transfer spans different chains.
func (e *BridgeTransferEvent) IsCrossChain() bool {
	return e.SourceChain != e.DestChain
}

// LayerZeroChainIDToEVM maps LayerZero chain IDs to EVM chain IDs for common chains.
var LayerZeroChainIDToEVM = map[uint64]uint64{
	101: 1,     // Ethereum Mainnet
	102: 56,    // BSC
	106: 137,   // Polygon
	110: 42161, // Arbitrum One
	111: 43114, // Avalanche C-Chain
	112: 10,    // Optimism
}

// --- L2 Bridge Events (Optimism, Arbitrum) ---

// OptimismSentMessageEvent represents an Optimism L2→L1 message sent via
// the L2CrossDomainMessenger. This is emitted when a message is sent from L2
// to L1 and must be relayed on L1 after the challenge window.
//
// Reference: https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L2/L2CrossDomainMessenger.sol
type OptimismSentMessageEvent struct {
	Target       common.Address `json:"target"`
	Sender       common.Address `json:"sender"`
	Message      []byte         `json:"message"`
	MessageNonce *big.Int       `json:"message_nonce"`
	GasLimit     *big.Int       `json:"gas_limit"`
}

// OptimismRelayedMessageEvent represents a message successfully relayed on L1
// by the Optimism L1CrossDomainMessenger.
type OptimismRelayedMessageEvent struct {
	MessageHash common.Hash `json:"msg_hash"`
}

// OptimismFailedRelayedMessageEvent represents a message that failed to relay on L1.
type OptimismFailedRelayedMessageEvent struct {
	MessageHash common.Hash `json:"msg_hash"`
}

// ArbitrumL2ToL1TransactionEvent represents an Arbitrum L2→L1 transaction.
// Emitted by the Arbitrum Outbox contract when a withdrawal is initiated on L2.
//
// Reference: https://github.com/OffchainLabs/arbitrum/blob/master/src/bridge/ArbSys.sol
type ArbitrumL2ToL1TransactionEvent struct {
	Caller      common.Address `json:"caller"`
	Destination common.Address `json:"destination"`
	Hash        common.Hash    `json:"hash"`
	Position    *big.Int       `json:"position"`
	Timestamp   *big.Int       `json:"timestamp"`
	Data        []byte         `json:"data"`
}

// ArbitrumRetryableTicketEvent represents an Arbitrum retryable ticket creation.
// Retryable tickets are the L1→L2 deposit mechanism for Arbitrum.
type ArbitrumRetryableTicketEvent struct {
	TicketID    common.Hash    `json:"ticket_id"`
	Sender      common.Address `json:"sender"`
	Destination common.Address `json:"destination"`
	Value       *big.Int       `json:"value"`
}

// L1TransactionDepositedEvent represents an L1→L2 deposit on Optimism.
// Emitted by the OptimismPortal contract when ETH or data is deposited to L2.
type L1TransactionDepositedEvent struct {
	From    common.Address `json:"from"`
	To      common.Address `json:"to"`
	Version *big.Int       `json:"version"`
	Data    []byte         `json:"data"`
}

// BridgeEventType categorizes a cross-chain bridge event for routing and indexing.
type BridgeEventType string

const (
	BridgeOptimismSent      BridgeEventType = "optimism_sent"
	BridgeOptimismRelayed   BridgeEventType = "optimism_relayed"
	BridgeArbitrumOutbound  BridgeEventType = "arbitrum_outbound"
	BridgeArbitrumRetryable BridgeEventType = "arbitrum_retryable"
	BridgeL1Deposit         BridgeEventType = "l1_deposit"
)

// --- NFT Events ---

// NFTTransferEvent represents a parsed NFT transfer event for ERC-721 or ERC-1155.
type NFTTransferEvent struct {
	Standard string         `json:"standard"` // "ERC-721", "ERC-1155"
	Operator common.Address `json:"operator"`
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	TokenID  *big.Int       `json:"token_id,omitempty"`  // ERC-721 single
	TokenIDs []*big.Int     `json:"token_ids,omitempty"` // ERC-1155 batch
	Amounts  []*big.Int     `json:"amounts,omitempty"`   // ERC-1155 batch amounts
	Contract common.Address `json:"contract"`
}

// IsMint checks if this is a minting event (from zero address).
func (e *NFTTransferEvent) IsMint() bool {
	return e.From == (common.Address{})
}

// IsBurn checks if this is a burning event (to zero address).
func (e *NFTTransferEvent) IsBurn() bool {
	return e.To == (common.Address{})
}
