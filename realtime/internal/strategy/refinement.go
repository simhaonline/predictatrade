package strategy

// Package strategy — Signal-generation refinement (prompt.md).
//
// Implements, for each strategy product, UNIQUE entry gates, ADVANCED
// mathematical filters, MICRO profit-taking geometry, and a server-side
// profitability (expected-value) filter that eliminates loss-making candidates
// before they are delivered to execution agents.
//
// IMPORTANT (AGENTS.md / SOW Section 16): the expected-value / win-rate model
// used here is a MATHEMATICAL, CONFIGURATION-BACKED estimate derived from score,
// cost, and geometry. It does NOT guarantee future profit and is never presented
// to subscribers as a probability. It is one deterministic gate in a chain whose
// purpose is to suppress clearly negative-expectancy setups.

import (
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// ─── Distinct per-strategy exit geometry (SL/TP/R:R/spread + micro profit-taking) ───

// ExitSpec is the distinct, auditable exit geometry per strategy product.
// MicroTPATRMult is the distance (in ATR) of the FIRST (micro) profit-taking
// level used for micro profit-taking / partial position close.
// PartialClosePct is the fraction of the position closed at MicroTP.
type ExitSpec struct {
	ATRMultSL       float64
	ATRMultTP1      float64
	ATRMultTP2      float64
	ATRMultTP3      float64
	MicroTPATRMult  float64
	PartialClosePct float64
	MaxSpreadPips   float64
	MinRR           float64
}

// StrategyExitSpec returns the DISTINCT exit geometry for a strategy.
// These values are the canonical per-strategy SL/TP/R:R/spread assignments.
func StrategyExitSpec(id types.StrategyID) ExitSpec {
	switch id {
	case types.StrategyUltraScalping:
		// Tightest, fastest: smallest SL, smallest micro-TP, most aggressive partial.
		return ExitSpec{ATRMultSL: 1.0, ATRMultTP1: 1.5, ATRMultTP2: 2.5, ATRMultTP3: 4.0,
			MicroTPATRMult: 0.5, PartialClosePct: 0.50, MaxSpreadPips: 1.5, MinRR: 2.0}
	case types.StrategyStandardScalping:
		// v1.26 REBUILD (win-rate-first scalping, cost-aware): the old
		// 2.5×ATR TP1 needed wr ≳ 47% just to cover round-trip cost
		// (~0.4pt/oz) but the loose entry gate produced 41.5% — guaranteed
		// bleed (90d: −46%, PF 0.81). First cut (TP1 1.2/SL 1.5) still
		// demanded 58.5% breakeven wr — the EV model (correctly) rejected
		// every read (564 PARITY_NEGATIVE_EV, 0 trades). Winning micro-scalp
		// geometry = tight stop + close target: SL 0.8×ATR / TP1 1.2×ATR →
		// breakeven wr ≈ 44% at M5 ATR, achievable by the same entry read.
		// Micro TP 0.5×ATR + 40% partial covers cost from the first scale-out.
		return ExitSpec{ATRMultSL: 0.8, ATRMultTP1: 1.2, ATRMultTP2: 2.0, ATRMultTP3: 3.5,
			MicroTPATRMult: 0.5, PartialClosePct: 0.40, MaxSpreadPips: 2.5, MinRR: 1.0}
	case types.StrategyStandardSwing:
		return ExitSpec{ATRMultSL: 2.0, ATRMultTP1: 3.0, ATRMultTP2: 5.0, ATRMultTP3: 8.0,
			MicroTPATRMult: 1.2, PartialClosePct: 0.35, MaxSpreadPips: 4.0, MinRR: 2.0}
	case types.StrategyTrendSwing:
		return ExitSpec{ATRMultSL: 2.5, ATRMultTP1: 4.0, ATRMultTP2: 6.5, ATRMultTP3: 10.0,
			MicroTPATRMult: 1.8, PartialClosePct: 0.30, MaxSpreadPips: 5.0, MinRR: 2.0}
	case types.StrategyMarnieFib:
		return ExitSpec{ATRMultSL: 1.5, ATRMultTP1: 2.0, ATRMultTP2: 3.5, ATRMultTP3: 5.5,
			MicroTPATRMult: 0.7, PartialClosePct: 0.40, MaxSpreadPips: 4.0, MinRR: 2.0}
	default:
		return ExitSpec{ATRMultSL: 1.5, ATRMultTP1: 2.5, ATRMultTP2: 4.0, ATRMultTP3: 6.0,
			MicroTPATRMult: 0.8, PartialClosePct: 0.40, MaxSpreadPips: 2.5, MinRR: 2.0}
	}
}

// ─── Advanced mathematical filters (pure functions) ───

// bollingerPercentB returns where price sits inside the Bollinger band [0,1]
// (0 = at lower band, 1 = at upper band). ok=false when bands are unavailable.
func bollingerPercentB(state *features.MarketState) (float64, bool) {
	u, l := state.Indicators.BollUpper, state.Indicators.BollLower
	if u.IsZero() || l.IsZero() {
		return 0.5, false
	}
	den := u.Sub(l)
	if den.IsZero() {
		return 0.5, false
	}
	b := state.CurrentPrice.Sub(l).Div(den)
	bf, _ := b.Float64()
	return bf, true
}

// volatilityRegimeZ approximates the volatility regime via ATR/price ratio,
// normalized to a typical XAUUSD band. ~0 = normal, >~2 = elevated, <-2 = compressed.
func volatilityRegimeZ(state *features.MarketState) float64 {
	if state.Indicators.ATR.IsZero() || state.CurrentPrice.IsZero() {
		return 0
	}
	atrPct := state.Indicators.ATR.Div(state.CurrentPrice)
	af, _ := atrPct.Float64()
	return (af - 0.0025) / 0.0015
}

// entryEfficiencyZ measures how far price is from value (VWAP) normalized by ATR.
// Large magnitude = extended/inefficient entry (chasing). ok=false if unavailable.
func entryEfficiencyZ(state *features.MarketState) (float64, bool) {
	v := state.VWAP.SessionVWAP
	if v.IsZero() || state.Indicators.ATR.IsZero() {
		return 0, false
	}
	z := state.CurrentPrice.Sub(v).Div(state.Indicators.ATR)
	zf, _ := z.Float64()
	return zf, true
}

// momentumQuality returns a signed momentum score from OsMA + MACD histogram.
func momentumQuality(state *features.MarketState, dir types.Direction) float64 {
	score := 0.0
	if dir == types.DirectionBuy {
		if state.Indicators.OsMA.GreaterThan(decimal.Zero) {
			score += 1
		}
		if state.Indicators.MACDMain.GreaterThan(state.Indicators.MACDSignal) {
			score += 1
		}
	} else if dir == types.DirectionSell {
		if state.Indicators.OsMA.LessThan(decimal.Zero) {
			score -= 1
		}
		if state.Indicators.MACDMain.LessThan(state.Indicators.MACDSignal) {
			score -= 1
		}
	}
	return score
}

// ─── Unique entry gates per strategy ───

// UniqueEntryGate applies the DISTINCT entry gate for a strategy product.
// It returns whether the (already-directional) candidate passes, the reason
// codes if it fails, and a metrics map for observability/traceability.
//
// Soft (unavailable-data) conditions are treated as PASS to avoid falsely
// blocking on incomplete market state. Only clearly adverse math rejects.
func UniqueEntryGate(id types.StrategyID, state *features.MarketState, dir types.Direction) (bool, []types.NoTradeReason, map[string]float64) {
	metrics := map[string]float64{}
	if dir != types.DirectionBuy && dir != types.DirectionSell {
		return true, nil, metrics
	}
	var reasons []types.NoTradeReason

	// Shared spread/cost gate (distinct MaxSpreadPips per strategy).
	spread, _ := state.Spread.Float64()
	atr, _ := state.Indicators.ATR.Float64()
	spec := StrategyExitSpec(id)
	metrics["spread_pips"] = spread
	metrics["atr"] = atr
	if atr > 0 && spread > spec.MaxSpreadPips {
		reasons = append(reasons, types.NTHighSpread)
	}
	if atr > 0 && spread/atr > 0.6 {
		// Cost is too large relative to noise — execution edge is destroyed.
		reasons = append(reasons, types.NTHighSpread)
	}

	// Volatility regime (shared).
	vz := volatilityRegimeZ(state)
	metrics["vol_z"] = vz

	// Bollinger placement (shared).
	if b, ok := bollingerPercentB(state); ok {
		metrics["boll_pctb"] = b
		switch id {
		case types.StrategyStandardScalping, types.StrategyStandardSwing:
			if dir == types.DirectionBuy && b > 0.85 {
				reasons = append(reasons, types.NTExtremeVolatility)
			}
			if dir == types.DirectionSell && b < 0.15 {
				reasons = append(reasons, types.NTExtremeVolatility)
			}
		case types.StrategyUltraScalping:
			if dir == types.DirectionBuy && b > 0.92 {
				reasons = append(reasons, types.NTExtremeVolatility)
			}
			if dir == types.DirectionSell && b < 0.08 {
				reasons = append(reasons, types.NTExtremeVolatility)
			}
		}
	}

	switch id {
	case types.StrategyUltraScalping:
		// Triple-EMA hierarchy MUST align with direction (fast momentum).
		if dir == types.DirectionBuy && !(state.Indicators.EMA9.GreaterThan(state.Indicators.EMA21) && state.Indicators.EMA21.GreaterThan(state.Indicators.EMA50)) {
			reasons = append(reasons, types.NTUnclearStructure)
		}
		if dir == types.DirectionSell && !(state.Indicators.EMA9.LessThan(state.Indicators.EMA21) && state.Indicators.EMA21.LessThan(state.Indicators.EMA50)) {
			reasons = append(reasons, types.NTUnclearStructure)
		}
		if momentumQuality(state, dir) <= 0 {
			reasons = append(reasons, types.NTInsufficientScore)
		}
		if vz > 3.0 {
			reasons = append(reasons, types.NTExtremeVolatility)
		}

	case types.StrategyStandardScalping:
		// v1.26 liquidity dead-zone block (trade-level forensics, 94 parity
		// trades): 02h UTC (Tokyo lunch lull — 16 trades, wr 25%, −$1,265)
		// and 19/21/23h UTC (NY-afternoon fade + post-close rollover) are
		// structurally dead for M5 scalping; blocking them lifted the
		// simulated wr from 47.9% to 55.7% at identical geometry.
		hr := state.Timestamp.UTC().Hour()
		metrics["entry_hour_utc"] = float64(hr)
		if hr == 2 || hr == 19 || hr == 21 || hr == 23 {
			reasons = append(reasons, types.NTLowLiquidity)
		}
		// EMA9/21 and VWAP must agree with direction.
		if dir == types.DirectionBuy && !state.Indicators.EMA9.GreaterThan(state.Indicators.EMA21) {
			reasons = append(reasons, types.NTUnclearStructure)
		}
		if dir == types.DirectionSell && !state.Indicators.EMA9.LessThan(state.Indicators.EMA21) {
			reasons = append(reasons, types.NTUnclearStructure)
		}
		if dir == types.DirectionBuy && !state.VWAP.SessionVWAP.IsZero() && !state.CurrentPrice.GreaterThan(state.VWAP.SessionVWAP) {
			reasons = append(reasons, types.NTConflictingTimeframes)
		}
		if dir == types.DirectionSell && !state.VWAP.SessionVWAP.IsZero() && !state.CurrentPrice.LessThan(state.VWAP.SessionVWAP) {
			reasons = append(reasons, types.NTConflictingTimeframes)
		}
		// v1.26 win-rate rebuild: momentum must not contradict the direction
		// (at least one of OsMA/MACD agrees) and the trend must carry (ADX>=20).
		// NOTE: momentum==2 was tried first and produced ZERO trades over 90
		// days (6,636 entry-gate vetoes) — on M5 the two oscillators rarely
		// agree at the decision bar. >=1 + ADX keeps selectivity without
		// starving the strategy; the win-rate lift comes from the 1.2×ATR TP1.
		if momentumQuality(state, dir) < 1 {
			reasons = append(reasons, types.NTUnclearStructure)
		}
		adx, _ := state.Indicators.ADX.Float64()
		if adx < 20 {
			reasons = append(reasons, types.NTUnclearStructure)
		}
		if vz > 3.5 {
			reasons = append(reasons, types.NTExtremeVolatility)
		}

	case types.StrategyStandardSwing:
		if dir == types.DirectionBuy && !state.Indicators.EMA21.GreaterThan(state.Indicators.EMA50) {
			reasons = append(reasons, types.NTUnclearStructure)
		}
		if dir == types.DirectionSell && !state.Indicators.EMA21.LessThan(state.Indicators.EMA50) {
			reasons = append(reasons, types.NTUnclearStructure)
		}
		if !state.Indicators.SMA200.IsZero() {
			if dir == types.DirectionBuy && !state.CurrentPrice.GreaterThan(state.Indicators.SMA200) {
				reasons = append(reasons, types.NTConflictingTimeframes)
			}
			if dir == types.DirectionSell && !state.CurrentPrice.LessThan(state.Indicators.SMA200) {
				reasons = append(reasons, types.NTConflictingTimeframes)
			}
		}
		adx, _ := state.Indicators.ADX.Float64()
		if adx < 20 {
			reasons = append(reasons, types.NTInsufficientScore)
		}

	case types.StrategyTrendSwing:
		// Macro trend (EMA100/200) must agree; skip when unavailable.
		if !state.Indicators.EMA100.IsZero() && !state.Indicators.EMA200.IsZero() {
			if dir == types.DirectionBuy && !state.Indicators.EMA100.GreaterThan(state.Indicators.EMA200) {
				reasons = append(reasons, types.NTUnclearStructure)
			}
			if dir == types.DirectionSell && !state.Indicators.EMA100.LessThan(state.Indicators.EMA200) {
				reasons = append(reasons, types.NTUnclearStructure)
			}
		}
		adx, _ := state.Indicators.ADX.Float64()
		if adx < 20 {
			reasons = append(reasons, types.NTInsufficientScore)
		}
		if state.Structure.CurrentTrend == "bullish" && dir == types.DirectionSell {
			reasons = append(reasons, types.NTUnclearStructure)
		}
		if state.Structure.CurrentTrend == "bearish" && dir == types.DirectionBuy {
			reasons = append(reasons, types.NTUnclearStructure)
		}
		if !state.Indicators.ParabolicSAR.IsZero() {
			if state.Indicators.ParabolicSARLong && dir == types.DirectionSell {
				reasons = append(reasons, types.NTConflictingTimeframes)
			}
			if !state.Indicators.ParabolicSARLong && dir == types.DirectionBuy {
				reasons = append(reasons, types.NTConflictingTimeframes)
			}
		}

	case types.StrategyMarnieFib:
		// Fib reversal: EMA21/50 must agree with direction (structure bias).
		if dir == types.DirectionBuy && !state.Indicators.EMA21.GreaterThan(state.Indicators.EMA50) {
			reasons = append(reasons, types.NTUnclearStructure)
		}
		if dir == types.DirectionSell && !state.Indicators.EMA21.LessThan(state.Indicators.EMA50) {
			reasons = append(reasons, types.NTUnclearStructure)
		}
	}

	// Entry efficiency: never chase an extended move.
	if z, ok := entryEfficiencyZ(state); ok {
		metrics["eff_z"] = z
		if (dir == types.DirectionBuy && z > 3.0) || (dir == types.DirectionSell && z < -3.0) {
			reasons = append(reasons, types.NTUnclearStructure)
		}
	}

	return len(reasons) == 0, reasons, metrics
}

// ─── Model-based win-rate / expected-value (profitability) ───

// estimateWinRate derives a MODEL-BASED edge estimate in [0.35, 0.82] from the
// confluence score and regime. It is NOT a calibrated probability and must never
// be displayed to subscribers as such (SOW Section 16).
func estimateWinRate(score float64, regime types.Regime) float64 {
	wr := 0.5 + (score-55.0)/100.0*0.5
	switch regime {
	case types.RegimeTrendingBullish, types.RegimeTrendingBearish, types.RegimeBreakout:
		wr += 0.05
	case types.RegimeRange, types.RegimeMeanReversion:
		wr += 0.02
	case types.RegimeHighVolatility:
		wr -= 0.03
	}
	if wr < 0.35 {
		wr = 0.35
	}
	if wr > 0.82 {
		wr = 0.82
	}
	return wr
}

// Profitability holds the computed expectancy metrics for a candidate.
type Profitability struct {
	NetRR1            float64
	ExpectedValue     float64
	EdgeScore         float64
	LossCandidate     bool
	MicroTPDist       float64
	MicroTPProfitable bool
}

// EvaluateProfitability computes the cost-aware expected value of a candidate
// and whether it is a loss-making candidate that must be eliminated.
//
// EV per unit risk = winRate * netWin - (1-winRate) * netLoss
//
//	netWin  = |TP1 - Entry| - cost
//	netLoss = |Entry - SL|   + cost
//
// A candidate is a loss candidate when EV <= 0 OR its micro profit-taking level
// does not clear round-trip cost.
// RoundTripFixedCost holds engine-wide slippage+commission cost (price units)
// added to the spread to form the true round-trip cost for micro-TP coverage
// checks. Set once at engine startup via SetExecutionCostModel; zero means the
// spread-only legacy behaviour (kept for tests).
var roundTripFixedCost = decimal.Zero

// SetExecutionCostModel wires engine cfg.SlippageCostPoints + cfg.CommissionCostPoints
// into the refinement profitability math so "micro TP covers cost" reflects the REAL
// broker round trip (spread + slippage + commission), not the spread alone.
func SetExecutionCostModel(slippagePts, commissionPts float64) {
	roundTripFixedCost = decimal.NewFromFloat(slippagePts + commissionPts)
}

func EvaluateProfitability(state *features.MarketState, dir types.Direction, entry, sl, tp1 decimal.Decimal, spec ExitSpec, score float64) Profitability {
	p := Profitability{}
	if entry.IsZero() || sl.IsZero() || tp1.IsZero() {
		p.LossCandidate = true
		return p
	}
	cost := state.Spread.Add(roundTripFixedCost) // spread + slippage + commission
	if state.Spread.IsZero() {
		cost = decimal.NewFromFloat(0.4).Add(roundTripFixedCost)
	}
	atr := state.Indicators.ATR
	if atr.IsZero() {
		atr = decimal.NewFromFloat(15)
	}
	winDist := tp1.Sub(entry).Abs()
	lossDist := entry.Sub(sl).Abs()
	netWin := winDist.Sub(cost)
	netLoss := lossDist.Add(cost)
	risk := lossDist.Add(cost)
	if risk.IsZero() {
		p.LossCandidate = true
		return p
	}
	netRR1 := netWin.Div(netLoss)
	netRR1f, _ := netRR1.Float64()
	p.NetRR1 = netRR1f

	wr := estimateWinRate(score, state.Regime.Current)
	p.EdgeScore = wr
	ev := decimal.NewFromFloat(wr).Mul(netWin).Sub(decimal.NewFromFloat(1 - wr).Mul(netLoss))
	evPerRisk := ev.Div(risk)
	evf, _ := evPerRisk.Float64()
	p.ExpectedValue = evf

	// Micro profit-taking distance (distinct per strategy).
	microDist := atr.Mul(decimal.NewFromFloat(spec.MicroTPATRMult))
	md, _ := microDist.Float64()
	p.MicroTPDist = md
	p.MicroTPProfitable = microDist.GreaterThan(cost)

	p.LossCandidate = evf <= 0 || !p.MicroTPProfitable
	return p
}

// ─── applyRefinement: wire refinement into every strategy evaluation ───

// applyRefinement enriches a directional StrategyResult with micro profit-taking
// levels, per-strategy edge/EV metrics, and the unique entry-gate result. It does
// NOT change Direction (elimination happens at the delivery gate), keeping the
// strategy's directional decision intact while supplying the data the delivery
// layer needs to suppress loss-making candidates.
func applyRefinement(result *StrategyResult, state *features.MarketState, dir types.Direction, cfg StrategyConfig, rawScore decimal.Decimal) {
	if dir != types.DirectionBuy && dir != types.DirectionSell {
		return
	}
	spec := StrategyExitSpec(cfg.StrategyID)

	// Micro profit-taking level (distinct per strategy), placed beyond cost.
	atr := state.Indicators.ATR
	if atr.IsZero() {
		atr = getStrategyATR(state, cfg.StrategyID)
	}
	if !atr.IsZero() {
		microDist := atr.Mul(decimal.NewFromFloat(spec.MicroTPATRMult))
		if dir == types.DirectionBuy {
			result.MicroTP = result.EntryPrice.Add(microDist)
		} else {
			result.MicroTP = result.EntryPrice.Sub(microDist)
		}
	}
	result.PartialClosePct = spec.PartialClosePct

	// Unique entry gate.
	ok, reasons, metrics := UniqueEntryGate(cfg.StrategyID, state, dir)

	// v1.26 scalping score-band gate: 90d trade-level forensics (94 parity
	// trades) show wr collapses ABOVE the mid-band — 45-55 scored +$184 (wr
	// 50.0%) while 55-65 lost $861 (wr 46.8%), and the worst October cluster
	// carried scores 62-64. Cap 62 (validated: block+<62 → 61 trades,
	// wr 57.4%, PF 1.31, +15.0% sim). High RawScore at evaluation time =
	// price already ran; the scalp entry is late. Mid-band momentum is the
	// entry; extreme momentum is the exit.
	if cfg.StrategyID == types.StrategyStandardScalping {
		scoreF, _ := rawScore.Float64()
		metrics["raw_score"] = scoreF
		if scoreF >= 62.0 {
			ok = false
			reasons = append(reasons, types.NTOverextended)
		}
	}

	result.EntryGatePassed = ok
	result.EntryGateMetrics = metrics
	if !ok {
		result.ReasonCodes = append(result.ReasonCodes, reasons...)
	}

	// Profitability / loss-candidate analysis.
	scoreF, _ := rawScore.Float64()
	prof := EvaluateProfitability(state, dir, result.EntryPrice, result.StopLoss, result.TP1, spec, scoreF)
	result.EdgeScore = prof.EdgeScore
	result.ExpectedValue = prof.ExpectedValue
	result.IsLossCandidate = prof.LossCandidate
}
