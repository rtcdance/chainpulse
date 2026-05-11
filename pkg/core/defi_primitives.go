package core

import (
	"math/big"
)

// ─── Constant Product AMM (Uniswap v2) ───────────────────────────────────────

// ConstantProductAMM implements the x * y = k invariant for Uniswap v2-style pools.
type ConstantProductAMM struct {
	Reserve0 *big.Int `json:"reserve0"` // token0 reserve
	Reserve1 *big.Int `json:"reserve1"` // token1 reserve
	FeeRate  int64    `json:"fee_rate"` // in basis points (e.g., 30 = 0.3%)
}

// NewConstantProductAMM creates a new AMM pool.
func NewConstantProductAMM(reserve0, reserve1 *big.Int, feeBPS int64) *ConstantProductAMM {
	return &ConstantProductAMM{
		Reserve0: reserve0,
		Reserve1: reserve1,
		FeeRate:  feeBPS,
	}
}

// K returns the constant product k = reserve0 * reserve1.
func (amm *ConstantProductAMM) K() *big.Int {
	return new(big.Int).Mul(amm.Reserve0, amm.Reserve1)
}

// SpotPrice returns the price of token0 in terms of token1 (reserve1/reserve0).
func (amm *ConstantProductAMM) SpotPrice() *big.Float {
	if amm.Reserve0.Sign() == 0 {
		return big.NewFloat(0)
	}
	r0 := new(big.Float).SetInt(amm.Reserve0)
	r1 := new(big.Float).SetInt(amm.Reserve1)
	return new(big.Float).Quo(r1, r0)
}

// AmountOut calculates the output amount for a given input amount.
// Formula: amountOut = reserveOut * feeMultiplier * amountIn / (reserveIn + feeMultiplier * amountIn)
func (amm *ConstantProductAMM) AmountOut(amountIn, reserveIn, reserveOut *big.Int) *big.Int {
	if amountIn.Sign() <= 0 || reserveIn.Sign() <= 0 || reserveOut.Sign() <= 0 {
		return big.NewInt(0)
	}

	// Apply fee: amountInWithFee = amountIn * (10000 - feeRate)
	feeMultiplier := big.NewInt(10000 - amm.FeeRate)
	amountInWithFee := new(big.Int).Mul(amountIn, feeMultiplier)

	// numerator = reserveOut * amountInWithFee
	numerator := new(big.Int).Mul(reserveOut, amountInWithFee)

	// denominator = reserveIn * 10000 + amountInWithFee
	denominator := new(big.Int).Add(
		new(big.Int).Mul(reserveIn, big.NewInt(10000)),
		amountInWithFee,
	)

	return new(big.Int).Div(numerator, denominator)
}

// PriceImpact calculates the price impact of a swap as a percentage.
// Impact = 1 - (actualPrice / spotPrice) * 100
func (amm *ConstantProductAMM) PriceImpact(amountIn *big.Int, zeroForOne bool) float64 {
	var reserveIn, reserveOut *big.Int
	if zeroForOne {
		reserveIn, reserveOut = amm.Reserve0, amm.Reserve1
	} else {
		reserveIn, reserveOut = amm.Reserve1, amm.Reserve0
	}

	spotPrice := new(big.Float).Quo(
		new(big.Float).SetInt(reserveOut),
		new(big.Float).SetInt(reserveIn),
	)

	amountOut := amm.AmountOut(amountIn, reserveIn, reserveOut)
	if amountOut.Sign() == 0 {
		return 0
	}

	actualPrice := new(big.Float).Quo(
		new(big.Float).SetInt(amountOut),
		new(big.Float).SetInt(amountIn),
	)

	if spotPrice.Cmp(big.NewFloat(0)) == 0 {
		return 0
	}

	ratio := new(big.Float).Quo(actualPrice, spotPrice)
	impact := new(big.Float).Sub(big.NewFloat(1), ratio)
	result, _ := impact.Float64()
	return result * 100
}

// ImpermanentLoss calculates the impermanent loss for a liquidity provider
// given the price ratio (current price / entry price).
// IL = 2 * sqrt(priceRatio) / (1 + priceRatio) - 1
func ImpermanentLoss(priceRatio float64) float64 {
	if priceRatio <= 0 {
		return 0
	}
	sqrtRatio := sqrt(priceRatio)
	return 2*sqrtRatio/(1+priceRatio) - 1
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 100; i++ {
		z = (z + x/z) / 2
		if abs(z*z-x) < 1e-10 {
			break
		}
	}
	return z
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// ─── Concentrated Liquidity (Uniswap v3) ────────────────────────────────────

// TickRange represents a Uniswap v3 liquidity position's tick range.
type TickRange struct {
	LowerTick int `json:"lower_tick"`
	UpperTick int `json:"upper_tick"`
}

// TickToPrice converts a Uniswap v3 tick to a price.
// Formula: price = 1.0001^tick
func TickToPrice(tick int) float64 {
	price := 1.0001
	result := 1.0
	absTick := tick
	if absTick < 0 {
		absTick = -absTick
	}
	for i := 0; i < absTick; i++ {
		result *= price
	}
	if tick < 0 {
		return 1.0 / result
	}
	return result
}

// PriceToTick converts a price to the nearest Uniswap v3 tick.
func PriceToTick(price float64) int {
	if price <= 0 {
		return 0
	}
	// tick = log(price) / log(1.0001)
	tick := 0
	current := 1.0
	if price >= 1.0 {
		for current*1.0001 <= price {
			current *= 1.0001
			tick++
		}
	} else {
		for current/1.0001 >= price {
			current /= 1.0001
			tick--
		}
	}
	return tick
}

// IsInRange checks if the current tick falls within a position's range.
func (tr *TickRange) IsInRange(currentTick int) bool {
	return currentTick >= tr.LowerTick && currentTick <= tr.UpperTick
}

// ─── Lending Health Factor ──────────────────────────────────────────────────

// LendingPosition tracks the collateral/debt health factor for a lending position.
// HF > 1.0 = safe, HF = 1.0 = at liquidation threshold, HF < 1.0 = liquidatable
type LendingPosition struct {
	TotalCollateralValue *big.Float `json:"total_collateral_value"` // in USD
	TotalDebtValue       *big.Float `json:"total_debt_value"`       // in USD
	LiquidationThreshold float64    `json:"liquidation_threshold"`  // e.g., 0.8 = 80%
}

// Calculate returns the health factor value.
// HF = (totalCollateral * liquidationThreshold) / totalDebt
func (lp *LendingPosition) Calculate() *big.Float {
	if lp.TotalDebtValue.Cmp(big.NewFloat(0)) == 0 {
		return big.NewFloat(mathMaxFloat64) // no debt → infinite health
	}

	adjustedCollateral := new(big.Float).Mul(
		lp.TotalCollateralValue,
		big.NewFloat(lp.LiquidationThreshold),
	)
	return new(big.Float).Quo(adjustedCollateral, lp.TotalDebtValue)
}

// IsLiquidatable returns true if the health factor is below 1.0.
func (lp *LendingPosition) IsLiquidatable() bool {
	return lp.Calculate().Cmp(big.NewFloat(1.0)) < 0
}

// MaxBorrow calculates the maximum additional borrow amount
// before reaching the liquidation threshold.
func (lp *LendingPosition) MaxBorrow() *big.Float {
	maxDebt := new(big.Float).Mul(
		lp.TotalCollateralValue,
		big.NewFloat(lp.LiquidationThreshold),
	)
	currentDebt := lp.TotalDebtValue
	maxAdditional := new(big.Float).Sub(maxDebt, currentDebt)
	if maxAdditional.Cmp(big.NewFloat(0)) < 0 {
		return big.NewFloat(0)
	}
	return maxAdditional
}

// LiquidationBonus returns the typical liquidation bonus (e.g., 5-10%).
type LiquidationBonus struct {
	IncentivePercent float64 `json:"incentive_percent"` // e.g., 0.05 = 5%
	CloseFactor      float64 `json:"close_factor"`      // e.g., 0.5 = 50% of debt can be closed
}

// MaxLiquidation calculates the maximum amount of debt that can be liquidated.
func (lb *LiquidationBonus) MaxLiquidation(totalDebt *big.Float) *big.Float {
	return new(big.Float).Mul(totalDebt, big.NewFloat(lb.CloseFactor))
}

// LiquidationIncentive calculates the bonus collateral the liquidator receives.
func (lb *LiquidationBonus) LiquidationIncentive(debtToRepay *big.Float) *big.Float {
	return new(big.Float).Mul(debtToRepay, big.NewFloat(1.0+lb.IncentivePercent))
}

const mathMaxFloat64 = 1.7976931348623157e+308
