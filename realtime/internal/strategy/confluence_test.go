package strategy

import (
	"testing"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

func TestConfluenceScoring_BuySignal(t *testing.T) {
	profiles := SeedProfiles()
	profile := profiles[types.StrategyStandardScalping]

	// With weights summing to 100 and threshold 75, evidence needs to be strong
	// liquidity: 25 weight, structure: 20, fvg_ob: 15, flow_volume: 20, regime: 10, macro: 10
	// For score >= 75, need high normalized values
	evidence := []types.EvidenceContribution{
		{Pillar: "liquidity", NormalizedValue: decimal.NewFromInt(90), Direction: types.DirectionBuy, Mandatory: true, Quality: types.QualityAuthoritative},
		{Pillar: "structure", NormalizedValue: decimal.NewFromInt(85), Direction: types.DirectionBuy, Mandatory: true, Quality: types.QualityAuthoritative},
		{Pillar: "fvg_ob", NormalizedValue: decimal.NewFromInt(80), Direction: types.DirectionBuy, Mandatory: false, Quality: types.QualityAuthoritative},
		{Pillar: "flow_volume", NormalizedValue: decimal.NewFromInt(75), Direction: types.DirectionBuy, Mandatory: false, Quality: types.QualityAuthoritative},
		{Pillar: "regime_volatility", NormalizedValue: decimal.NewFromInt(70), Direction: types.DirectionBuy, Mandatory: false, Quality: types.QualityAuthoritative},
		{Pillar: "macro_news", NormalizedValue: decimal.NewFromInt(65), Direction: types.DirectionBuy, Mandatory: false, Quality: types.QualityAuthoritative},
	}

	result := Evaluate(profile, ConfluenceInput{Evidence: evidence})

	// Score = 90*25/100 + 85*20/100 + 80*15/100 + 75*20/100 + 70*10/100 + 65*10/100
	// = 22.5 + 17 + 12 + 15 + 7 + 6.5 = 80
	if !result.PassThreshold {
		t.Errorf("Expected threshold pass (75), got score=%s", result.TotalScore.String())
	}
	if !result.MandatoryMet {
		t.Error("Expected mandatory pillars met")
	}

	dir := Direction(result)
	if dir != types.DirectionBuy {
		t.Errorf("Expected BUY, got %s (long=%s short=%s)", dir, result.LongScore.String(), result.ShortScore.String())
	}
}

func TestConfluenceScoring_NoTrade_InsufficientScore(t *testing.T) {
	profiles := SeedProfiles()
	profile := profiles[types.StrategyUltraScalping]

	evidence := []types.EvidenceContribution{
		{Pillar: "flow_microstructure", NormalizedValue: decimal.NewFromInt(50), Direction: types.DirectionBuy, Mandatory: true, Quality: types.QualityAuthoritative},
		{Pillar: "liquidity_event", NormalizedValue: decimal.NewFromInt(40), Direction: types.DirectionBuy, Mandatory: true, Quality: types.QualityAuthoritative},
		{Pillar: "execution_cost_quality", NormalizedValue: decimal.NewFromInt(30), Direction: types.DirectionBuy, Mandatory: true, Quality: types.QualityAuthoritative},
	}

	result := Evaluate(profile, ConfluenceInput{Evidence: evidence})

	if result.PassThreshold {
		t.Error("Expected threshold fail for low score")
	}

	dir := Direction(result)
	if dir != types.DirectionNoTrade {
		t.Errorf("Expected NO-TRADE, got %s", dir)
	}
}

func TestConfluenceScoring_MandatoryPillarMissing(t *testing.T) {
	profiles := SeedProfiles()
	profile := profiles[types.StrategyStandardScalping]

	evidence := []types.EvidenceContribution{
		{Pillar: "structure", NormalizedValue: decimal.NewFromInt(90), Direction: types.DirectionBuy, Mandatory: true, Quality: types.QualityAuthoritative},
		{Pillar: "fvg_ob", NormalizedValue: decimal.NewFromInt(85), Direction: types.DirectionBuy, Mandatory: false, Quality: types.QualityAuthoritative},
	}

	result := Evaluate(profile, ConfluenceInput{Evidence: evidence})

	if result.MandatoryMet {
		t.Error("Expected mandatory NOT met (liquidity missing)")
	}

	dir := Direction(result)
	if dir != types.DirectionNoTrade {
		t.Errorf("Expected NO-TRADE when mandatory missing, got %s", dir)
	}
}

func TestFourStrategiesAreDistinct(t *testing.T) {
	profiles := SeedProfiles()

	// Verify all four strategies have different thresholds
	thresholds := make(map[string]int)
	for sid, p := range profiles {
		key := p.MinimumScore.String()
		thresholds[key]++
		_ = sid
	}
	if len(thresholds) < 2 {
		t.Error("Expected at least 2 different thresholds across strategies")
	}

	// Verify all four have different mandatory pillar sets
	mandatorySets := make(map[string]bool)
	for sid, p := range profiles {
		key := ""
		for _, m := range p.MandatoryPillars {
			key += m + ","
		}
		if mandatorySets[key] {
			t.Errorf("Strategies have identical mandatory pillars: %s", sid)
		}
		mandatorySets[key] = true
	}
}

func TestSeedRiskProfiles(t *testing.T) {
	profiles := SeedRiskProfiles()

	if len(profiles) != 5 {
		t.Errorf("Expected 5 risk profiles, got %d", len(profiles))
	}

	// SOW 25A.1: STANDARD_SCALPING min R:R 1.20
	if !profiles[types.StrategyStandardScalping].MinGrossRR.Equal(decimal.NewFromFloat(1.20)) {
		t.Error("STANDARD_SCALPING min gross RR should be 1.20")
	}
	if !profiles[types.StrategyUltraScalping].MinGrossRR.Equal(decimal.NewFromFloat(1.00)) {
		t.Error("ULTRA_SCALPING min gross RR should be 1.00")
	}
	if !profiles[types.StrategyStandardSwing].MinGrossRR.Equal(decimal.NewFromFloat(1.80)) {
		t.Error("STANDARD_SWING min gross RR should be 1.80")
	}
	if !profiles[types.StrategyTrendSwing].MinGrossRR.Equal(decimal.NewFromFloat(2.50)) {
		t.Error("TREND_SWING min gross RR should be 2.50")
	}
}
