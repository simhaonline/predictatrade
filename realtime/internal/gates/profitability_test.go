package gates

import (
	"testing"

	"github.com/predictatrade/realtime/internal/types"
)

// TestProfitabilityGate_VetoesLossCandidates verifies the delivery gate
// eliminates loss-making candidates (negative-EV flag or failed entry gate)
// and passes sound setups.
func TestProfitabilityGate_VetoesLossCandidates(t *testing.T) {
	g := NewProfitabilityGate()

	// 1) Strategy flagged as negative-EV → veto.
	in := GateInput{
		EntryPrice:         2430, StopLoss: 2426, TakeProfit1: 2435,
		RoundTripCost: 0.30, SignalScore: 70, Regime: types.RegimeTrendingBullish,
		RefinementProvided: true, IsLossCandidate: true,
	}
	if ev := g.Evaluate(in, GateState{}); ev.Result != types.GateVeto {
		t.Errorf("expected VETO for negative-EV candidate, got %s", ev.Result)
	}

	// 2) Strategy entry gate rejected → veto.
	in2 := in
	in2.IsLossCandidate = false
	in2.EntryGatePassed = false
	if ev := g.Evaluate(in2, GateState{}); ev.Result != types.GateVeto {
		t.Errorf("expected VETO for rejected entry gate, got %s", ev.Result)
	}

	// 3) Sound candidate → PASS.
	in3 := in2
	in3.EntryGatePassed = true
	if ev := g.Evaluate(in3, GateState{}); ev.Result != types.GatePass {
		t.Errorf("expected PASS for sound candidate, got %s (reasons=%v)", ev.Result, ev.ReasonCodes)
	}

	// 4) Refinement not provided → gate must not veto on zero defaults
	// (fail-open; relies on its own EV computation when a score is present).
	in4 := GateInput{EntryPrice: 2430, StopLoss: 2426, TakeProfit1: 2435, RoundTripCost: 0.30, SignalScore: 0}
	if ev := g.Evaluate(in4, GateState{}); ev.Result != types.GatePass {
		t.Errorf("expected PASS (fail-open) when refinement not provided, got %s", ev.Result)
	}
}
