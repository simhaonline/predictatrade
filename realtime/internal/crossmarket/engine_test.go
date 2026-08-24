package crossmarket

import (
	"testing"
	"time"
)

func TestNormalizeDXY_BullishForGold(t *testing.T) {
	// DXY falling = bullish for gold
	snap := NormalizeDXY(98.5, 99.0, time.Now())
	if snap.Direction != DirBullish {
		t.Errorf("falling DXY should be bullish for gold, got %s (impact=%.1f)", snap.Direction, snap.ImpactScore)
	}
	if snap.ImpactScore <= 0 {
		t.Errorf("falling DXY should have positive impact, got %.1f", snap.ImpactScore)
	}
}

func TestNormalizeDXY_BearishForGold(t *testing.T) {
	// DXY rising = bearish for gold
	snap := NormalizeDXY(99.5, 99.0, time.Now())
	if snap.Direction != DirBearish {
		t.Errorf("rising DXY should be bearish for gold, got %s (impact=%.1f)", snap.Direction, snap.ImpactScore)
	}
	if snap.ImpactScore >= 0 {
		t.Errorf("rising DXY should have negative impact, got %.1f", snap.ImpactScore)
	}
}

func TestNormalizeDXY_ScoreBounds(t *testing.T) {
	// Extreme move should be clamped
	snap := NormalizeDXY(90.0, 100.0, time.Now())
	if snap.ImpactScore > 100 || snap.ImpactScore < -100 {
		t.Errorf("impact score out of bounds: %.1f", snap.ImpactScore)
	}
}

func TestNormalizeEURUSD_ConfirmationRole(t *testing.T) {
	// Rising EURUSD = weakening USD = bullish for gold
	snap := NormalizeEURUSD(1.10, 1.09, time.Now())
	if snap.Direction != DirBullish {
		t.Errorf("rising EURUSD should be bullish for gold, got %s", snap.Direction)
	}
	// EURUSD impact should be capped lower than DXY (confirmation role)
	if snap.ImpactScore > 60 {
		t.Errorf("EURUSD impact should be capped at 60, got %.1f", snap.ImpactScore)
	}
}

func TestAntiDoubleCounting(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeActive
	engine := NewEngine(cfg)

	// Both DXY falling and EURUSD rising = same USD weakness signal
	engine.UpdateDriver(NormalizeDXY(98.0, 99.0, time.Now())) // bullish for gold
	engine.UpdateDriver(NormalizeEURUSD(1.10, 1.09, time.Now())) // bullish for gold

	result := engine.Evaluate(DirBullish, EventNormal)

	// Score should be positive (bullish) but NOT double-counted
	if result.Score <= 0 {
		t.Errorf("bullish drivers should produce positive score, got %.1f", result.Score)
	}
	// The EURUSD weight should have been reduced due to DXY presence
	// Check that the effective weight in driver snapshot reflects the reduction
	var eurusdEffWeight float64
	var dxyEffWeight float64
	for _, d := range result.DriverSnapshot {
		if d.Name == DriverEURUSD {
			eurusdEffWeight = d.EffectiveWeight
		}
		if d.Name == DriverDXY {
			dxyEffWeight = d.EffectiveWeight
		}
	}
	// EURUSD effective weight should be significantly less than DXY
	if eurusdEffWeight >= dxyEffWeight {
		t.Errorf("EURUSD weight (%.2f) should be less than DXY (%.2f) due to collinearity", eurusdEffWeight, dxyEffWeight)
	}
}

func TestMissingDataGracefulDegradation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeActive
	engine := NewEngine(cfg)

	// No drivers provided at all
	result := engine.Evaluate(DirBullish, EventNormal)
	if result.DataQuality != QualityMissing {
		t.Errorf("no drivers should produce MISSING quality, got %s", result.DataQuality)
	}
	if result.Score != 0 {
		t.Errorf("no drivers should produce zero score, got %.1f", result.Score)
	}
}

func TestStaleData(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeActive
	engine := NewEngine(cfg)

	// Driver with very old timestamp
	old := time.Now().Add(-2 * time.Hour)
	engine.UpdateDriver(NormalizeDXY(98.0, 99.0, old))

	result := engine.Evaluate(DirBullish, EventNormal)
	// Should have stale quality and low freshness
	hasStale := false
	for _, d := range result.DriverSnapshot {
		if d.Quality == QualityStale {
			hasStale = true
		}
	}
	if !hasStale {
		t.Error("old driver should be marked stale")
	}
}

func TestShadowModeZeroAdjustment(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeShadow
	engine := NewEngine(cfg)
	engine.UpdateDriver(NormalizeDXY(98.0, 99.0, time.Now()))

	result := engine.Evaluate(DirBullish, EventNormal)
	if result.ScoreAdjustment != 0 {
		t.Errorf("shadow mode should produce zero adjustment, got %.1f", result.ScoreAdjustment)
	}
}

func TestScoreBounds(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeActive
	engine := NewEngine(cfg)

	// Add extreme drivers
	engine.UpdateDriver(DriverSnapshot{
		Name: DriverDXY, ImpactScore: 100, Direction: DirBullish,
		Confidence: 1.0, Quality: QualityConnected, Timestamp: time.Now(),
	})
	engine.UpdateDriver(DriverSnapshot{
		Name: DriverEURUSD, ImpactScore: 100, Direction: DirBullish,
		Confidence: 1.0, Quality: QualityConnected, Timestamp: time.Now(),
	})

	result := engine.Evaluate(DirBullish, EventNormal)
	if result.Score > 100 || result.Score < -100 {
		t.Errorf("score out of bounds [-100,100]: %.1f", result.Score)
	}
}

func TestAdjustmentBounds(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeActive
	cfg.MaxBonus = 10.0
	cfg.MaxPenalty = -15.0
	engine := NewEngine(cfg)

	// Strong bullish drivers
	engine.UpdateDriver(DriverSnapshot{
		Name: DriverDXY, ImpactScore: 100, Direction: DirBullish,
		Confidence: 1.0, Quality: QualityConnected, Timestamp: time.Now(),
	})

	result := engine.Evaluate(DirBullish, EventNormal)
	if result.ScoreAdjustment > cfg.MaxBonus {
		t.Errorf("adjustment exceeds max bonus: %.1f > %.1f", result.ScoreAdjustment, cfg.MaxBonus)
	}
	if result.ScoreAdjustment < cfg.MaxPenalty {
		t.Errorf("adjustment below max penalty: %.1f < %.1f", result.ScoreAdjustment, cfg.MaxPenalty)
	}
}

func TestDivergenceDetection(t *testing.T) {
	detector := NewDivergenceDetector()

	// Signal is bullish but DXY is rising (bearish for gold)
	drivers := []DriverSnapshot{
		{Name: DriverDXY, Direction: DirBearish, ImpactScore: -60},
	}
	sev := detector.Detect(DirBullish, drivers, SHNORMAL)
	if sev == DivNone {
		t.Error("bearish DXY vs bullish signal should detect divergence")
	}
}

func TestSafeHavenOverride(t *testing.T) {
	detector := NewDivergenceDetector()

	// During dual safe haven, DXY rising + gold bullish is EXPECTED
	drivers := []DriverSnapshot{
		{Name: DriverDXY, Direction: DirBearish, ImpactScore: -60},
	}
	sev := detector.Detect(DirBullish, drivers, SHDualSafeHaven)
	// Should be reduced severity due to safe-haven override
	if sev == DivExtreme {
		t.Error("safe-haven override should reduce divergence severity")
	}
}

func TestEventExtremeBlocksAdjustment(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeActive
	engine := NewEngine(cfg)
	engine.UpdateDriver(DriverSnapshot{
		Name: DriverDXY, ImpactScore: 100, Direction: DirBullish,
		Confidence: 1.0, Quality: QualityConnected, Timestamp: time.Now(),
	})

	result := engine.Evaluate(DirBullish, EventExtreme)
	if result.ScoreAdjustment != 0 {
		t.Errorf("extreme event risk should block adjustment, got %.1f", result.ScoreAdjustment)
	}
}

func TestFreshnessDecay(t *testing.T) {
	now := time.Now()
	// Fresh
	fresh := computeFreshness(now, 300, now)
	if fresh != 1.0 {
		t.Errorf("fresh data should have freshness 1.0, got %.2f", fresh)
	}
	// Half TTL
	half := computeFreshness(now.Add(-150*time.Second), 300, now)
	if half < 0.45 || half > 0.55 {
		t.Errorf("half-TTL data should have freshness ~0.5, got %.2f", half)
	}
	// Expired
	expired := computeFreshness(now.Add(-600*time.Second), 300, now)
	if expired != 0 {
		t.Errorf("expired data should have freshness 0, got %.2f", expired)
	}
}

func TestCorrelationDetector(t *testing.T) {
	detector := NewCorrelationDetector(50)

	// Not enough data
	regime := detector.Classify()
	if regime != CorrInsufficient {
		t.Errorf("insufficient data should return INSUFFICIENT_DATA, got %s", regime)
	}

	// Add perfectly inversely correlated data
	for i := 0; i < 30; i++ {
		detector.AddGoldPrice(float64(100 + i))
		detector.AddDXY(float64(100 - i))
	}
	regime = detector.Classify()
	if regime != CorrInverse {
		t.Errorf("inversely correlated data should return INVERSE, got %s", regime)
	}
}

func TestHardGateProtection(t *testing.T) {
	// The engine's score adjustment must never bypass hard gates.
	// This is verified by checking that adjustment is bounded and
	// the engine operates in shadow mode by default.
	cfg := DefaultConfig()
	if cfg.Mode != ModeShadow {
		t.Error("default mode should be SHADOW — no production impact")
	}
	if cfg.MaxBonus > 15 {
		t.Error("max bonus should be bounded (≤15)")
	}
	if cfg.MaxPenalty < -20 {
		t.Error("max penalty should be bounded (≥-20)")
	}
}

func TestBTCLowWeight(t *testing.T) {
	cfg := DefaultConfig()
	btcWeight := cfg.Weights[DriverBTC]
	dxyWeight := cfg.Weights[DriverDXY]
	if btcWeight >= dxyWeight {
		t.Errorf("BTC weight (%.1f) should be much lower than DXY (%.1f)", btcWeight, dxyWeight)
	}
	if btcWeight > 5 {
		t.Errorf("BTC weight should be ≤5, got %.1f", btcWeight)
	}
}

func TestDisabledMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeDisabled
	engine := NewEngine(cfg)
	engine.UpdateDriver(NormalizeDXY(98.0, 99.0, time.Now()))

	result := engine.Evaluate(DirBullish, EventNormal)
	if result.DataQuality != QualityMissing {
		t.Errorf("disabled mode should return MISSING, got %s", result.DataQuality)
	}
}

func TestCOTNormalization(t *testing.T) {
	// Extreme long positioning = crowding risk
	snap := NormalizeCOT(200000, 0.95, time.Now())
	if snap.Direction != DirBearish {
		t.Errorf("extreme long COT should be bearish (crowding), got %s", snap.Direction)
	}

	// Extreme short positioning = squeeze potential
	snap = NormalizeCOT(-200000, 0.05, time.Now())
	if snap.Direction != DirBullish {
		t.Errorf("extreme short COT should be bullish (squeeze), got %s", snap.Direction)
	}
}

func TestVIXExtremeFear(t *testing.T) {
	// VIX > 40 = liquidity stress, can be bearish for gold
	snap := NormalizeVIX(45, 20, time.Now())
	if snap.ImpactScore >= 0 {
		t.Errorf("extreme VIX should be bearish for gold (liquidity stress), got impact %.1f", snap.ImpactScore)
	}
}

func TestProviderFailureNotBlocking(t *testing.T) {
	// If one provider fails, others should still contribute
	cfg := DefaultConfig()
	cfg.Mode = ModeActive
	engine := NewEngine(cfg)

	// Only DXY available, EURUSD missing
	engine.UpdateDriver(NormalizeDXY(98.0, 99.0, time.Now()))

	result := engine.Evaluate(DirBullish, EventNormal)
	if result.Score == 0 {
		t.Error("partial data should still produce a score")
	}
	if len(result.MissingDrivers) == 0 {
		t.Error("missing drivers should be listed")
	}
}

func TestFormatReason(t *testing.T) {
	result := ConfluenceResult{
		Score:      68,
		Direction:  DirBullish,
		EventRisk:  EventElevated,
		MissingDrivers: []string{"vix", "btc"},
	}
	reason := result.FormatReason()
	if reason == "" {
		t.Error("reason should not be empty")
	}
}
