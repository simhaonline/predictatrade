// Package risk implements capital-protection math shared by the gate layer
// and the signal engine: position sizing, margin-aware lot caps, session
// P&L anchoring and forward-test edge statistics.
//
// All functions are pure (no I/O) except the Valkey-backed anchor store.
package risk

import "math"

const (
	// XAUUSD standard contract economics (AGENTS.md: never assume a universal
	// definition — these are fallback constants overridden by broker snapshot).
	DefaultContractSize = 100.0 // ounces per 1.00 lot
	DefaultTickValue    = 1.0   // $ per tick per 1.00 lot
	DefaultTickSize     = 0.01  // price increment per tick
	DefaultLotStep      = 0.01
	MinLot              = 0.01

	DefaultLeverage          = 500.0
	DefaultMaxMarginUsagePct = 30.0 // % of free margin usable by one candidate
)

// SymbolEconomics holds broker symbol spec data; zero fields fall back to
// XAUUSD defaults.
type SymbolEconomics struct {
	TickValue    float64
	TickSize     float64
	LotStep      float64
	ContractSize float64
}

// NormalizeEconomics fills zero fields with XAUUSD defaults.
func NormalizeEconomics(e SymbolEconomics) SymbolEconomics {
	if e.TickValue <= 0 {
		e.TickValue = DefaultTickValue
	}
	if e.TickSize <= 0 {
		e.TickSize = DefaultTickSize
	}
	if e.LotStep <= 0 {
		e.LotStep = DefaultLotStep
	}
	if e.ContractSize <= 0 {
		e.ContractSize = DefaultContractSize
	}
	return e
}

// RiskPerLot returns the $ loss per 1.00 lot for a given stop distance
// (price units): dist / tickSize * tickValue.
func RiskPerLot(stopDistance float64, e SymbolEconomics) float64 {
	e = NormalizeEconomics(e)
	if stopDistance <= 0 {
		return 0
	}
	return stopDistance / e.TickSize * e.TickValue
}

// FloorToStep rounds v DOWN to the nearest multiple of step.
func FloorToStep(v, step float64) float64 {
	if step <= 0 {
		return v
	}
	return math.Floor(v/step+1e-9) * step
}

// SuggestedLot computes recommended_lot = floor((equity×pct)/riskPerLot)
// rounded DOWN to the lot step; returns 0 when below the minimum lot
// (account too small for this stop distance).
func SuggestedLot(equity, riskPctOfEquity, stopDistance float64, e SymbolEconomics) float64 {
	e = NormalizeEconomics(e)
	rpl := RiskPerLot(stopDistance, e)
	if equity <= 0 || riskPctOfEquity <= 0 || rpl <= 0 {
		return 0
	}
	maxRiskDollars := equity * riskPctOfEquity / 100.0
	lots := FloorToStep(maxRiskDollars/rpl, e.LotStep)
	if lots < MinLot {
		return 0
	}
	return lots
}

// RiskDollars returns the $ risk of `lot` at the given stop distance.
func RiskDollars(lot, stopDistance float64, e SymbolEconomics) float64 {
	return RiskPerLot(stopDistance, e) * lot
}

// MarginCheck is the result of a margin-aware lot cap evaluation.
type MarginCheck struct {
	Allowed        bool    `json:"allowed"`
	Reason         string  `json:"reason,omitempty"`
	RequiredMargin float64 `json:"required_margin"`
	MarginBudget   float64 `json:"margin_budget"` // freeMargin × maxUsagePct
	CappedLot      float64 `json:"capped_lot"`    // largest lot within budget (0 if none)
	Equity         float64 `json:"equity"`
}

// MarginAwareLotCap validates that requiredMargin = lot×100×price/leverage
// fits within freeMargin×30% (MA1-MA2 defaults). Use MarginAwareLotCapWith
// for broker-snapshot leverage/margin-policy overrides.
func MarginAwareLotCap(equity, freeMargin, lot, price, leverage float64) MarginCheck {
	return MarginAwareLotCapWith(equity, freeMargin, lot, price, leverage, DefaultMaxMarginUsagePct, SymbolEconomics{})
}

// MarginAwareLotCapWith is the fully parameterized variant.
func MarginAwareLotCapWith(equity, freeMargin, lot, price, leverage, maxMarginUsagePct float64, e SymbolEconomics) MarginCheck {
	e = NormalizeEconomics(e)
	result := MarginCheck{Equity: equity}
	if leverage <= 0 {
		leverage = DefaultLeverage
	}
	if maxMarginUsagePct <= 0 {
		maxMarginUsagePct = DefaultMaxMarginUsagePct
	}
	result.MarginBudget = freeMargin * maxMarginUsagePct / 100.0
	result.RequiredMargin = lot * e.ContractSize * price / leverage

	if result.RequiredMargin <= 0 || result.MarginBudget <= 0 {
		result.Allowed = false
		result.Reason = "margin_state_unknown"
		return result
	}
	if result.RequiredMargin > result.MarginBudget {
		result.Allowed = false
		result.Reason = "margin_exceeded"
		result.CappedLot = FloorToStep(result.MarginBudget*leverage/(e.ContractSize*price), e.LotStep)
		if result.CappedLot < MinLot {
			result.CappedLot = 0
		}
		return result
	}
	result.Allowed = true
	result.CappedLot = lot
	return result
}

// SizingResult aggregates the sizing annotations attached to a Signal.
type SizingResult struct {
	SLDistancePoints float64     `json:"sl_distance_points"`
	RequestedLot     float64     `json:"requested_lot"`
	SuggestedLot     float64     `json:"suggested_lot"`
	RiskDollars      float64     `json:"risk_dollars"`
	RiskPctOfEquity  float64     `json:"risk_pct_of_equity"`
	Oversize         bool        `json:"oversize"`
	VetoOversize     bool        `json:"veto_oversize"` // oversize AND no viable suggested lot
	Margin           MarginCheck `json:"margin"`
}

// ComputeSizing performs the full R1/R7 sizing computation for a candidate.
func ComputeSizing(equity, riskPctOfEquity, entry, sl, requestedLot float64, e SymbolEconomics) SizingResult {
	res := SizingResult{RequestedLot: requestedLot}
	if entry <= 0 || sl <= 0 {
		res.VetoOversize = true
		return res
	}
	res.SLDistancePoints = abs(entry - sl)
	rpl := RiskPerLot(res.SLDistancePoints, e)
	if rpl <= 0 || equity <= 0 {
		res.VetoOversize = true
		return res
	}
	res.SuggestedLot = SuggestedLot(equity, riskPctOfEquity, res.SLDistancePoints, e)
	res.RiskDollars = rpl * requestedLot
	res.RiskPctOfEquity = res.RiskDollars / equity * 100.0
	res.Oversize = res.RiskDollars > equity*riskPctOfEquity/100.0
	// Veto only when over cap AND the account cannot trade any viable lot
	// for this stop distance (R1: account too small).
	res.VetoOversize = res.Oversize && res.SuggestedLot < MinLot
	return res
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
