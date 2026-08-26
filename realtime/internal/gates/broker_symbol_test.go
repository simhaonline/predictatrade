package gates

import (
	"testing"

	"github.com/predictatrade/realtime/internal/types"
)

// TestBrokerSymbolValidatorGate tests P0-001 broker symbol validation scenarios.
func TestBrokerSymbolValidatorGate(t *testing.T) {
	makeInput := func(entry, sl, tp1, lot float64) GateInput {
		return GateInput{
			EntryPrice:  entry,
			StopLoss:    sl,
			TakeProfit1: tp1,
			RequestedLot: lot,
			SymbolTickSize: 0.01, // XAUUSD typical tick size
			Direction:    types.DirectionBuy,
		}
	}

	passState := GateState{State: types.GatePass}

	t.Run("pass_normal_levels", func(t *testing.T) {
		g := &BrokerSymbolValidatorGate{
			MinStopPoints:   50,   // 50 points = 5.0 for XAUUSD at 0.01 tick
			MinFreezePoints: 0,    // not set
			MinLot:          0.01,
			MaxLot:          10.0,
			LotStep:         0.01,
			Digits:          2,
		}
		// Entry=2650, SL=2630 → 20.0 distance = 2000 points → passes 50-point min
		input := makeInput(2650.0, 2630.0, 2670.0, 0.01)
		eval := g.Evaluate(input, passState)
		if eval.Result != types.GatePass {
			t.Errorf("expected PASS, got %v: %v", eval.Result, eval.ReasonCodes)
		}
	})

	t.Run("veto_sl_too_close", func(t *testing.T) {
		g := &BrokerSymbolValidatorGate{
			MinStopPoints:   50,   // 50 points = 0.50 minimum distance
			MinFreezePoints: 0,
			MinLot:          0.01,
			MaxLot:          10.0,
			LotStep:         0.01,
			Digits:          2,
		}
		// Entry=2650, SL=2649.80 → 0.20 = 20 points → FAILS 50-point min
		input := makeInput(2650.0, 2649.80, 2670.0, 0.01)
		eval := g.Evaluate(input, passState)
		if eval.Result != types.GateVeto {
			t.Errorf("expected VETO for SL too close, got %v", eval.Result)
		}
	})

	t.Run("veto_tp_too_close", func(t *testing.T) {
		g := &BrokerSymbolValidatorGate{
			MinStopPoints:   50,
			MinFreezePoints: 0,
			MinLot:          0.01,
			MaxLot:          10.0,
			LotStep:         0.01,
			Digits:          2,
		}
		// Entry=2650, TP=2650.10 → 0.10 = 10 points → FAILS
		input := makeInput(2650.0, 2630.0, 2650.10, 0.01)
		eval := g.Evaluate(input, passState)
		if eval.Result != types.GateVeto {
			t.Errorf("expected VETO for TP too close, got %v", eval.Result)
		}
	})

	t.Run("veto_lot_below_min", func(t *testing.T) {
		g := &BrokerSymbolValidatorGate{
			MinStopPoints:   0,
			MinFreezePoints: 0,
			MinLot:          0.01,
			MaxLot:          10.0,
			LotStep:         0.01,
			Digits:          2,
		}
		input := makeInput(2650.0, 2630.0, 2670.0, 0.005) // lot below 0.01 min
		eval := g.Evaluate(input, passState)
		if eval.Result != types.GateVeto {
			t.Errorf("expected VETO for lot below minimum, got %v", eval.Result)
		}
	})

	t.Run("veto_lot_above_max", func(t *testing.T) {
		g := &BrokerSymbolValidatorGate{
			MinStopPoints:   0,
			MinFreezePoints: 0,
			MinLot:          0.01,
			MaxLot:          10.0,
			LotStep:         0.01,
			Digits:          2,
		}
		input := makeInput(2650.0, 2630.0, 2670.0, 50.0) // lot above 10.0 max
		eval := g.Evaluate(input, passState)
		if eval.Result != types.GateVeto {
			t.Errorf("expected VETO for lot above maximum, got %v", eval.Result)
		}
	})

	t.Run("veto_lot_not_aligned", func(t *testing.T) {
		g := &BrokerSymbolValidatorGate{
			MinStopPoints:   0,
			MinFreezePoints: 0,
			MinLot:          0.01,
			MaxLot:          10.0,
			LotStep:         0.01,
			Digits:          2,
		}
		input := makeInput(2650.0, 2630.0, 2670.0, 0.015) // not aligned to 0.01 step
		eval := g.Evaluate(input, passState)
		if eval.Result != types.GateVeto {
			t.Errorf("expected VETO for lot not aligned to step, got %v", eval.Result)
		}
	})

	t.Run("degrade_when_state_not_pass", func(t *testing.T) {
		g := &BrokerSymbolValidatorGate{
			MinStopPoints:   50,
			MinFreezePoints: 0,
			MinLot:          0.01,
			MaxLot:          10.0,
			LotStep:         0.01,
			Digits:          2,
		}
		unknownState := GateState{State: types.GateUnknown}
		input := makeInput(2650.0, 2630.0, 2670.0, 0.01)
		eval := g.Evaluate(input, unknownState)
		if eval.Result != types.GateDegraded {
			t.Errorf("expected DEGRADED when gate state is unknown, got %v", eval.Result)
		}
	})

	t.Run("pass_zero_constraints", func(t *testing.T) {
		g := &BrokerSymbolValidatorGate{} // all zero = no constraints
		input := makeInput(2650.0, 2649.99, 2650.01, 0.01)
		eval := g.Evaluate(input, passState)
		if eval.Result != types.GatePass {
			t.Errorf("expected PASS with zero constraints, got %v: %v", eval.Result, eval.ReasonCodes)
		}
	})
}
