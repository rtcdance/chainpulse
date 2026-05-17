package core

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// ─── ERC-4626 Tokenized Vault Standard ───────────────────────────────────────

// VaultShareMath implements the deposit/withdrawal math for ERC-4626 vaults.
// The core invariant: shares = assets * totalSupply / totalAssets
// (or the inverse for previewDeposit/previewRedeem).
type VaultShareMath struct {
	TotalAssets *big.Int `json:"total_assets"` // total managed assets
	TotalSupply *big.Int `json:"total_supply"` // total vault shares
	Decimals    uint8    `json:"decimals"`     // vault share decimals
}

// NewVaultShareMath creates a new vault share math helper.
func NewVaultShareMath(totalAssets, totalSupply *big.Int, decimals uint8) *VaultShareMath {
	return &VaultShareMath{
		TotalAssets: totalAssets,
		TotalSupply: totalSupply,
		Decimals:    decimals,
	}
}

// PreviewDeposit returns the shares that would be minted for a given deposit.
// Formula: shares = depositAssets * totalSupply / totalAssets
func (v *VaultShareMath) PreviewDeposit(assets *big.Int) *big.Int {
	if v.TotalAssets.Sign() == 0 || v.TotalSupply.Sign() == 0 {
		// First deposit: 1:1 ratio
		return new(big.Int).Set(assets)
	}
	return new(big.Int).Div(
		new(big.Int).Mul(assets, v.TotalSupply),
		v.TotalAssets,
	)
}

// PreviewMint returns the assets needed to mint a given number of shares.
// Formula: assets = shares * totalAssets / totalSupply
func (v *VaultShareMath) PreviewMint(shares *big.Int) *big.Int {
	if v.TotalSupply.Sign() == 0 {
		return new(big.Int).Set(shares)
	}
	return new(big.Int).Div(
		new(big.Int).Mul(shares, v.TotalAssets),
		v.TotalSupply,
	)
}

// PreviewWithdraw returns the shares that would be burned for a given withdrawal.
// Formula: shares = withdrawalAssets * totalSupply / totalAssets (rounding up)
func (v *VaultShareMath) PreviewWithdraw(assets *big.Int) *big.Int {
	if v.TotalAssets.Sign() == 0 || v.TotalSupply.Sign() == 0 {
		return new(big.Int).Set(assets)
	}
	// Round up to ensure vault is not drained
	numerator := new(big.Int).Mul(assets, v.TotalSupply)
	result := new(big.Int).Div(numerator, v.TotalAssets)
	// If there's a remainder, add 1 (round up)
	if new(big.Int).Rem(numerator, v.TotalAssets).Sign() > 0 {
		result.Add(result, big.NewInt(1))
	}
	return result
}

// PreviewRedeem returns the assets that would be returned for burning shares.
// Formula: assets = shares * totalAssets / totalSupply (rounding down)
func (v *VaultShareMath) PreviewRedeem(shares *big.Int) *big.Int {
	if v.TotalSupply.Sign() == 0 {
		return new(big.Int).Set(shares)
	}
	return new(big.Int).Div(
		new(big.Int).Mul(shares, v.TotalAssets),
		v.TotalSupply,
	)
}

// VaultFeeConfig represents fee configuration for an ERC-4626 vault.
type VaultFeeConfig struct {
	ManagementFeeBP  uint64         `json:"management_fee_bp"`  // annual management fee in basis points
	PerformanceFeeBP uint64         `json:"performance_fee_bp"` // performance fee in basis points
	EntryFeeBP       uint64         `json:"entry_fee_bp"`       // deposit fee in basis points
	ExitFeeBP        uint64         `json:"exit_fee_bp"`        // withdrawal fee in basis points
	FeeRecipient     common.Address `json:"fee_recipient"`
}

// CalculateEntryFee returns the entry fee for a deposit amount.
func (f *VaultFeeConfig) CalculateEntryFee(depositAmount *big.Int) *big.Int {
	if f.EntryFeeBP == 0 {
		return big.NewInt(0)
	}
	return new(big.Int).Div(
		new(big.Int).Mul(depositAmount, big.NewInt(int64(f.EntryFeeBP))),
		big.NewInt(10000),
	)
}

// CalculateExitFee returns the exit fee for a withdrawal amount.
func (f *VaultFeeConfig) CalculateExitFee(withdrawalAmount *big.Int) *big.Int {
	if f.ExitFeeBP == 0 {
		return big.NewInt(0)
	}
	return new(big.Int).Div(
		new(big.Int).Mul(withdrawalAmount, big.NewInt(int64(f.ExitFeeBP))),
		big.NewInt(10000),
	)
}

// ─── ERC-2981 NFT Royalty Standard ──────────────────────────────────────────

// RoyaltyInfo implements ERC-2981 royalty calculation.
type RoyaltyInfo struct {
	RoyaltyFraction  *big.Int       `json:"royalty_fraction"` // royalty in basis points (e.g., 250 = 2.5%)
	RoyaltyRecipient common.Address `json:"royalty_recipient"`
}

// NewRoyaltyInfo creates a new royalty config.
func NewRoyaltyInfo(recipient common.Address, basisPoints uint16) *RoyaltyInfo {
	return &RoyaltyInfo{
		RoyaltyFraction:  big.NewInt(int64(basisPoints)),
		RoyaltyRecipient: recipient,
	}
}

// CalculateRoyalty returns the royalty amount for a given sale price.
// ERC-2981: royalty = salePrice * royaltyFraction / 10000
func (r *RoyaltyInfo) CalculateRoyalty(salePrice *big.Int) *big.Int {
	if salePrice.Sign() <= 0 || r.RoyaltyFraction.Sign() <= 0 {
		return big.NewInt(0)
	}
	return new(big.Int).Div(
		new(big.Int).Mul(salePrice, r.RoyaltyFraction),
		big.NewInt(10000),
	)
}

// RoyaltyRecipientForToken returns the royalty recipient (per-token override support).
type RoyaltyManager struct {
	defaultRoyalty *RoyaltyInfo
	tokenRoyalties map[uint64]*RoyaltyInfo // tokenId → override
}

// NewRoyaltyManager creates a new royalty manager.
func NewRoyaltyManager(defaultRoyalty *RoyaltyInfo) *RoyaltyManager {
	return &RoyaltyManager{
		defaultRoyalty: defaultRoyalty,
		tokenRoyalties: make(map[uint64]*RoyaltyInfo),
	}
}

// SetTokenRoyalty sets a per-token royalty override.
func (rm *RoyaltyManager) SetTokenRoyalty(tokenID uint64, royalty *RoyaltyInfo) {
	rm.tokenRoyalties[tokenID] = royalty
}

// RoyaltyInfoForToken returns the royalty info for a specific token.
func (rm *RoyaltyManager) RoyaltyInfoForToken(tokenID uint64) *RoyaltyInfo {
	if r, ok := rm.tokenRoyalties[tokenID]; ok {
		return r
	}
	return rm.defaultRoyalty
}

// ─── ERC-721 / ERC-1155 Batch Operations ────────────────────────────────────

// BatchTransfer represents a batch of token transfers for ERC-1155.
type BatchTransfer struct {
	Operator common.Address `json:"operator"`
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	TokenIDs []*big.Int     `json:"token_ids"`
	Amounts  []*big.Int     `json:"amounts"`
	Data     []byte         `json:"data,omitempty"`
}

// Validate checks that the batch transfer is well-formed.
func (bt *BatchTransfer) Validate() error {
	if len(bt.TokenIDs) != len(bt.Amounts) {
		return fmt.Errorf("token_ids length (%d) != amounts length (%d)",
			len(bt.TokenIDs), len(bt.Amounts))
	}
	if len(bt.TokenIDs) == 0 {
		return fmt.Errorf("empty batch transfer")
	}
	for i, id := range bt.TokenIDs {
		if id == nil || id.Sign() < 0 {
			return fmt.Errorf("invalid token ID at index %d", i)
		}
	}
	for i, amt := range bt.Amounts {
		if amt == nil || amt.Sign() < 0 {
			return fmt.Errorf("invalid amount at index %d", i)
		}
	}
	return nil
}

// TotalValue computes the total number of tokens being transferred.
func (bt *BatchTransfer) TotalValue() *big.Int {
	total := big.NewInt(0)
	for _, amt := range bt.Amounts {
		total.Add(total, amt)
	}
	return total
}

// ERC721BatchMint represents a batch of ERC-721 mints.
// Each recipient gets exactly one NFT with a unique token ID.
type ERC721BatchMint struct {
	Recipients []common.Address `json:"recipients"`
	TokenIDs   []*big.Int       `json:"token_ids"`
	Contract   common.Address   `json:"contract"`
}

// Validate checks that the batch mint is well-formed.
func (bm *ERC721BatchMint) Validate() error {
	if len(bm.Recipients) != len(bm.TokenIDs) {
		return fmt.Errorf("recipients length (%d) != token_ids length (%d)",
			len(bm.Recipients), len(bm.TokenIDs))
	}
	if len(bm.Recipients) == 0 {
		return fmt.Errorf("empty batch mint")
	}

	seen := make(map[string]bool)
	for i, id := range bm.TokenIDs {
		if id == nil || id.Sign() < 0 {
			return fmt.Errorf("invalid token ID at index %d", i)
		}
		key := id.String()
		if seen[key] {
			return fmt.Errorf("duplicate token ID %s at index %d", key, i)
		}
		seen[key] = true
	}
	return nil
}
