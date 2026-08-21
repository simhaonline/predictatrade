// Package gates — Capital Protection Engine (prompt.md Section 3).
//
// Enforces hard capital protection limits:
//   - Maximum daily loss: 5% of equity
//   - Maximum per-trade risk: 1% of equity
//   - Maximum total open risk: 5% of equity
//   - Minimum reward:risk: 1:2 overall
//
// Also implements position sizing using tick value / tick size for XAUUSD.
package gates

import (

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// CapitalProtectionConfig holds the hard capital protection limits.
// Phase 4: Two-stage halt system (soft halt at -4%, hard halt at -6%).
type CapitalProtectionConfig struct {
	SoftHaltLossPct     float64 // 4.0 — block new entries, let existing trades run
	HardHaltLossPct     float64 // 6.0 — emergency close all positions
	MaxPerTradeRiskPct  float64 // 1.0 — max risk per trade as % of equity
	MaxTotalOpenRiskPct float64 // 5.0 — max total open risk as % of equity
	MinRR               float64 // 2.0 — minimum reward:risk ratio
	MaxDailyLossPct     float64 // 6.0 — backward compat = HardHalt
}

// DefaultCapitalProtectionConfig returns the prompt-specified defaults.
func DefaultCapitalProtectionConfig() CapitalProtectionConfig {
	return CapitalProtectionConfig{
		SoftHaltLossPct:     4.0,
		HardHaltLossPct:     6.0,
		MaxDailyLossPct:     6.0,
		MaxPerTradeRiskPct:  1.0,
		MaxTotalOpenRiskPct: 5.0,
		MinRR:              2.0,
	}
}

type HaltLevel int

const (
	HaltNone HaltLevel = 0
	HaltSoft HaltLevel = 1
	HaltHard HaltLevel = 2
)

func CheckDailyLossStaged(dailyLossPct float64, cfg CapitalProtectionConfig) (HaltLevel, bool) {
	if dailyLossPct <= -cfg.HardHaltLossPct {
		return HaltHard, true
	}
	if dailyLossPct <= -cfg.SoftHaltLossPct {
		return HaltSoft, true
	}
	return HaltNone, false
}

func DynamicRiskTapering(equity decimal.Decimal, baseRiskPct float64) float64 {
	if equity.LessThan(decimal.NewFromInt(200)) {
		return 0.5
	}
	return baseRiskPct
}

// BrokerSymbolInfo holds broker-specific symbol economics for position sizing.
type BrokerSymbolInfo struct {
	TickValue decimal.Decimal // value of 1 tick per 1.00 lot (e.g. $1.00)
	TickSize  decimal.Decimal // minimum price increment (e.g. 0.01)
	LotStep   decimal.Decimal // minimum lot step (e.g. 0.01)
	MaxLot    decimal.Decimal // maximum lot size
	MinLot    decimal.Decimal // minimum lot size
}

// DefaultXAUSymbolInfo returns typical XAUUSD broker symbol info.
// These should be overridden with live broker values at runtime.
func DefaultXAUSymbolInfo() BrokerSymbolInfo {
	return BrokerSymbolInfo{
		TickValue: decimal.NewFromInt(1),    // $1.00 per tick per 1.00 lot
		TickSize:  decimal.NewFromFloat(0.01), // 0.01 price increment
		LotStep:   decimal.NewFromFloat(0.01),
		MaxLot:    decimal.NewFromFloat(100),
		MinLot:    decimal.NewFromFloat(0.01),
	}
}

// CalculatePositionSize computes lot size for 1% risk per trade.
//
// risk_amount = equity * 0.01
// point_value = tick_value / tick_size
// lots = risk_amount / (stop_distance_price * point_value)
//
// Result is rounded down to broker lot step.
func CalculatePositionSize(equity, stopDistancePrice decimal.Decimal, symbol BrokerSymbolInfo) decimal.Decimal {
	if equity.IsZero() || stopDistancePrice.IsZero() || symbol.TickSize.IsZero() {
		return decimal.Zero
	}

	riskAmount := equity.Mul(decimal.NewFromFloat(0.01)) // 1% risk
	pointValue := symbol.TickValue.Div(symbol.TickSize)
	lots := riskAmount.Div(stopDistancePrice.Mul(pointValue))

	// Round down to lot step (exact decimal arithmetic to avoid float precision issues)
	if !symbol.LotStep.IsZero() {
		lots = lots.Div(symbol.LotStep).Floor().Mul(symbol.LotStep)
	}

	// Clamp to broker limits
	if lots.LessThan(symbol.MinLot) {
		return decimal.Zero // Below minimum — reject
	}
	if lots.GreaterThan(symbol.MaxLot) {
		lots = symbol.MaxLot
	}
	return lots
}

// CheckDailyLoss verifies the daily loss has not exceeded the 5% limit.
// Returns true if trading should be halted.
func CheckDailyLoss(dailyLossPct, maxDailyLossPct float64) bool {
	return dailyLossPct <= -maxDailyLossPct
}

// CheckTotalOpenRisk verifies total open risk does not exceed 5% of equity.
// Returns true if the limit is exceeded (no new trades allowed).
func CheckTotalOpenRisk(totalOpenRisk, equity decimal.Decimal, maxPct float64) bool {
	if equity.IsZero() {
		return true
	}
	maxRisk := equity.Mul(decimal.NewFromFloat(maxPct / 100.0))
	return totalOpenRisk.GreaterThan(maxRisk)
}

// ─── Partial Close / Profit Locking Schedule (prompt.md Section 3.3) ───

// PartialCloseStage represents a stage in the partial close schedule.
type PartialCloseStage int

const (
	PartialCloseNone PartialCloseStage = iota
	PartialCloseTP1  // Close 50%, move SL to breakeven
	PartialCloseTP2  // Close 30%, move SL to TP1
	PartialCloseTP3  // Close remaining 20%, trail by 1.5*ATR
)

// PartialCloseAction describes the action to take at each TP stage.
type PartialCloseAction struct {
	Stage           PartialCloseStage
	ClosePercent    float64 // percentage of ORIGINAL position to close
	NewStopLoss     decimal.Decimal // new SL after this stage
	TrailATRMultiplier float64 // trailing stop multiplier (0 = no trail)
}

// BuildPartialCloseSchedule creates the profit-locking schedule per prompt.md Section 3.3.
//
// TP1 hit → Close 50%, SL → breakeven (entry price)
// TP2 hit → Close 30%, SL → TP1 price
// TP3 hit → Close remaining 20%, trail by 1.5*ATR
func BuildPartialCloseSchedule(entry, tp1, tp2, atr decimal.Decimal) []PartialCloseAction {
	return []PartialCloseAction{
		{
			Stage:        PartialCloseTP1,
			ClosePercent: 50.0,
			NewStopLoss:  entry, // breakeven
		},
		{
			Stage:        PartialCloseTP2,
			ClosePercent: 30.0,
			NewStopLoss:  tp1, // move SL to TP1
		},
		{
			Stage:           PartialCloseTP3,
			ClosePercent:    20.0,
			TrailATRMultiplier: 1.5, // trail by 1.5*ATR
		},
	}
}

// ─── Swap Protection (prompt.md Section 4.1) ───

// SwapCheckResult holds the result of a swap protection check.
type SwapCheckResult struct {
	Allowed            bool
	ReasonCode         string
	ExpectedSwapCost   decimal.Decimal
	EffectiveNetProfit decimal.Decimal
	NetRR              decimal.Decimal
}

// CheckSwapProtection evaluates whether a trade should be allowed given swap costs.
//
// For intraday strategies: close before rollover if negative swap.
// For swing strategies: include expected swap cost in R:R.
// Reject if effective_net_profit / SL_distance < 2.0.
func CheckSwapProtection(
	direction types.Direction,
	entry, stopLoss, takeProfit decimal.Decimal,
	swapRatePerLot, lots decimal.Decimal,
	expectedNights int,
	isIntraday bool,
) SwapCheckResult {
	result := SwapCheckResult{Allowed: true}

	// Determine swap rate based on direction
	var swapRate decimal.Decimal
	if direction == types.DirectionBuy {
		swapRate = swapRatePerLot // swap long
	} else {
		swapRate = swapRatePerLot // swap short (caller provides appropriate rate)
	}

	// If swap is zero or positive, no restriction
	if !swapRate.LessThan(decimal.Zero) {
		return result
	}

	// Calculate expected swap cost (negative swap = cost to holder)
	result.ExpectedSwapCost = swapRate.Abs().Mul(lots).Mul(decimal.NewFromInt(int64(expectedNights)))

	// For intraday strategies: close before rollover (no swap exposure)
	if isIntraday {
		// Intraday should close before rollover — no swap cost expected
		return result
	}

	// For swing strategies: include swap cost in R:R
	targetDistance := takeProfit.Sub(entry).Abs()
	stopDistance := entry.Sub(stopLoss).Abs()

	result.EffectiveNetProfit = targetDistance.Sub(result.ExpectedSwapCost)

	if stopDistance.IsZero() {
		result.Allowed = false
		result.ReasonCode = "ZERO_STOP_DISTANCE"
		return result
	}

	result.NetRR = result.EffectiveNetProfit.Div(stopDistance)

	// Reject if net R:R < 2.0
	if result.NetRR.LessThan(decimal.NewFromFloat(2.0)) {
		result.Allowed = false
		result.ReasonCode = "SWAP_ADJUSTED_RR_BELOW_MINIMUM"
		return result
	}

	return result
}

// ─── Slippage Protection (prompt.md Section 4.2) ───

// SlippageCheckResult holds the result of a slippage/spread check.
type SlippageCheckResult struct {
	Allowed    bool
	ReasonCode string
	Spread     float64
	MaxSpread  float64
}

// CheckSpreadSlippage verifies current spread does not exceed the strategy's max.
// spreadPoints is the current spread in points.
// maxSpreadPoints is the per-strategy maximum spread in points.
func CheckSpreadSlippage(spreadPoints, maxSpreadPoints float64) SlippageCheckResult {
	result := SlippageCheckResult{
		Allowed:   true,
		Spread:    spreadPoints,
		MaxSpread: maxSpreadPoints,
	}
	if spreadPoints > maxSpreadPoints {
		result.Allowed = false
		result.ReasonCode = "SPREAD_EXCEEDS_MAXIMUM"
	}
	return result
}
