// Tests for per-(strategy, timeframe) gate-state isolation.
//
// These protect the capital-protection fix: each strategy on each timeframe now
// carries its OWN cached gate state, so a stale/errored/missing scope for one
// strategy can NEVER veto or degrade an unrelated strategy (the old central-gate
// "block everything" risk), and ATR/threshold gates are evaluated per timeframe.
package gates

import (
	"testing"

	"github.com/predictatrade/realtime/internal/types"
)

// TestMinATRPerTimeframeIsolation proves the ATR gate uses a different minimum
// per timeframe. A single global ATR floor is meaningless across M1 vs H4; this
// is the concrete "different calculation per strategy/timeframe" requirement.
func TestMinATRPerTimeframeIsolation(t *testing.T) {
	g := &MinAbsoluteATRGate{
		MinATR:    100, // global default floor
		MinATRByTF: map[types.Timeframe]float64{"M1": 10, "H4": 200},
	}

	// M1: ATR 15 >= M1 floor 10 → PASS
	if r := g.Evaluate(GateInput{StrategyID: "A", Timeframe: types.TFM1, ATR: 15}, GateState{}); r.Result != types.GatePass {
		t.Fatalf("M1 ATR 15 should PASS, got %s", r.Result)
	}
	// H4: ATR 15 < H4 floor 200 → VETO
	if r := g.Evaluate(GateInput{StrategyID: "A", Timeframe: types.TFH4, ATR: 15}, GateState{}); r.Result != types.GateVeto {
		t.Fatalf("H4 ATR 15 should VETO, got %s", r.Result)
	}
	// No timeframe match → global default floor 100 → ATR 15 < 100 → VETO
	if r := g.Evaluate(GateInput{StrategyID: "A", ATR: 15}, GateState{}); r.Result != types.GateVeto {
		t.Fatalf("default ATR 15 should VETO, got %s", r.Result)
	}
	// Strategy B on a different timeframe is unaffected by A's threshold.
	if r := g.Evaluate(GateInput{StrategyID: "B", Timeframe: types.TFM1, ATR: 15}, GateState{}); r.Result != types.GatePass {
		t.Fatalf("strategy B M1 ATR 15 should PASS independently, got %s", r.Result)
	}
}

// TestPerStrategyStateIsolation proves cached gate state is scoped by
// (strategy, timeframe) and never implicitly shared across strategies.
func TestPerStrategyStateIsolation(t *testing.T) {
	reg := NewRegistry()

	// Seed edge-validation state for strategy A only, at its (strategy, tf) scope.
	reg.UpdateStateScoped(GateScope{
		GateID:     types.GateEdgeValidation,
		StrategyID: "STANDARD_SCALPING",
		Timeframe:  types.TFM1,
	}, GateState{GateID: types.GateEdgeValidation, State: types.GatePass, Value: "ok"})

	// A's scoped state exists.
	if _, ok := reg.GetStateScoped(GateScope{GateID: types.GateEdgeValidation, StrategyID: "STANDARD_SCALPING", Timeframe: types.TFM1}); !ok {
		t.Fatal("strategy A scoped state should exist")
	}
	// B must NOT inherit A's state (no cross-strategy contamination).
	if _, ok := reg.GetStateScoped(GateScope{GateID: types.GateEdgeValidation, StrategyID: "ULTRA_SCALPING", Timeframe: types.TFM1}); ok {
		t.Fatal("strategy B must NOT inherit A's scoped state")
	}
	// The global scope must remain absent (state is per-scope, not centralised).
	if _, ok := reg.GetState(types.GateEdgeValidation); ok {
		t.Fatal("global edge state must be absent when only per-scope state was written")
	}
	// A different timeframe for A is also independent (per-timeframe scoping).
	if _, ok := reg.GetStateScoped(GateScope{GateID: types.GateEdgeValidation, StrategyID: "STANDARD_SCALPING", Timeframe: types.TFH4}); ok {
		t.Fatal("strategy A H4 scope must be independent of its M1 scope")
	}
}
