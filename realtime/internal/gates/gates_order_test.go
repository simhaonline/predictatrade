package gates

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

// prompt.md Sections 16-17, 21: stale/degraded feed state must veto —
// the data-quality gate must never pass on a non-PASS state.
func TestDataQualityGateVetoesOnDegradedState(t *testing.T) {
	g := &DataQualityGate{}
	for _, state := range []types.GateResult{types.GateDegraded, types.GateVeto} {
		eval := g.Evaluate(GateInput{}, GateState{
			State:       state,
			EvaluatedAt: time.Now(),
			ValidUntil:  time.Now().Add(time.Minute),
		})
		if eval.Result != types.GateVeto {
			t.Errorf("state=%s: result=%s, want VETO", state, eval.Result)
		}
	}
}

func TestStopHuntAndMinATRGatesWiredWithStateSeeding(t *testing.T) {
	// stop_hunt_filter / min_atr are self-evaluating (they read ATR and
	// StructuralLow/High directly from GateInput). They sit in the canonical
	// order but MUST have a seeded state, otherwise EvaluateAll fails closed
	// with GATE_NOT_INITIALIZED before the refresh ticker fires.
	// SeedConservativeGateStates provides that seed (matching main.go B-04).
	r := NewRegistry()
	registerAllGates(r)
	registered := map[types.GateID]bool{}
	for _, g := range r.gates {
		registered[g.ID()] = true
	}
	if !registered[types.GateStopHuntFilter] || !registered[types.GateMinATR] {
		t.Fatal("stop_hunt/min_atr gates should stay registered")
	}
	if !containsID(r.order, types.GateStopHuntFilter) || !containsID(r.order, types.GateMinATR) {
		t.Fatalf("stop_hunt/min_atr gates are part of the canonical evaluation order")
	}

	// Without state seeding they (like every ordered gate) fail closed.
	in := GateInput{Tick: &types.Tick{Quality: types.QualityAuthoritative}, ATR: 3.0,
		SessionAllowed: true, NewsRisk: "LOW", EntryPrice: 2430, StopLoss: 2426,
		TakeProfit1: 2435, EntitlementOK: true, LicenseActive: true,
		ExecutionPermitted: true}
	if allPass, _, _ := r.EvaluateAll(in); allPass {
		t.Fatal("unseeded registry must fail closed")
	}

	// After seeding they evaluate from live input and pass valid geometry.
	setAllGateStatesPass(r)
	allPass, evals, veto := r.EvaluateAll(in)
	if !allPass || veto != nil {
		t.Fatalf("seeded registry should pass valid input, veto=%v", veto)
	}
	found := 0
	for _, e := range evals {
		if e.GateID == types.GateMinATR || e.GateID == types.GateStopHuntFilter {
			if e.Result != types.GatePass {
				t.Errorf("%s = %s, want PASS for ATR=3.0 with no structure", e.GateID, e.Result)
			}
			found++
		}
	}
	if found < 2 {
		t.Errorf("expected both self-evaluating gates in evaluations, found %d of %d evals", found, len(evals))
	}
}

func containsID(order []types.GateID, id types.GateID) bool {
	for _, o := range order {
		if o == id {
			return true
		}
	}
	return false
}
