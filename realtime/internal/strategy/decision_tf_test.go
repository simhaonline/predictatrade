package strategy

import (
	"testing"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
)

// prompt.md Sections 6, 67-68, 97-98: each engine must evaluate only on its
// declared decision timeframes. M1 closes must never trigger swing/daily
// engines and vice versa.
func TestShouldEvaluateOnRespectsDecisionTFs(t *testing.T) {
	all := AllStrategies()
	if len(all) == 0 {
		t.Fatal("no strategies registered")
	}
	tfs := []types.Timeframe{types.TFM1, types.TFM5, types.TFM15, types.TFM30, types.TFH1, types.TFH4, types.TFD1, types.TFW1}

	for _, s := range all {
		p, implements := s.(DecisionTFProvider)
		if !implements {
			t.Fatalf("%s must implement DecisionTFProvider", s.ID())
		}
		declared := map[types.Timeframe]bool{}
		for _, tf := range p.DecisionTimeframes() {
			declared[tf] = true
		}
		if len(declared) == 0 {
			t.Fatalf("%s declares no decision timeframes", s.ID())
		}
		for _, tf := range tfs {
			got := ShouldEvaluateOn(s, tf)
			want := declared[tf]
			if got != want {
				t.Errorf("%s ShouldEvaluateOn(%s) = %v, want %v", s.ID(), tf, got, want)
			}
		}
	}
}

// prompt.md Section 98: updating M1 may re-evaluate Ultra Scalping but must
// never trigger Trend Swing weekly logic.
func TestM1UpdateDoesNotTriggerTrendSwing(t *testing.T) {
	found := false
	for _, s := range AllStrategies() {
		if s.ID() != types.StrategyTrendSwing {
			continue
		}
		found = true
		if ShouldEvaluateOn(s, types.TFM1) {
			t.Error("TrendSwing must not be evaluated on M1 closes")
		}
		if !ShouldEvaluateOn(s, types.TFH1) && !ShouldEvaluateOn(s, types.TFH4) {
			t.Error("TrendSwing must evaluate on its H1/H4 decision timeframes")
		}
	}
	if !found {
		t.Fatal("TrendSwing not registered")
	}
}

// Non-provider strategies (legacy-compatible) always evaluate.
type nonProviderStrategy struct{}

func (n *nonProviderStrategy) ID() types.StrategyID { return "NON_PROVIDER" }
func (n *nonProviderStrategy) Evaluate(_ *features.MarketState) StrategyResult {
	return StrategyResult{}
}

func TestNonProviderStrategyAlwaysEvaluates(t *testing.T) {
	s := &nonProviderStrategy{}
	for _, tf := range []types.Timeframe{types.TFM1, types.TFH1, types.TFD1} {
		if !ShouldEvaluateOn(s, tf) {
			t.Errorf("legacy strategy should evaluate on %s", tf)
		}
	}
}
