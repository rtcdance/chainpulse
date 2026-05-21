package defi

import (
	"math/big"
	"testing"
)

// ─── Constant Product AMM Tests ──────────────────────────────────────────────

func TestConstantProductAMMK(t *testing.T) {
	t.Parallel()
	amm := NewConstantProductAMM(big.NewInt(1000), big.NewInt(2000), 30)
	k := amm.K()
	expected := big.NewInt(2000000)
	if k.Cmp(expected) != 0 {
		t.Errorf("K() = %d, want %d", k, expected)
	}
}

func TestConstantProductAMMSpotPrice(t *testing.T) {
	t.Parallel()
	amm := NewConstantProductAMM(big.NewInt(1000), big.NewInt(2000), 30)
	price := amm.SpotPrice()
	// 2000/1000 = 2.0
	if price.Cmp(big.NewFloat(2.0)) != 0 {
		t.Errorf("SpotPrice() = %v, want 2.0", price)
	}
}

func TestConstantProductAMMSpotPriceZeroReserve(t *testing.T) {
	t.Parallel()
	amm := NewConstantProductAMM(big.NewInt(0), big.NewInt(2000), 30)
	price := amm.SpotPrice()
	if price.Cmp(big.NewFloat(0)) != 0 {
		t.Errorf("SpotPrice() with zero reserve should be 0, got %v", price)
	}
}

func TestConstantProductAMMAmountOut(t *testing.T) {
	t.Parallel()
	// Classic Uniswap v2 0.3% fee
	amm := NewConstantProductAMM(big.NewInt(1000000), big.NewInt(1000000), 30)

	// Swap 1000 token0 for token1
	amountIn := big.NewInt(1000)
	amountOut := amm.AmountOut(amountIn, amm.Reserve0, amm.Reserve1)

	// amountOut = 1000000 * 9970 * 1000 / (1000000 * 10000 + 9970 * 1000)
	// = 1000000 * 9970000 / (10000000000 + 9970000)
	// = 9970000000000 / 10009970000
	// ≈ 996
	if amountOut.Sign() <= 0 {
		t.Errorf("AmountOut should be positive, got %d", amountOut)
	}
	if amountOut.Cmp(big.NewInt(990)) < 0 || amountOut.Cmp(big.NewInt(999)) > 0 {
		t.Errorf("AmountOut = %d, expected ~996", amountOut)
	}
}

func TestConstantProductAMMAmountOutZero(t *testing.T) {
	t.Parallel()
	amm := NewConstantProductAMM(big.NewInt(1000), big.NewInt(2000), 30)

	tests := []struct {
		name       string
		amountIn   *big.Int
		reserveIn  *big.Int
		reserveOut *big.Int
	}{
		{"zero amount in", big.NewInt(0), big.NewInt(1000), big.NewInt(2000)},
		{"negative amount in", big.NewInt(-100), big.NewInt(1000), big.NewInt(2000)},
		{"zero reserve in", big.NewInt(100), big.NewInt(0), big.NewInt(2000)},
		{"zero reserve out", big.NewInt(100), big.NewInt(1000), big.NewInt(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := amm.AmountOut(tt.amountIn, tt.reserveIn, tt.reserveOut)
			if out.Sign() != 0 {
				t.Errorf("AmountOut should be 0, got %d", out)
			}
		})
	}
}

func TestConstantProductAMMPriceImpact(t *testing.T) {
	t.Parallel()
	amm := NewConstantProductAMM(big.NewInt(1000000), big.NewInt(1000000), 30)

	// Small swap should have small impact
	impact := amm.PriceImpact(big.NewInt(100), true)
	if impact < 0 || impact > 1 {
		t.Errorf("PriceImpact for small swap should be near 0, got %f", impact)
	}

	// Large swap should have significant impact
	impact = amm.PriceImpact(big.NewInt(500000), true)
	if impact < 20 {
		t.Errorf("PriceImpact for large swap should be significant, got %f", impact)
	}
}

func TestImpermanentLoss(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		priceRatio  float64
		wantNonZero bool
	}{
		{"no price change (ratio=1)", 1.0, false},
		{"price doubled", 2.0, true},
		{"price halved", 0.5, true},
		{"price 5x", 5.0, true},
		{"invalid (zero)", 0, false},
		{"invalid (negative)", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			il := ImpermanentLoss(tt.priceRatio)
			if tt.wantNonZero && il >= 0 {
				t.Errorf("ImpermanentLoss(%f) = %f, expected negative", tt.priceRatio, il)
			}
			if !tt.wantNonZero && il != 0 {
				t.Errorf("ImpermanentLoss(%f) = %f, expected 0", tt.priceRatio, il)
			}
		})
	}

	// Well-known IL: price doubles → IL ≈ -5.72%
	il := ImpermanentLoss(2.0)
	if il > -0.05 || il < -0.06 {
		t.Errorf("ImpermanentLoss(2.0) = %f, expected ≈ -0.057", il)
	}
}

// ─── Concentrated Liquidity Tests ────────────────────────────────────────────

func TestTickToPriceRoundTrip(t *testing.T) {
	t.Parallel()
	// Test that PriceToTick(TickToPrice(tick)) ≈ tick for small ticks
	for _, tick := range []int{0, 100, -100, 1000, -1000} {
		price := TickToPrice(tick)
		recoveredTick := PriceToTick(price)
		diff := recoveredTick - tick
		if diff < 0 {
			diff = -diff
		}
		if diff > 1 {
			t.Errorf("round trip failed: tick=%d → price=%f → tick=%d", tick, price, recoveredTick)
		}
	}
}

func TestTickToPriceZero(t *testing.T) {
	t.Parallel()
	price := TickToPrice(0)
	if price != 1.0 {
		t.Errorf("TickToPrice(0) = %f, want 1.0", price)
	}
}

func TestTickRange(t *testing.T) {
	t.Parallel()
	tr := TickRange{LowerTick: -1000, UpperTick: 1000}

	if !tr.IsInRange(0) {
		t.Error("tick 0 should be in range [-1000, 1000]")
	}
	if !tr.IsInRange(1000) {
		t.Error("tick 1000 should be in range [-1000, 1000] (inclusive)")
	}
	if tr.IsInRange(1001) {
		t.Error("tick 1001 should be out of range")
	}
	if tr.IsInRange(-1001) {
		t.Error("tick -1001 should be out of range")
	}
}

// ─── Lending Position (Health Factor) Tests ──────────────────────────────────

func TestLendingPositionCalculate(t *testing.T) {
	t.Parallel()
	lp := LendingPosition{
		TotalCollateralValue: big.NewFloat(10000), // $10k collateral
		TotalDebtValue:       big.NewFloat(5000),  // $5k debt
		LiquidationThreshold: 0.8,                 // 80% threshold
	}

	hf := lp.Calculate()
	// HF = (10000 * 0.8) / 5000 = 8000/5000 = 1.6
	expected := big.NewFloat(1.6)
	if hf.Cmp(expected) != 0 {
		t.Errorf("Calculate() = %v, want 1.6", hf)
	}
}

func TestLendingPositionNoDebt(t *testing.T) {
	t.Parallel()
	lp := LendingPosition{
		TotalCollateralValue: big.NewFloat(10000),
		TotalDebtValue:       big.NewFloat(0),
		LiquidationThreshold: 0.8,
	}

	hf := lp.Calculate()
	// mathMaxFloat64 is a very large number, not technically +Inf in big.Float
	if hf.Cmp(big.NewFloat(1e100)) < 0 {
		t.Errorf("no debt should give very large health factor, got %v", hf)
	}
}

func TestLendingPositionIsLiquidatable(t *testing.T) {
	t.Parallel()
	safe := LendingPosition{
		TotalCollateralValue: big.NewFloat(10000),
		TotalDebtValue:       big.NewFloat(5000),
		LiquidationThreshold: 0.8,
	}
	if safe.IsLiquidatable() {
		t.Error("HF=1.6 should not be liquidatable")
	}

	atRisk := LendingPosition{
		TotalCollateralValue: big.NewFloat(5000),
		TotalDebtValue:       big.NewFloat(5000),
		LiquidationThreshold: 0.8,
	}
	// HF = (5000 * 0.8) / 5000 = 0.8
	if !atRisk.IsLiquidatable() {
		t.Error("HF=0.8 should be liquidatable")
	}
}

func TestLendingPositionMaxBorrow(t *testing.T) {
	t.Parallel()
	lp := LendingPosition{
		TotalCollateralValue: big.NewFloat(10000),
		TotalDebtValue:       big.NewFloat(3000),
		LiquidationThreshold: 0.8,
	}

	maxBorrow := lp.MaxBorrow()
	// maxDebt = 10000 * 0.8 = 8000
	// maxAdditional = 8000 - 3000 = 5000
	expected := big.NewFloat(5000)
	if maxBorrow.Cmp(expected) != 0 {
		t.Errorf("MaxBorrow() = %v, want 5000", maxBorrow)
	}
}

func TestLendingPositionMaxBorrowAtLimit(t *testing.T) {
	t.Parallel()
	lp := LendingPosition{
		TotalCollateralValue: big.NewFloat(10000),
		TotalDebtValue:       big.NewFloat(9000),
		LiquidationThreshold: 0.8,
	}

	maxBorrow := lp.MaxBorrow()
	// maxDebt = 10000 * 0.8 = 8000, already > 9000 → 0
	if maxBorrow.Cmp(big.NewFloat(0)) != 0 {
		t.Errorf("MaxBorrow() should be 0 when already past threshold, got %v", maxBorrow)
	}
}

// ─── Liquidation Bonus Tests ────────────────────────────────────────────────

func TestLiquidationBonus(t *testing.T) {
	t.Parallel()
	lb := LiquidationBonus{
		IncentivePercent: 0.05, // 5% bonus
		CloseFactor:      0.5,  // can close 50% of debt
	}

	totalDebt := big.NewFloat(10000)
	maxLiq := lb.MaxLiquidation(totalDebt)
	if maxLiq.Cmp(big.NewFloat(5000)) != 0 {
		t.Errorf("MaxLiquidation() = %v, want 5000", maxLiq)
	}

	debtToRepay := big.NewFloat(5000)
	incentive := lb.LiquidationIncentive(debtToRepay)
	// 5000 * 1.05 = 5250
	if incentive.Cmp(big.NewFloat(5250)) != 0 {
		t.Errorf("LiquidationIncentive() = %v, want 5250", incentive)
	}
}
