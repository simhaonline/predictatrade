package strategy

import (
	"fmt"
	"testing"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// TestStrategyExitSpec_Distinct verifies each strategy has a DISTINCT exit
// geometry (SL/TP multipliers, spread cap, micro-TP, partial-close) as required.
func TestStrategyExitSpec_Distinct(t *testing.T) {
	ids := []types.StrategyID{
		types.StrategyUltraScalping,
		types.StrategyStandardScalping,
		types.StrategyStandardSwing,
		types.StrategyTrendSwing,
		types.StrategyMarnieFib,
	}
	seen := map[string]bool{}
	for _, id := range ids {
		s := StrategyExitSpec(id)
		key := fmt.Sprintf("%v|%v|%v|%v|%v|%v|%v", s.ATRMultSL, s.ATRMultTP1, s.ATRMultTP2, s.ATRMultTP3, s.MicroTPATRMult, s.PartialClosePct, s.MaxSpreadPips)
		if seen[key] {
			t.Errorf("exit spec for %s is not distinct: %s", id, key)
		}
		seen[key] = true
		if s.MicroTPATRMult <= 0 {
			t.Errorf("%s: micro-TP multiplier must be > 0", id)
		}
		if s.PartialClosePct <= 0 || s.PartialClosePct > 1 {
			t.Errorf("%s: partial-close pct must be in (0,1]", id)
		}
		if s.MaxSpreadPips <= 0 {
			t.Errorf("%s: max spread must be > 0", id)
		}
	}
}

// TestRefinementEnrichment_BullishFixture verifies the refinement layer enriches
// a directional result with micro-TP, edge and EV, and does NOT mark the strong
// bullish fixture as a loss candidate (so legitimate signals still pass).
func TestRefinementEnrichment_BullishFixture(t *testing.T) {
	for _, strat := range []Strategy{NewUltraScalping(), NewStandardScalping(), NewStandardSwing(), NewTrendSwing()} {
		state := makeBullishState()
		res := strat.Evaluate(state)
		if res.Direction != types.DirectionBuy {
			// WAIT/SELL/NO_TRADE are not the asserted tradeable path here; the
			// refinement only enriches confirmed directional (BUY) candidates.
			t.Logf("%s -> %s (skipping enrichment assertion)", strat.ID(), res.Direction)
			continue
		}
		if res.MicroTP.IsZero() {
			t.Errorf("%s: MicroTP must be populated after refinement", strat.ID())
		}
		if res.PartialClosePct <= 0 {
			t.Errorf("%s: PartialClosePct must be > 0", strat.ID())
		}
		if !res.EntryGatePassed {
			t.Errorf("%s: unique entry gate should PASS on strong bullish fixture (reasons=%v)", strat.ID(), res.ReasonCodes)
		}
		if res.IsLossCandidate {
			t.Errorf("%s: strong bullish fixture must not be a loss candidate (EV=%.3f)", strat.ID(), res.ExpectedValue)
		}
	}
}

// TestEvaluateProfitability_NegativeWhenCostHigh verifies the loss-candidate
// filter flags clearly unprofitable setups (excessive cost vs geometry).
func TestEvaluateProfitability_NegativeWhenCostHigh(t *testing.T) {
	state := makeBullishState()
	state.Spread = decimal.NewFromFloat(50) // absurdly wide spread destroys edge
	spec := StrategyExitSpec(types.StrategyStandardScalping)
	// Build a trivial directional geometry.
	entry := decimal.NewFromFloat(4400)
	sl := decimal.NewFromFloat(4377.5)
	tp1 := decimal.NewFromFloat(4437.5)
	prof := EvaluateProfitability(state, types.DirectionBuy, entry, sl, tp1, spec, 70)
	if !prof.LossCandidate {
		t.Errorf("expected loss candidate with excessive spread, got LossCandidate=false (EV=%.3f)", prof.ExpectedValue)
	}
	if prof.MicroTPProfitable {
		t.Errorf("micro-TP must be unprofitable when spread dominates")
	}
}

// TestUniqueEntryGate_RejectsExtendedChase verifies the entry gate rejects an
// inefficient (over-extended) entry that would chase the move.
func TestUniqueEntryGate_RejectsExtendedChase(t *testing.T) {
	state := makeBullishState()
	// Force an extremely extended entry far above VWAP/value.
	state.CurrentPrice = decimal.NewFromFloat(4600)
	state.VWAP.SessionVWAP = decimal.NewFromFloat(4400)
	state.Indicators.ATR = decimal.NewFromFloat(15)
	ok, reasons, _ := UniqueEntryGate(types.StrategyStandardScalping, state, types.DirectionBuy)
	if ok {
		t.Errorf("entry gate should reject an over-extended chase, got PASS (reasons=%v)", reasons)
	}
}
