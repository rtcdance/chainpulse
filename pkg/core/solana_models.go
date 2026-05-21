package core

import "strings"

const TokenProgramID = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
const Token2022ProgramID = "TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb"
const AssociatedTokenProgramID = "ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL"
const MetaplexTokenMetadataProgramID = "metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s"
const JupiterV6ProgramID = "JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4"
const RaydiumV4ProgramID = "675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8"
const OrcaWhirlpoolProgramID = "whirLbMiicVdio4qvUfM5KAg6Ct8VwpYzGff3uctyCc"

const SPLTransfer = "SPL:Transfer"
const SPLTransferChecked = "SPL:TransferChecked"
const SPLMintTo = "SPL:MintTo"
const SPLBurn = "SPL:Burn"
const SPLInitializeMint = "SPL:InitializeMint"
const SPLInitializeAccount = "SPL:InitializeAccount"
const SPLCloseAccount = "SPL:CloseAccount"

type SolanaEvent struct {
	BlockchainEvent
	Slot                    uint64   `json:"slot"`
	ProgramID               string   `json:"program_id"`
	AccountKeys             []string `json:"account_keys"`
	InstructionDiscriminator string   `json:"instruction_discriminator"`
	SPLTokenProgram         bool     `json:"spl_token_program"`
}

func NewSolanaEvent() *SolanaEvent {
	return &SolanaEvent{
		BlockchainEvent: BlockchainEvent{
			ChainID: "solana",
			Network: "solana",
		},
	}
}

type SolanaTransaction struct {
	Signature            string   `json:"signature"`
	Slot                 uint64   `json:"slot"`
	BlockTime            int64    `json:"block_time"`
	Err                  any      `json:"err"`
	LogMessages          []string `json:"log_messages"`
	AccountKeys          []string `json:"account_keys"`
	Instructions         []any    `json:"instructions"`
	Fee                  uint64   `json:"fee"`
	ComputeUnitsConsumed uint64   `json:"compute_units_consumed"`
	InstructionTypes     []string `json:"instruction_types"`
}

func ParseSolanaLogMessages(logs []string) map[string]any {
	result := make(map[string]any)
	var programData []string
	for _, log := range logs {
		if strings.HasPrefix(log, "Program data:") {
			trimmed := strings.TrimPrefix(log, "Program data:")
			programData = append(programData, trimmed)
		}
	}
	if len(programData) > 0 {
		result["program_data"] = programData
	}
	return result
}