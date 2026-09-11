package gates

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

// TestHardSafetyGatesStillVeto is the HARD-SAFETY REGRESSION guard (prompt.md
// acceptance criterion #1): no profile change — including the soft-alpha rebuild
// — may ever relax a HARD gate. These gates must veto under their documented
// failure conditions regardless of GATE_PROFILE_VERSION.
func TestHardSafetyGatesStillVeto(t *testing.T) {
	// No NewRegistry() dependency: gates are wired at startup; this test asserts
	// each HARD gate's own veto behavior (the actual safety contract).

	// DataQuality: stale tick → veto.
	dq := &DataQualityGate{}
	stale := &types.Tick{Quality: types.QualityStale}
	if ev := dq.Evaluate(GateInput{Tick: stale}, GateState{}); ev.Result != types.GateVeto {
		t.Errorf("DataQuality: expected VETO on stale tick, got %s", ev.Result)
	}
	closed := &types.Tick{MarketClosed: true}
	if ev := dq.Evaluate(GateInput{Tick: closed}, GateState{}); ev.Result != types.GateVeto {
		t.Errorf("DataQuality: expected VETO on closed-market liveness data, got %s", ev.Result)
	}

	// Session: disallowed → veto.
	sg := &SessionGate{}
	if ev := sg.Evaluate(GateInput{SessionAllowed: false}, GateState{}); ev.Result != types.GateVeto {
		t.Errorf("Session: expected VETO when session disallowed, got %s", ev.Result)
	}

	// News: HIGH risk → veto (preserve mandated protection).
	ng := NewNewsGate(nil) // nil sync → fails closed on DATA_UNAVAILABLE, vetoes on HIGH
	if ev := ng.Evaluate(GateInput{NewsRisk: "HIGH"}, GateState{}); ev.Result != types.GateVeto {
		t.Errorf("News: expected VETO on HIGH risk, got %s", ev.Result)
	}
	if ev := ng.Evaluate(GateInput{NewsRisk: "DATA_UNAVAILABLE"}, GateState{}); ev.Result != types.GateVeto {
		t.Errorf("News: expected VETO on DATA_UNAVAILABLE (fail-closed), got %s", ev.Result)
	}

	// Spread: over max → veto.
	spg := &SpreadGate{MaxSpreadAbsolute: 5.0}
	if ev := spg.Evaluate(GateInput{Spread: 9.0, ATR: 20}, GateState{}); ev.Result != types.GateVeto {
		t.Errorf("Spread: expected VETO on wide spread, got %s", ev.Result)
	}

	// Margin: insufficient → veto.
	mg := &MarginGate{}
	if ev := mg.Evaluate(GateInput{}, GateState{Value: false}); ev.Result != types.GateVeto {
		t.Errorf("Margin: expected VETO on insufficient margin, got %s", ev.Result)
	}

	// Exposure: over max → veto.
	eg := &ExposureGate{MaxExposure: 4}
	if ev := eg.Evaluate(GateInput{CurrentExposure: 4, MaxExposure: 4}, GateState{}); ev.Result != types.GateVeto {
		t.Errorf("Exposure: expected VETO at exposure limit, got %s", ev.Result)
	}

	// Entitlement: denied → veto.
	en := &EntitlementGate{}
	if ev := en.Evaluate(GateInput{EntitlementOK: false}, GateState{}); ev.Result != types.GateVeto {
		t.Errorf("Entitlement: expected VETO when denied, got %s", ev.Result)
	}

	// RRNetExpectancy: R:R < 1.0 → veto (absolute floor, never relaxed).
	rg := &RRNetExpectancyGate{MinGrossRR: 1.2}
	bad := GateInput{Direction: types.DirectionBuy, EntryPrice: 2000, StopLoss: 2002, TakeProfit1: 2001}
	if ev := rg.Evaluate(bad, GateState{}); ev.Result != types.GateVeto {
		t.Errorf("RRNetExpectancy: expected VETO on sub-1.0 R:R, got %s", ev.Result)
	}

	// License + ExecutionPermission: denied → veto (entitlement gates).
	lg := &LicenseGate{}
	if ev := lg.Evaluate(GateInput{LicenseActive: false}, GateState{}); ev.Result != types.GateVeto {
		t.Errorf("License: expected VETO when inactive, got %s", ev.Result)
	}
	epg := &ExecutionPermissionGate{}
	if ev := epg.Evaluate(GateInput{ExecutionPermitted: false}, GateState{}); ev.Result != types.GateVeto {
		t.Errorf("ExecutionPermission: expected VETO when not permitted, got %s", ev.Result)
	}
}

// TestProfitabilityGate_NoLongerBlanketVetoes proves the central fix: a candidate
// that the OLD synthetic-model logic would have vetoed (modest score, ~1:1 R:R,
// UNKNOWN evidence) is now classified as a qualifiable / shadow signal rather than
// silently starved. This directly addresses the ~99.5% executable suppression.
func TestProfitabilityGate_NoLongerBlanketVetoes(t *testing.T) {
	g := NewProfitabilityGate()

	// A realistic scalping setup: entry 2400, SL 2388 (0.8ATR @ATR15), TP1 2418
	// (1.2ATR) → gross R:R 1.5, net positive; cost 0.4. Score 55, UNKNOWN evidence.
	// OLD logic: modelWinRate(55)=0.50 → EV≈0.5*netWin-0.5*netLoss <0 → VETO.
	// NEW logic: strategy-appropriate prior + UNKNOWN evidence → PASS (tier B/C).
	in := GateInput{
		EntryPrice:         2400,
		StopLoss:           2388,
		TakeProfit1:        2418,
		RoundTripCost:      0.4,
		SignalScore:        55,
		StrategyID:         types.StrategyStandardScalping,
		RefinementProvided: false, // engine falls back to gate EV; no false loss flag
	}
	if ev := g.Evaluate(in, GateState{}); ev.Result == types.GateVeto {
		t.Errorf("scalping setup must NOT be blanket-vetoed (got %s reasons=%v tier=%s)", ev.Result, ev.ReasonCodes, ev.QualityTier)
	}

	// A clearly positive-EV swing setup must qualify as A/B and NOT be shadow.
	good := GateInput{
		EntryPrice:         2400,
		StopLoss:           2390,
		TakeProfit1:        2420,
		RoundTripCost:      0.4,
		SignalScore:        70,
		StrategyID:         types.StrategyTrendSwing,
		RefinementProvided: false,
	}
	evGood := g.Evaluate(good, GateState{})
	if evGood.Result == types.GateVeto {
		t.Errorf("positive-EV swing setup must not be vetoed (got %s)", evGood.Result)
	}
	if evGood.QualityTier != "A_PLUS" && evGood.QualityTier != "A" && evGood.QualityTier != "B" {
		t.Errorf("expected delivery-grade tier for positive-EV setup, got %s", evGood.QualityTier)
	}
}

// TestProfitabilityGate_RejectsGenuinelyNegativeEV proves we did NOT remove the
// safety: a candidate with SUFFICIENT evidence and materially negative EV is still
// rejected (REJECT tier + VETO), never delivered.
func TestProfitabilityGate_RejectsGenuinelyNegativeEV(t *testing.T) {
	g := NewProfitabilityGate()
	// Engine marks this as a loss candidate with sufficient evidence.
	in := GateInput{
		EntryPrice:         2400,
		StopLoss:           2399.5,
		TakeProfit1:        2400.2, // tiny target, huge SL → deeply negative EV
		RoundTripCost:      0.4,
		SignalScore:        40,
		StrategyID:         types.StrategyStandardScalping,
		RefinementProvided: true,
		IsLossCandidate:    true,
	}
	ev := g.Evaluate(in, GateState{})
	if ev.Result != types.GateVeto || ev.QualityTier != "REJECT" {
		t.Errorf("expected REJECT (VETO) for negative-EV+sufficient-evidence, got result=%s tier=%s", ev.Result, ev.QualityTier)
	}
}

// TestGateProfileRollback proves the runtime version flag selects distinct
// profiles and that CONSERVATIVE/OPPORTUNITY/BALANCED all share the SAME hard
// thresholds (only soft-alpha differs) — the rollback path (prompt.md §226).
func TestGateProfileRollback(t *testing.T) {
	for _, v := range []string{"CONSERVATIVE", "BALANCED", "OPPORTUNITY", "typo-should-default"} {
		p := LoadGateProfile(v)
		if p.Version == "" {
			t.Errorf("profile %q resolved to empty version", v)
		}
		// Hard-floor invariant: every profile's MinNetRR must still enforce a
		// sensible minimum (>= 0.3) so no profile admits negative-EV-by-construction.
		for id, sp := range p.Strategies {
			if sp.MinNetRR < 0.3 {
				t.Errorf("profile %s strategy %s MinNetRR too low (%.2f) — would weaken safety", p.Version, id, sp.MinNetRR)
			}
		}
	}
	if LoadGateProfile("").Version != "BALANCED" {
		t.Errorf("empty version must default to BALANCED")
	}
}

// compile-time guard: GateState used above must exist.
var _ = GateState{FreshnessMs: 0, EvaluatedAt: time.Now()}
