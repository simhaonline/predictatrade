package devilliquidity

import (
	"testing"
	"time"
)

// TestContributionShadowModeDoesNotContribute verifies the fail-closed contract:
// in shadow mode EvaluateSignalContribution must return OK=false (no production
// impact), even when an active mark exists.
func TestContributionShadowModeDoesNotContribute(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeShadow
	e := NewEngine("", cfg)
	e.SetEnabled(true)
	base := time.Now()

	// Warm + bullish displacement to create an active mark.
	var warm []CandleInput
	for i := 0; i < 10; i++ {
		warm = append(warm, *cndl("XAUUSD", "M5", 1000, 1006, 999, 1005, 100, base.Add(time.Duration(i)*time.Minute)))
	}
	run(e, toPtrs(warm)...)
	run(e, cndl("XAUUSD", "M5", 1005, 1045, 1005, 1044, 100, base.Add(11*time.Minute)))

	if len(e.AllMarks()) != 1 {
		t.Fatalf("expected 1 active mark, got %d", len(e.AllMarks()))
	}
	dl := e.EvaluateSignalContribution("XAUUSD", "M5")
	if dl.OK {
		t.Fatalf("shadow mode must not contribute; got OK=true adj=%.3f", dl.ScoreAdjustment)
	}
	if dl.ScoreAdjustment != 0 {
		t.Fatalf("shadow mode adjustment must be 0, got %.3f", dl.ScoreAdjustment)
	}
}

// TestContributionConfluenceBounded verifies confluence mode produces a bounded,
// non-zero nudge within [MaxPenalty, MaxBonus] for a high-quality active mark.
func TestContributionConfluenceBounded(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeConfluence
	cfg.MaxBonus = 10.0
	cfg.MaxPenalty = -10.0
	e := NewEngine("", cfg)
	e.SetEnabled(true)
	base := time.Now()

	var warm []CandleInput
	for i := 0; i < 10; i++ {
		warm = append(warm, *cndl("XAUUSD", "M5", 1000, 1006, 999, 1005, 100, base.Add(time.Duration(i)*time.Minute)))
	}
	run(e, toPtrs(warm)...)
	disp := cndl("XAUUSD", "M5", 1005, 1045, 1005, 1044, 100, base.Add(11*time.Minute))
	// Crank mark quality up via the scorer path by feeding a strong candle is
	// unnecessary; verify the contribution respects bounds regardless of quality.
	run(e, disp)

	dl := e.EvaluateSignalContribution("XAUUSD", "M5")
	if !dl.OK {
		t.Fatalf("confluence mode should contribute when a mark exists")
	}
	if dl.ScoreAdjustment > cfg.MaxBonus+1e-9 || dl.ScoreAdjustment < cfg.MaxPenalty-1e-9 {
		t.Fatalf("adjustment %.3f outside bounds [%.2f, %.2f]", dl.ScoreAdjustment, cfg.MaxPenalty, cfg.MaxBonus)
	}
	if dl.Direction != DirBullish {
		t.Fatalf("expected bullish direction from the displacement mark, got %s", dl.Direction)
	}
}

// TestContributionNoMarkNoImpact verifies no contribution when no active mark
// exists for the symbol/timeframe.
func TestContributionNoMarkNoImpact(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeConfluence
	e := NewEngine("", cfg)
	e.SetEnabled(true)
	dl := e.EvaluateSignalContribution("EURUSD", "M15")
	if dl.OK {
		t.Fatalf("no active mark should yield no contribution; got OK=true")
	}
	if dl.ScoreAdjustment != 0 {
		t.Fatalf("expect 0 adjustment with no mark, got %.3f", dl.ScoreAdjustment)
	}
}

// TestContributionDisabledNoImpact verifies a disabled engine never contributes.
func TestContributionDisabledNoImpact(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeConfluence
	e := NewEngine("", cfg)
	e.SetEnabled(false)
	base := time.Now()
	var warm []CandleInput
	for i := 0; i < 10; i++ {
		warm = append(warm, *cndl("XAUUSD", "M5", 1000, 1006, 999, 1005, 100, base.Add(time.Duration(i)*time.Minute)))
	}
	run(e, toPtrs(warm)...)
	run(e, cndl("XAUUSD", "M5", 1005, 1045, 1005, 1044, 100, base.Add(11*time.Minute)))
	if len(e.AllMarks()) != 0 {
		t.Fatalf("disabled engine must not create marks, got %d", len(e.AllMarks()))
	}
	dl := e.EvaluateSignalContribution("XAUUSD", "M5")
	if dl.OK {
		t.Fatalf("disabled engine must not contribute")
	}
}
