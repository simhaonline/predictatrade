package igs

import (
	"testing"
	"time"
)

func TestClassifyBands(t *testing.T) {
	cases := []struct {
		score     float64
		available int
		want      Classification
	}{
		{85, 4, ClassExtremeBull},
		{60, 4, ClassStrongBull},
		{30, 4, ClassModerateBull},
		{10, 4, ClassNeutral},
		{-10, 4, ClassNeutral},
		{-30, 4, ClassModerateBear},
		{-60, 4, ClassStrongBear},
		{-90, 4, ClassExtremeBear},
		{90, 1, ClassInsufficient},
		{-90, 0, ClassInsufficient},
	}
	for _, c := range cases {
		got := classify(c.score, c.available)
		if got != c.want {
			t.Errorf("classify(%v, %d) = %v, want %v", c.score, c.available, got, c.want)
		}
	}
}

func TestShadowModeZeroAdjustment(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Mode = ModeShadow
	e := NewEngine(cfg)
	now := time.Now().UTC()
	// Stack all enabled components aggressively bullish.
	e.UpdateComponent(Component{Name: ComponentUSDRegime, Impact: 100, Confidence: 1, Quality: QualityConnected, Timestamp: now})
	e.UpdateComponent(Component{Name: ComponentRealYield, Impact: 100, Confidence: 1, Quality: QualityConnected, Timestamp: now})
	e.UpdateComponent(Component{Name: ComponentCOT, Impact: 100, Confidence: 1, Quality: QualityConnected, Timestamp: now})

	res := e.Evaluate(DirBullish)
	if res.Mode != ModeShadow {
		t.Fatalf("expected shadow mode, got %s", res.Mode)
	}
	if res.ScoreAdjustment != 0 {
		t.Fatalf("shadow mode must produce zero adjustment, got %v", res.ScoreAdjustment)
	}
	if res.Classification != ClassStrongBull && res.Classification != ClassModerateBull && res.Classification != ClassExtremeBull {
		t.Fatalf("expected bullish classification, got %s (score %v)", res.Classification, res.Score)
	}
	if res.Score <= 0 {
		t.Fatalf("expected positive score, got %v", res.Score)
	}
}

func TestDisabledEngineReturnsEmpty(t *testing.T) {
	e := NewEngine(DefaultConfig()) // Enabled=false
	res := e.Evaluate(DirNeutral)
	if res.Score != 0 || res.ComponentsAvailable != 0 {
		t.Fatalf("disabled engine must return empty composite, got %+v", res)
	}
}

func TestMissingComponentsSurfacedNotFabricated(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.EnabledComponents[ComponentETF] = true         // enabled but never fed
	cfg.EnabledComponents[ComponentCentralBank] = true // enabled but never fed
	e := NewEngine(cfg)
	e.UpdateComponent(Component{Name: ComponentUSDRegime, Impact: 30, Confidence: 0.8, Quality: QualityConnected, Timestamp: time.Now().UTC()})

	res := e.Evaluate(DirNeutral)
	foundETF, foundCB := false, false
	for _, m := range res.MissingComponents {
		if m == "etf_flows" {
			foundETF = true
		}
		if m == "central_bank_flow" {
			foundCB = true
		}
	}
	if !foundETF || !foundCB {
		t.Fatalf("missing components must be surfaced: %+v", res.MissingComponents)
	}
	if res.DataQuality != QualityDegraded && res.DataQuality != QualityUnavailable {
		t.Fatalf("expected degraded quality with missing feeds, got %s", res.DataQuality)
	}
}

func TestStaleComponentDegrades(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.EnabledComponents[ComponentCOT] = true
	cfg.EnabledComponents[ComponentUSDRegime] = true
	cfg.EnabledComponents[ComponentRealYield] = true
	e := NewEngine(cfg)
	old := time.Now().UTC().Add(-14 * 24 * time.Hour) // way past weekly TTL
	e.UpdateComponent(Component{Name: ComponentCOT, Impact: 50, Confidence: 1, Quality: QualityConnected, Timestamp: old})
	e.UpdateComponent(Component{Name: ComponentUSDRegime, Impact: 0, Confidence: 1, Quality: QualityConnected, Timestamp: time.Now().UTC()})
	e.UpdateComponent(Component{Name: ComponentRealYield, Impact: 0, Confidence: 1, Quality: QualityConnected, Timestamp: time.Now().UTC()})

	res := e.Evaluate(DirNeutral)
	staleFound := false
	for _, w := range res.Warnings {
		if w == "cot_positioning is stale" {
			staleFound = true
		}
	}
	if !staleFound {
		t.Fatalf("expected stale warning for COT, warnings=%v", res.Warnings)
	}
}

func TestActiveModeBoundedAdjustment(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Mode = ModeActive
	e := NewEngine(cfg)
	now := time.Now().UTC()
	e.UpdateComponent(Component{Name: ComponentUSDRegime, Impact: 100, Confidence: 1, Quality: QualityConnected, Timestamp: now})
	e.UpdateComponent(Component{Name: ComponentRealYield, Impact: 100, Confidence: 1, Quality: QualityConnected, Timestamp: now})
	e.UpdateComponent(Component{Name: ComponentCOT, Impact: 100, Confidence: 1, Quality: QualityConnected, Timestamp: now})

	res := e.Evaluate(DirBullish)
	if res.ScoreAdjustment > cfg.MaxBonus {
		t.Fatalf("adjustment %v exceeds MaxBonus %v", res.ScoreAdjustment, cfg.MaxBonus)
	}
	if res.ScoreAdjustment < cfg.MaxPenalty {
		t.Fatalf("adjustment %v below MaxPenalty %v", res.ScoreAdjustment, cfg.MaxPenalty)
	}
}

func TestFromCrossMarketFanIn(t *testing.T) {
	c := FromCrossMarket(CrossMarketDriver{
		Name: "dxy", ImpactScore: -42.5, Confidence: 0.7, Quality: "CONNECTED",
		Source: "twelvedata", Reason: "DXY falling", Timestamp: time.Now().UTC(),
	})
	if c.Name != ComponentUSDRegime {
		t.Fatalf("expected usd_regime, got %s", c.Name)
	}
	if c.Impact != -42.5 {
		t.Fatalf("expected impact preserved, got %v", c.Impact)
	}
	if c.Quality != QualityConnected {
		t.Fatalf("expected CONNECTED, got %s", c.Quality)
	}
	bad := FromCrossMarket(CrossMarketDriver{Name: "vix", Quality: "CONNECTED"})
	if bad.Quality != QualityUnavailable {
		t.Fatalf("unsupported driver must map to UNAVAILABLE, got %s", bad.Quality)
	}
}

func TestFreshnessDecayReducesInfluence(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	e := NewEngine(cfg)
	old := time.Now().UTC().Add(-3 * 24 * time.Hour) // past 1d real-yield TTL
	e.UpdateComponent(Component{Name: ComponentUSDRegime, Impact: 100, Confidence: 1, Quality: QualityConnected, Timestamp: time.Now().UTC()})
	e.UpdateComponent(Component{Name: ComponentRealYield, Impact: 100, Confidence: 1, Quality: QualityConnected, Timestamp: old})
	e.UpdateComponent(Component{Name: ComponentCOT, Impact: 100, Confidence: 1, Quality: QualityConnected, Timestamp: time.Now().UTC()})

	res := e.Evaluate(DirNeutral)
	staleFound := false
	for _, w := range res.Warnings {
		if w == "real_yield_regime is stale" {
			staleFound = true
		}
	}
	if !staleFound {
		t.Fatalf("expected real_yield_regime staleness warning, got %+v", res.Warnings)
	}
}
