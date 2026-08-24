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

func TestStopHuntAndMinATRGatesNotInOrderWithoutStateSeeding(t *testing.T) {
	// stop_hunt_filter / min_atr are registered but intentionally NOT in the
	// evaluation order: they have no state seeding, and EvaluateAll fails
	// closed (GATE_NOT_INITIALIZED → veto) which would block ALL signals.
	// Min-ATR is enforced per-strategy in strategy/engines instead.
	// Wiring them requires state hydration first — tracked as a gap.
	r := NewRegistry()
	r.Register(&StopHuntFilterGate{MinDistanceATR: 1.5})
	r.Register(&MinAbsoluteATRGate{MinATR: 2.0})
	registered := map[types.GateID]bool{}
	for _, g := range r.gates {
		registered[g.ID()] = true
	}
	if !registered[types.GateStopHuntFilter] || !registered[types.GateMinATR] {
		t.Fatal("stop_hunt/min_atr gates should stay registered")
	}
	for _, id := range r.order {
		if id == types.GateStopHuntFilter || id == types.GateMinATR {
			t.Errorf("%s must not be evaluated until state seeding exists", id)
		}
	}
}
