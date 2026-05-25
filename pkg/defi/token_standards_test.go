package defi

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// ─── ERC-4626 Vault Tests ───────────────────────────────────────────────────

func TestVaultShareMathFirstDeposit(t *testing.T) {
	t.Parallel()
	v := NewVaultShareMath(big.NewInt(0), big.NewInt(0), 18)

	// First deposit: 1:1 ratio
	shares := v.PreviewDeposit(big.NewInt(1000))
	if shares.Cmp(big.NewInt(1000)) != 0 {
		t.Errorf("first deposit: PreviewDeposit(1000) = %d, want 1000", shares)
	}

	assets := v.PreviewMint(big.NewInt(500))
	if assets.Cmp(big.NewInt(500)) != 0 {
		t.Errorf("first mint: PreviewMint(500) = %d, want 500", assets)
	}
}

func TestVaultShareMathFirstRedeem(t *testing.T) {
	t.Parallel()
	v := NewVaultShareMath(big.NewInt(10000), big.NewInt(0), 18)
	assets := v.PreviewRedeem(big.NewInt(500))
	if assets.Cmp(big.NewInt(500)) != 0 {
		t.Errorf("first redeem: PreviewRedeem(500) = %d, want 500", assets)
	}
}

func TestVaultShareMathFirstWithdraw(t *testing.T) {
	t.Parallel()
	v := NewVaultShareMath(big.NewInt(0), big.NewInt(0), 18)
	shares := v.PreviewWithdraw(big.NewInt(1000))
	if shares.Cmp(big.NewInt(1000)) != 0 {
		t.Errorf("first withdraw: PreviewWithdraw(1000) = %d, want 1000", shares)
	}
}

func TestVaultShareMathSubsequentDeposit(t *testing.T) {
	// Vault has 10000 assets, 1000 shares (each share = 10 assets)
	v := NewVaultShareMath(big.NewInt(10000), big.NewInt(1000), 18)

	// Deposit 1000 assets → should get 100 shares
	shares := v.PreviewDeposit(big.NewInt(1000))
	if shares.Cmp(big.NewInt(100)) != 0 {
		t.Errorf("PreviewDeposit(1000) = %d, want 100", shares)
	}

	// Mint 50 shares → should need 500 assets
	assets := v.PreviewMint(big.NewInt(50))
	if assets.Cmp(big.NewInt(500)) != 0 {
		t.Errorf("PreviewMint(50) = %d, want 500", assets)
	}
}

func TestVaultShareMathWithdrawal(t *testing.T) {
	// 10000 assets, 1000 shares
	v := NewVaultShareMath(big.NewInt(10000), big.NewInt(1000), 18)

	// Withdraw 500 assets → burn 50 shares
	shares := v.PreviewWithdraw(big.NewInt(500))
	if shares.Cmp(big.NewInt(50)) != 0 {
		t.Errorf("PreviewWithdraw(500) = %d, want 50", shares)
	}

	// Redeem 100 shares → get 1000 assets
	assets := v.PreviewRedeem(big.NewInt(100))
	if assets.Cmp(big.NewInt(1000)) != 0 {
		t.Errorf("PreviewRedeem(100) = %d, want 1000", assets)
	}
}

func TestVaultShareMathRoundingUp(t *testing.T) {
	// 7 assets, 3 shares (not evenly divisible)
	v := NewVaultShareMath(big.NewInt(7), big.NewInt(3), 18)

	// Withdraw 5 assets: shares = 5 * 3 / 7 = 2.14... → round up to 3
	shares := v.PreviewWithdraw(big.NewInt(5))
	if shares.Cmp(big.NewInt(3)) != 0 {
		t.Errorf("PreviewWithdraw should round up: got %d, want 3", shares)
	}
}

func TestVaultFeeConfig(t *testing.T) {
	cfg := VaultFeeConfig{
		EntryFeeBP:   100, // 1%
		ExitFeeBP:    50,  // 0.5%
		FeeRecipient: common.HexToAddress("0x1234"),
	}

	// Entry fee on 10000 = 100
	entryFee := cfg.CalculateEntryFee(big.NewInt(10000))
	if entryFee.Cmp(big.NewInt(100)) != 0 {
		t.Errorf("entry fee = %d, want 100", entryFee)
	}

	// Exit fee on 10000 = 50
	exitFee := cfg.CalculateExitFee(big.NewInt(10000))
	if exitFee.Cmp(big.NewInt(50)) != 0 {
		t.Errorf("exit fee = %d, want 50", exitFee)
	}

	// Zero fees
	zeroCfg := VaultFeeConfig{}
	if f := zeroCfg.CalculateEntryFee(big.NewInt(10000)); f.Sign() != 0 {
		t.Errorf("zero entry fee should be 0, got %d", f)
	}
	if f := zeroCfg.CalculateExitFee(big.NewInt(10000)); f.Sign() != 0 {
		t.Errorf("zero exit fee should be 0, got %d", f)
	}
}

// ─── ERC-2981 Royalty Tests ─────────────────────────────────────────────────

func TestRoyaltyInfoCalculate(t *testing.T) {
	royalty := NewRoyaltyInfo(common.HexToAddress("0xabcd"), 250) // 2.5%

	// Sale at 1 ETH (1e18 wei) → royalty = 0.025 ETH
	salePrice := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	royaltyAmount := royalty.CalculateRoyalty(salePrice)

	expected := new(big.Int).Mul(big.NewInt(25), big.NewInt(1e15)) // 0.025 ETH
	if royaltyAmount.Cmp(expected) != 0 {
		t.Errorf("royalty = %d, want %d", royaltyAmount, expected)
	}
}

func TestRoyaltyInfoZeroPrice(t *testing.T) {
	royalty := NewRoyaltyInfo(common.HexToAddress("0xabcd"), 250)
	if r := royalty.CalculateRoyalty(big.NewInt(0)); r.Sign() != 0 {
		t.Errorf("zero price should give zero royalty, got %d", r)
	}
}

func TestRoyaltyManager(t *testing.T) {
	defaultRoyalty := NewRoyaltyInfo(common.HexToAddress("0x1111"), 500) // 5%
	rm := NewRoyaltyManager(defaultRoyalty)

	// Default royalty
	info := rm.RoyaltyInfoForToken(1)
	if info.RoyaltyFraction.Cmp(big.NewInt(500)) != 0 {
		t.Errorf("default royalty should be 500 BP, got %d", info.RoyaltyFraction)
	}

	// Override for token 42
	tokenRoyalty := NewRoyaltyInfo(common.HexToAddress("0x2222"), 1000) // 10%
	rm.SetTokenRoyalty(42, tokenRoyalty)

	info = rm.RoyaltyInfoForToken(42)
	if info.RoyaltyFraction.Cmp(big.NewInt(1000)) != 0 {
		t.Errorf("token 42 royalty should be 1000 BP, got %d", info.RoyaltyFraction)
	}

	// Other tokens still use default
	info = rm.RoyaltyInfoForToken(99)
	if info.RoyaltyFraction.Cmp(big.NewInt(500)) != 0 {
		t.Errorf("token 99 should use default 500 BP, got %d", info.RoyaltyFraction)
	}
}

// ─── ERC-1155 Batch Transfer Tests ──────────────────────────────────────────

func TestBatchTransferValidate(t *testing.T) {
	valid := &BatchTransfer{
		Operator: common.HexToAddress("0x1"),
		From:     common.HexToAddress("0x2"),
		To:       common.HexToAddress("0x3"),
		TokenIDs: []*big.Int{big.NewInt(1), big.NewInt(2)},
		Amounts:  []*big.Int{big.NewInt(10), big.NewInt(20)},
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("valid batch transfer should pass: %v", err)
	}

	// Mismatched lengths
	mismatched := &BatchTransfer{
		TokenIDs: []*big.Int{big.NewInt(1)},
		Amounts:  []*big.Int{big.NewInt(10), big.NewInt(20)},
	}
	if err := mismatched.Validate(); err == nil {
		t.Error("mismatched lengths should fail validation")
	}

	// Empty batch
	empty := &BatchTransfer{
		TokenIDs: []*big.Int{},
		Amounts:  []*big.Int{},
	}
	if err := empty.Validate(); err == nil {
		t.Error("empty batch should fail validation")
	}

	// Negative token ID
	negID := &BatchTransfer{
		TokenIDs: []*big.Int{big.NewInt(-1)},
		Amounts:  []*big.Int{big.NewInt(10)},
	}
	if err := negID.Validate(); err == nil {
		t.Error("negative token ID should fail validation")
	}

	// Nil token ID
	nilID := &BatchTransfer{
		TokenIDs: []*big.Int{nil},
		Amounts:  []*big.Int{big.NewInt(10)},
	}
	if err := nilID.Validate(); err == nil {
		t.Error("nil token ID should fail validation")
	}

	// Nil amount
	nilAmt := &BatchTransfer{
		TokenIDs: []*big.Int{big.NewInt(1)},
		Amounts:  []*big.Int{nil},
	}
	if err := nilAmt.Validate(); err == nil {
		t.Error("nil amount should fail validation")
	}

	// Negative amount
	negAmt := &BatchTransfer{
		TokenIDs: []*big.Int{big.NewInt(1)},
		Amounts:  []*big.Int{big.NewInt(-1)},
	}
	if err := negAmt.Validate(); err == nil {
		t.Error("negative amount should fail validation")
	}
}

func TestBatchTransferTotalValue(t *testing.T) {
	bt := &BatchTransfer{
		Amounts: []*big.Int{big.NewInt(10), big.NewInt(20), big.NewInt(30)},
	}
	total := bt.TotalValue()
	if total.Cmp(big.NewInt(60)) != 0 {
		t.Errorf("TotalValue() = %d, want 60", total)
	}
}

// ─── ERC-721 Batch Mint Tests ───────────────────────────────────────────────

func TestERC721BatchMintValidate(t *testing.T) {
	valid := &ERC721BatchMint{
		Recipients: []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")},
		TokenIDs:   []*big.Int{big.NewInt(1), big.NewInt(2)},
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("valid batch mint should pass: %v", err)
	}

	// Mismatched lengths
	mismatched := &ERC721BatchMint{
		Recipients: []common.Address{common.HexToAddress("0x1")},
		TokenIDs:   []*big.Int{big.NewInt(1), big.NewInt(2)},
	}
	if err := mismatched.Validate(); err == nil {
		t.Error("mismatched lengths should fail")
	}

	// Duplicate token IDs
	dupes := &ERC721BatchMint{
		Recipients: []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2")},
		TokenIDs:   []*big.Int{big.NewInt(42), big.NewInt(42)},
	}
	if err := dupes.Validate(); err == nil {
		t.Error("duplicate token IDs should fail")
	}

	// Empty batch
	empty := &ERC721BatchMint{
		Recipients: []common.Address{},
		TokenIDs:   []*big.Int{},
	}
	if err := empty.Validate(); err == nil {
		t.Error("empty batch mint should fail")
	}

	// Nil token ID
	nilID := &ERC721BatchMint{
		Recipients: []common.Address{common.HexToAddress("0x1")},
		TokenIDs:   []*big.Int{nil},
	}
	if err := nilID.Validate(); err == nil {
		t.Error("nil token ID should fail validation")
	}

	// Negative token ID
	negID := &ERC721BatchMint{
		Recipients: []common.Address{common.HexToAddress("0x1")},
		TokenIDs:   []*big.Int{big.NewInt(-1)},
	}
	if err := negID.Validate(); err == nil {
		t.Error("negative token ID should fail validation")
	}
}
