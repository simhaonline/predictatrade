// execution_risk.go — P3 (prompt.md): execution/risk EA-contract parity.
//
// New, pure functions (existing risk/sizing.go, recovery manager, capital
// gates untouched — overlap avoided by design):
//
//	MaxSlUSDCap      — hard cap on SL distance (reference InpMaxSlUSD=8);
//	                   cap ≤ 0 disables (documented pass-through).
//	SLFloorPoints    — broker-constraint minimum distance:
//	                   max((stopsLevel+2)×point, spread×1.5) — reference
//	                   BuildTradePlan's minStopPrice + spread buffer.
//	EnforceSLFloor   — widens an SL to the floor if thinner (BUY/SELL
//	                   agnostic: works on distances, caller keeps the side).
//	ShouldFlattenFriday — reference FridayCloseHour: after the cutoff hour on
//	                   Friday, block new entries / flatten. Cutoff 0 disables.
//
// Complementarity (no overlap):
//   - MaxSlUSDCap narrows; SLFloor widens. Order: floor first, then cap —
//     if the floor exceeds the cap the CAP WINS (capital protection beats
//     broker-constraint widening; documented in the wiring below).
//   - Recovery sizing (internal/recovery) and profit-lock (ProfitTargetGate)
//     already exist — NOT re-implemented here.
package risk

import (
	"time"

	"github.com/shopspring/decimal"
)

// MaxSlUSDCap caps the SL distance (abs(entry−sl)) at maxSLUSD.
// maxSLUSD ≤ 0 disables the cap (pass-through, clamped=false).
// Zero/negative SL returns the entry distance unchanged (invalid geometry is
// the caller's problem; this function never fabricates a side).
func MaxSlUSDCap(entry, sl, maxSLUSD float64) (dist float64, clamped bool) {
	if maxSLUSD <= 0 || sl <= 0 {
		return erAbs(entry - sl), false
	}
	d := entry - sl
	if d < 0 {
		d = -d
	}
	if d <= maxSLUSD {
		return d, false
	}
	return maxSLUSD, true
}

func erAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// SLFloorPoints computes the broker-constraint minimum SL distance:
// max((stopsLevelPoints+2)×point, spread×1.5).
func SLFloorPoints(entry, spread float64, stopsLevelPoints int, point float64) float64 {
	stops := float64(stopsLevelPoints+2) * point
	spreadBuf := spread * 1.5
	if stops > spreadBuf {
		return stops
	}
	return spreadBuf
}

// EnforceSLFloor widens the SL (moved further from entry, preserving the
// caller's side) so its distance meets the floor. Already-compliant SLs pass
// through unchanged.
func EnforceSLFloor(entry, sl, floor float64) float64 {
	dist := erAbs(entry - sl)
	if dist >= floor {
		return sl
	}
	if sl > entry {
		return entry + floor // SELL: SL above entry
	}
	return entry - floor // BUY: SL below entry
}

// ShouldFlattenFriday: true when t is Friday at/after cutoffHour (UTC).
// cutoffHour ≤ 0 disables the rule (returns false).
func ShouldFlattenFriday(t time.Time, cutoffHourUTC int) bool {
	if cutoffHourUTC <= 0 {
		return false
	}
	if t.Weekday() != time.Friday {
		return false
	}
	return t.Hour() >= cutoffHourUTC
}

// MaxSlUSDConfigured is a decimal convenience wrapper for geometry call sites
// that work in decimal (computeEntrySLTP).
func MaxSlUSDCapDec(entry, sl decimal.Decimal, maxSLUSD float64) (dist decimal.Decimal, clamped bool) {
	d, c := MaxSlUSDCap(
		mustF(entry), mustF(sl), maxSLUSD)
	return decimal.NewFromFloat(d), c
}

func mustF(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}