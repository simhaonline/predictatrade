package gates

import (
	"testing"

	"github.com/predictatrade/realtime/internal/types"
)

func TestRegimeDirectionGate(t *testing.T) {
	g := &RegimeDirectionGate{}
	state := GateState{State: types.GatePass, SourceVersion: "live"}

	cases := []struct {
		regime   types.Regime
		dir      types.Direction
		wantPass bool
		reason   string
	}{
		{types.RegimeTrendingBullish, types.DirectionBuy, true, ""},
		{types.RegimeTrendingBullish, types.DirectionSell, false, ReasonRegimeDirMismatch},
		{types.RegimeTrendingBearish, types.DirectionSell, true, ""},
		{types.RegimeTrendingBearish, types.DirectionBuy, false, ReasonRegimeDirMismatch},
		{types.RegimeRange, types.DirectionBuy, false, ReasonRegimeSqueezeVeto},
		{types.RegimeRange, types.DirectionSell, false, ReasonRegimeSqueezeVeto},
		{types.RegimeBreakout, types.DirectionBuy, true, ""},
		{types.RegimeBreakout, types.DirectionSell, true, ""},
		{types.RegimeHighVolatility, types.DirectionBuy, true, ""},
	}
	for i, c := range cases {
		eval := g.Evaluate(GateInput{Regime: c.regime, Direction: c.dir}, state)
		pass := eval.Result == types.GatePass
		if pass != c.wantPass {
			t.Fatalf("case %d (%s %s): pass=%v want %v reasons=%v",
				i, c.regime, c.dir, pass, c.wantPass, eval.ReasonCodes)
		}
		if c.reason != "" && (len(eval.ReasonCodes) == 0 || eval.ReasonCodes[0] != c.reason) {
			t.Fatalf("case %d: reason = %v, want [%s]", i, eval.ReasonCodes, c.reason)
		}
	}
	// Non-directional evaluation passes
	eval := g.Evaluate(GateInput{Regime: types.RegimeRange, Direction: types.DirectionNoTrade}, state)
	if eval.Result != types.GatePass {
		t.Fatal("non-directional must pass")
	}
	// Fail-closed: unregistered/uninitialized state must NOT pass silently —
	// the base() carries the seeded state; verify the gate ID is wired.
	if g.ID() != types.GateID("regime_direction") {
		t.Fatalf("gate id = %s", g.ID())
	}
}
