package strategy

import (
	"testing"

	"github.com/predictatrade/realtime/internal/types"
)

func TestShouldEvaluateOnRespectsDecisionTFs(t *testing.T) {
	strategies := AllStrategies()

	for _, s := range strategies {
		if p, ok := s.(DecisionTFProvider); ok {
			tfs := p.DecisionTimeframes()
			for _, tf := range tfs {
				if !ShouldEvaluateOn(s, tf) {
					t.Errorf("%s ShouldEvaluateOn(%s) = false, want true (it's a declared decision TF)", s.ID(), tf)
				}
			}
		}
	}
}

func TestShouldEvaluateOnRejectsNonDecisionTFs(t *testing.T) {
	strategies := AllStrategies()

	// TREND_SWING should NOT evaluate on M1
	for _, s := range strategies {
		if s.ID() == types.StrategyTrendSwing {
			if ShouldEvaluateOn(s, types.TFM1) {
				t.Errorf("TREND_SWING should not evaluate on M1 (not a decision TF)")
			}
		}
		// STANDARD_SCALPING should NOT evaluate on H1
		if s.ID() == types.StrategyStandardScalping {
			if ShouldEvaluateOn(s, types.TFH1) {
				t.Errorf("STANDARD_SCALPING should not evaluate on H1 (not a decision TF)")
			}
		}
	}
}

func TestAllStrategiesHaveDecisionTFs(t *testing.T) {
	strategies := AllStrategies()
	if len(strategies) != 5 {
		t.Errorf("Expected 5 strategies, got %d", len(strategies))
	}
	for _, s := range strategies {
		if p, ok := s.(DecisionTFProvider); ok {
			tfs := p.DecisionTimeframes()
			if len(tfs) == 0 {
				t.Errorf("%s has no decision timeframes", s.ID())
			}
		}
	}
}

func TestMarnieFibDecisionTFs(t *testing.T) {
	strategies := AllStrategies()
	for _, s := range strategies {
		if s.ID() == types.StrategyMarnieFib {
			if p, ok := s.(DecisionTFProvider); ok {
				tfs := p.DecisionTimeframes()
				if len(tfs) != 2 {
					t.Errorf("MARNIE_FIB should have 2 decision TFs (M15, H1), got %d", len(tfs))
				}
				hasM15, hasH1 := false, false
				for _, tf := range tfs {
					if tf == types.TFM15 { hasM15 = true }
					if tf == types.TFH1 { hasH1 = true }
				}
				if !hasM15 || !hasH1 {
					t.Errorf("MARNIE_FIB decision TFs should include M15 and H1")
				}
			}
		}
	}
}
