package gates

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

// Phase-0 regression (live incident 2026-09-17 12:27 UTC): EvaluateAll held
// r.mu.RLock across the gate loop while gates called GetState (recursive
// RLock). A writer queued by any hydrate loop blocked the recursive RLock →
// engine-wide self-deadlock, strategy evaluation stopped. The fix snapshots
// the gate map and evaluates lock-free; this test hammers the exact
// reader/writer interleaving that deadlocked production.
func TestEvaluateAllNoDeadlockWithConcurrentWriters(t *testing.T) {
	reg := NewRegistry()
	for _, g := range []Gate{
		&DataQualityGate{}, &SessionGate{}, &SpreadGate{MaxSpreadAbsolute: 1},
		&RegimeDirectionGate{}, &MinAbsoluteATRGate{MinATR: 0.5},
		&StopHuntFilterGate{MinDistanceATR: 0.5}, &EntitlementGate{}, &LicenseGate{},
	} {
		reg.Register(g)
	}
	SeedConservativeGateStates(reg)

	done := make(chan struct{})
	go func() { // writer storm: 3 hydrate loops, 5-10s cadence like production
		for i := 0; i < 200; i++ {
			reg.UpdateState(types.GateDailyLoss, GateState{State: types.GatePass, SourceVersion: "pnl"})
			reg.UpdateState(types.GateProfitTarget, GateState{State: types.GatePass, SourceVersion: "pnl"})
			reg.UpdateState(types.GateRegimeDirection, GateState{State: types.GatePass, SourceVersion: "seed"})
			time.Sleep(time.Millisecond)
		}
		close(done)
	}()

	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-done:
			return // writers finished — readers never deadlocked
		case <-deadline:
			t.Fatal("EvaluateAll deadlocked against concurrent writers (gates.go EvaluateAll recursion)")
		default:
			input := GateInput{
				Tick: &types.Tick{Quality: types.QualityAuthoritative},
				SessionAllowed: true, NewsRisk: "LOW", Spread: 0.2, ATR: 3,
				EntryPrice: 2400, StopLoss: 2396, TakeProfit1: 2404,
				Regime: types.RegimeTrendingBullish, Direction: types.DirectionBuy,
				EntitlementOK: true, LicenseActive: true, ExecutionPermitted: true,
			}
			allPass, _, _ := reg.EvaluateAll(input)
			_ = allPass
		}
	}
}

// EvaluateAll must keep its fail-closed contract under the snapshot model:
// an unregistered gate in the order list still vetoes.
func TestEvaluateAllUnregisteredStillFailsClosed(t *testing.T) {
	reg := NewRegistry()
	SeedConservativeGateStates(reg)
	// order contains gates; none registered
	allPass, _, veto := reg.EvaluateAll(GateInput{})
	if allPass || veto == nil {
		t.Fatalf("unregistered gates must fail closed: allPass=%v veto=%v", allPass, veto)
	}
}
