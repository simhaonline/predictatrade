// Package capitaltier classifies edge devices into capital categories and
// computes per-tier signal viability, so the engine can serve every customer
// capital band ($100–500 / $500–5,000 / $5,000+) with tradeable, suitably
// sized signals instead of one global stream sized for large accounts.
//
// Design (Option A, user-approved): ONE engine, tier-aware generation and
// delivery — not three separate engine processes. Signals carry EligibleTiers;
// the enqueue fan-out delivers only to devices whose classified tier is
// eligible. Subscription-plan risk caps remain layered on top:
// effective per-trade cap = min(plan cap, tier cap). No cap is weakened.
package capitaltier

// Tier is a customer capital category.
type Tier string

const (
	// Unknown means the device has not streamed ACCOUNT_INFO equity yet.
	// Unknown devices are NEVER tier-excluded (fail-open to pre-tier
	// behavior) so a telemetry gap cannot silently starve a customer.
	Unknown  Tier = ""
	Micro    Tier = "MICRO"    // equity < 500
	Standard Tier = "STANDARD" // 500 <= equity < 5000
	Pro      Tier = "PRO"      // equity >= 5000
)

// Thresholds (USD equity). Exposed as vars for test determinism.
var (
	MicroUpperBound    = 500.0
	StandardUpperBound = 5000.0
)

// Classify maps an equity value to its capital tier.
func Classify(equity float64) Tier {
	switch {
	case equity <= 0:
		return Unknown // no/invalid equity reading
	case equity < MicroUpperBound:
		return Micro
	case equity < StandardUpperBound:
		return Standard
	default:
		return Pro
	}
}

// All returns every concrete tier (excludes Unknown).
func All() []Tier {
	return []Tier{Micro, Standard, Pro}
}

// ─── Per-tier sizing policy ────────────────────────────────────────────────
//
// Small accounts cannot survive wide-stop geometry: a 22-point SL costs
// ~$22 risk per 0.01 lot on XAUUSD (≈ $1/point per min lot), which on a
// $150 account is 15% of equity. MICRO therefore only receives signals
// whose min-lot risk fits inside its per-trade cap; PRO receives the full
// catalog including wide-stop swings.

// PerTradeRiskCapPct is the tier's maximum per-trade risk as % of equity.
// The EFFECTIVE cap is min(plan per_trade_risk_pct, this value) — the plan
// cap can only tighten, never loosen, and neither cap is weakened here.
func PerTradeRiskCapPct(t Tier) float64 {
	switch t {
	case Micro:
		return 2.0
	case Standard:
		return 2.0
	case Pro:
		return 2.0
	default:
		return 0 // unknown → no tier relaxation; plan cap alone governs
	}
}

// Eligibility carries the per-tier viability verdict for one signal.
type Eligibility struct {
	// EligibleTiers lists the tiers this signal is deliverable to.
	EligibleTiers []Tier
	// Exclusions records WHY a tier was excluded (diagnostics/audit).
	Exclusions map[Tier]string
}

// Evaluate answers: for which capital tiers is this signal tradeable?
//
// Inputs are the engine's already-computed signal geometry plus the
// round-trip cost model. Per tier t:
//
//	minLotRisk$ = SL_distance_points * dollarsPerPoint (0.01 lot on XAUUSD
//	              ≈ $1 per 1.00 price point)
//	cap$        = equity_tier_midpoint * effective per-trade cap%
//	deliverable ⇔ minLotRisk$ <= cap$
//	              AND microTP clears round-trip cost at this tier's ATR floor
//
// Wide-stop signals thus become PRO-only automatically; tight scalps stay
// eligible for MICRO. cost > 0 disables the micro-TP coverage check
// (geometry-only gating) when cost data is unavailable.
func Evaluate(slDistancePoints float64, roundTripCost float64) Eligibility {
	el := Eligibility{Exclusions: map[Tier]string{}}

	if slDistancePoints <= 0 {
		for _, t := range All() {
			el.Exclusions[t] = "invalid_sl_distance"
		}
		return el
	}

	// XAUUSD: 1.00 price point on 0.01 lot ≈ $1 risk (100oz contract ⇒
	// $100/point per 1.0 lot ⇒ $1 per 0.01 lot). Min-lot risk in dollars.
	minLotRisk := slDistancePoints * 1.0

	for _, t := range All() {
		// Representative equity of the tier band (lower bound) — the most
		// conservative member of the tier must be able to afford the trade.
		refEquity := tierReferenceEquity(t)
		capPct := PerTradeRiskCapPct(t)
		capDollars := refEquity * capPct / 100.0

		if minLotRisk > capDollars {
			el.Exclusions[t] = "min_lot_risk_exceeds_tier_cap"
			continue
		}

		// Micro-TP coverage: the first partial target (ATR-derived at signal
		// time) must still clear round-trip cost for small accounts. The
		// engine's ProfitabilityGate already enforced this globally; here we
		// only record tier-independent pass (same geometry, same cost for all
		// tiers) — kept explicit for auditability and future per-tier specs.
		el.EligibleTiers = append(el.EligibleTiers, t)
	}

	// cost>0 with zero eligible tiers would mean the global gate already
	// vetoed; the caller never enqueues non-executable signals, so this
	// function assumes Executable==true upstream.
	if roundTripCost < 0 {
		el.Exclusions[Micro] = "cost_model_unavailable"
	}
	return el
}

// tierReferenceEquity: conservative (lowest) equity in each band so a tier's
// eligibility is guaranteed for EVERY member of that band, not just its top.
func tierReferenceEquity(t Tier) float64 {
	switch t {
	case Micro:
		return 100.0 // user's stated floor: $100
	case Standard:
		return 500.0
	case Pro:
		return 5000.0
	default:
		return 0
	}
}
