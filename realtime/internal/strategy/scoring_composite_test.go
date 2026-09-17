package strategy

import (
	"testing"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// P1 (prompt.md): 8-family signed-sum composite with per-family diagnostics.
// Families: TREND, MTF, MOMENTUM, VOLUME, SMC, GEOMETRY, CANDLE, MACRO.
// The composite = signed sum of family scores × 1.5, clamped ±100.
// The engine's existing confluence (weighted pillar contributions) is NOT
// replaced — FamilySubScores is a pure diagnostics layer over the same
// evidence rows (no overlap: additive, config-gated, read-only).
func TestFamilySubScoresAndComposite(t *testing.T) {
	ev := []types.EvidenceContribution{}
	addEvidenceRaw(&ev, "TREND", "EMA9_ABOVE_EMA21", types.DirectionBuy, 15, 0.12, types.QualityAuthoritative, "", decimal.NewFromInt(4321))
	addEvidence(&ev, "MOMENTUM", "MACD_BULLISH", types.DirectionBuy, 10, 0.06, types.QualityAuthoritative, "")
	addEvidence(&ev, "SMC", "BOS_CONFIRMED", types.DirectionBuy, 10, 0.10, types.QualityAuthoritative, "")
	addEvidence(&ev, "CANDLE", "BULLISH_ENGULF", types.DirectionSell, 8, -0.08, types.QualityAuthoritative, "")

	fs := ComputeFamilySubScores(ev)
	if tf, _ := fs.Trend.Float64(); tf <= 0 {
		t.Fatalf("TREND family should be positive (EMA9 buy), got %v", fs.Trend)
	}
	if sf, _ := fs.SMC.Float64(); sf <= 0 {
		t.Fatalf("SMC family should be positive (BOS), got %v", fs.SMC)
	}
	if cf, _ := fs.Candle.Float64(); cf >= 0 {
		t.Fatalf("CANDLE family should be negative (sell-direction candle row), got %v", fs.Candle)
	}
	if !fs.MTF.IsZero() || !fs.Macro.IsZero() {
		t.Fatalf("absent families must score 0 — MTF %v MACRO %v", fs.MTF, fs.Macro)
	}

	comp := FamilyComposite(fs)
	// deterministic: (0.12 + 0.06 + 0.10 − 0.08) × 1.5 = 0.30
	want := decimal.NewFromFloat(0.30)
	if !comp.Equal(want) {
		t.Fatalf("composite = %s, want %s", comp, want)
	}

	// clamp: huge scores pin to ±100
	big := []types.EvidenceContribution{}
	for i := 0; i < 50; i++ {
		addEvidence(&big, "TREND", "X", types.DirectionBuy, 100, 10, types.QualityAuthoritative, "")
	}
	bigComp := FamilyComposite(ComputeFamilySubScores(big))
	if bigComp.GreaterThan(decimal.NewFromInt(100)) {
		t.Fatalf("composite must clamp at +100, got %s", bigComp)
	}
}

func TestReferenceTierLadder(t *testing.T) {
	t.Setenv("REFERENCE_TIER_LADDER", "true")

	// prompt.md P1 tier ladder: A+ 55, A 42, B 30, C 22, Watch 12.
	// Feature-flagged (REFERENCE_TIER_LADDER) — the existing quality_grade
	// ladder (70/55/15) remains the default until the operator enables it.
	if v := ReferenceTier(56); v != "A+" {
		t.Fatalf("56 → A+, got %s", v)
	}
	if v := ReferenceTier(43); v != "A" {
		t.Fatalf("43 → A, got %s", v)
	}
	if v := ReferenceTier(31); v != "B" {
		t.Fatalf("31 → B, got %s", v)
	}
	if v := ReferenceTier(23); v != "C" {
		t.Fatalf("23 → C, got %s", v)
	}
	if v := ReferenceTier(13); v != "WATCH" {
		t.Fatalf("13 → WATCH, got %s", v)
	}
	if v := ReferenceTier(5); v != "" {
		t.Fatalf("5 → no tier (empty), got %q", v)
	}
}

// flag-off default: the ladder returns "" so existing ComputeQualityGrade
// behavior remains the default until the operator enables it.
func TestReferenceTierLadderFlagOff(t *testing.T) {
	t.Setenv("REFERENCE_TIER_LADDER", "")
	if v := ReferenceTier(80); v != "" {
		t.Fatalf("flag off must return empty tier, got %q", v)
	}
}

func TestRegimeAllowsDirection(t *testing.T) {
	// prompt.md P1: RegimeAllows — trending regimes admit only the aligned
	// direction; squeeze admits nothing; everything else admits both.
	if !RegimeAllowsDirection(types.RegimeTrendingBullish, types.DirectionBuy) {
		t.Fatal("trending bullish must allow BUY")
	}
	if RegimeAllowsDirection(types.RegimeTrendingBullish, types.DirectionSell) {
		t.Fatal("trending bullish must NOT allow SELL")
	}
	if RegimeAllowsDirection(types.RegimeTrendingBearish, types.DirectionBuy) {
		t.Fatal("trending bearish must NOT allow BUY")
	}
	if RegimeAllowsDirection(types.RegimeRange, types.DirectionBuy) {
		t.Fatal("squeeze must NOT allow any direction")
	}
	if !RegimeAllowsDirection(types.RegimeBreakout, types.DirectionBuy) ||
		!RegimeAllowsDirection(types.RegimeBreakout, types.DirectionSell) {
		t.Fatal("breakout admits both directions")
	}
}
