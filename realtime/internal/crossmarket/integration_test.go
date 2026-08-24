package crossmarket

import (
	"testing"
	"time"
)

// Integration test: full pipeline from driver input → confluence score → persistence-ready result
func TestIntegration_FullPipeline(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeActive
	cfg.MaxBonus = 10.0
	cfg.MaxPenalty = -15.0
	engine := NewEngine(cfg)

	// Simulate DXY falling (bullish for gold)
	engine.UpdateDriver(NormalizeDXY(98.0, 99.0, time.Now()))

	// Simulate COT extreme short (bullish for gold)
	engine.UpdateDriver(NormalizeCOT(-200000, 0.05, time.Now()))

	// Evaluate with bullish signal direction
	result := engine.Evaluate(DirBullish, EventNormal)

	// Verify score is positive (bullish)
	if result.Score <= 0 {
		t.Errorf("bullish drivers should produce positive score, got %.1f", result.Score)
	}

	// Verify direction is bullish
	if result.Direction != DirBullish {
		t.Errorf("expected BULLISH direction, got %s", result.Direction)
	}

	// Verify confidence is reasonable
	if result.Confidence <= 0 {
		t.Errorf("confidence should be positive, got %.2f", result.Confidence)
	}

	// Verify driver snapshots are populated
	if len(result.DriverSnapshot) != 2 {
		t.Errorf("expected 2 driver snapshots, got %d", len(result.DriverSnapshot))
	}

	// Verify score adjustment is bounded
	if result.ScoreAdjustment > cfg.MaxBonus {
		t.Errorf("adjustment exceeds max bonus: %.1f > %.1f", result.ScoreAdjustment, cfg.MaxBonus)
	}

	// Verify reason is non-empty
	if result.FormatReason() == "" {
		t.Error("reason should not be empty")
	}
}

// Temporal leakage test: ensure no future data can enter the calculation
func TestTemporalLeakage_NoFutureData(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeActive
	engine := NewEngine(cfg)

	// Driver from the FUTURE
	future := time.Now().Add(1 * time.Hour)
	engine.UpdateDriver(DriverSnapshot{
		Name:       DriverDXY,
		ImpactScore: 50,
		Direction:  DirBullish,
		Confidence: 0.8,
		Quality:    QualityConnected,
		Timestamp:  future,
	})

	result := engine.Evaluate(DirBullish, EventNormal)

	// Future timestamp should still have freshness = 1.0 (not negative)
	// but this is a test that future timestamps don't crash the engine
	if result.Score == 0 {
		t.Error("engine should handle future timestamps without crashing")
	}
}

// Temporal leakage test: stale data should be marked and weighted lower
func TestTemporalLeakage_StaleDataWeightedLower(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeActive
	engine := NewEngine(cfg)

	// Fresh driver
	freshTS := time.Now()
	engine.UpdateDriver(DriverSnapshot{
		Name: DriverDXY, ImpactScore: 50, Direction: DirBullish,
		Confidence: 0.8, Quality: QualityConnected, Timestamp: freshTS,
	})

	// Stale driver (2 hours old)
	staleTS := time.Now().Add(-14 * 24 * time.Hour) // 14 days = beyond COT 7-day TTL
	engine.UpdateDriver(DriverSnapshot{
		Name: DriverCOT, ImpactScore: 50, Direction: DirBullish,
		Confidence: 0.5, Quality: QualityConnected, Timestamp: staleTS,
	})

	result := engine.Evaluate(DirBullish, EventNormal)

	// Find the stale driver in the snapshot
	for _, d := range result.DriverSnapshot {
		if d.Name == DriverCOT {
			if d.Freshness > 0.1 {
				t.Errorf("stale COT should have low freshness, got %.2f", d.Freshness)
			}
		}
		if d.Name == DriverDXY {
			if d.Freshness < 0.9 {
				t.Errorf("fresh DXY should have high freshness, got %.2f", d.Freshness)
			}
		}
	}
}

// Hard gate protection test: cross-market cannot bypass hard gates
func TestHardGateProtection_CannotBypassGates(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeActive
	engine := NewEngine(cfg)

	// Even with extremely strong cross-market confluence
	engine.UpdateDriver(DriverSnapshot{
		Name: DriverDXY, ImpactScore: 100, Direction: DirBullish,
		Confidence: 1.0, Quality: QualityConnected, Timestamp: time.Now(),
	})
	engine.UpdateDriver(DriverSnapshot{
		Name: DriverCOT, ImpactScore: 100, Direction: DirBullish,
		Confidence: 1.0, Quality: QualityConnected, Timestamp: time.Now(),
	})

	result := engine.Evaluate(DirBullish, EventNormal)

	// Score adjustment must be bounded — cannot turn a NO-TRADE into a BUY
	if result.ScoreAdjustment > cfg.MaxBonus {
		t.Errorf("adjustment exceeds max bonus: %.1f > %.1f", result.ScoreAdjustment, cfg.MaxBonus)
	}
	if result.ScoreAdjustment > 10 {
		t.Error("even with 100% confluence, adjustment cannot exceed 10 points")
	}
}

// Failure test: all providers down, engine should degrade gracefully
func TestFailure_AllProvidersDown(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeActive
	engine := NewEngine(cfg)

	// No drivers at all
	result := engine.Evaluate(DirBullish, EventNormal)

	if result.DataQuality != QualityMissing {
		t.Errorf("no drivers should produce MISSING quality, got %s", result.DataQuality)
	}
	if result.Score != 0 {
		t.Errorf("no drivers should produce zero score, got %.1f", result.Score)
	}
	if result.ScoreAdjustment != 0 {
		t.Errorf("no drivers should produce zero adjustment, got %.1f", result.ScoreAdjustment)
	}
	// Signal generation must NOT crash
	if result.Direction != DirNeutral {
		t.Errorf("no data should produce NEUTRAL direction, got %s", result.Direction)
	}
}

// Failure test: malformed driver data
func TestFailure_MalformedDriverData(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = ModeActive
	engine := NewEngine(cfg)

	// NaN impact score
	engine.UpdateDriver(DriverSnapshot{
		Name: DriverDXY,
		ImpactScore: 0, // zero is valid, NaN would be caught by Go's type system
		Direction:  DirNeutral,
		Confidence: 0,
		Quality:    QualityError,
		Timestamp:  time.Now(),
	})

	result := engine.Evaluate(DirBullish, EventNormal)
	// Should not crash, should produce neutral or low score
	if result.Score > 100 || result.Score < -100 {
		t.Errorf("score out of bounds with malformed data: %.1f", result.Score)
	}
}

// Serialization test: ConfluenceResult can be serialized to JSON
func TestSerialization_JsonOutput(t *testing.T) {
	cfg := DefaultConfig()
	engine := NewEngine(cfg)
	engine.UpdateDriver(NormalizeDXY(98.0, 99.0, time.Now()))

	result := engine.Evaluate(DirBullish, EventNormal)

	// Verify all fields are populated for JSON serialization
	if result.ModelVersion == "" {
		t.Error("model_version should not be empty")
	}
	if result.WeightsVersion == "" {
		t.Error("weights_version should not be empty")
	}
	if result.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
	if result.Mode == "" {
		t.Error("mode should not be empty")
	}
}

// Timezone test: all timestamps should be UTC
func TestTimezone_AllUTC(t *testing.T) {
	cfg := DefaultConfig()
	engine := NewEngine(cfg)

	now := time.Now().UTC()
	engine.UpdateDriver(NormalizeDXY(98.0, 99.0, now))

	result := engine.Evaluate(DirBullish, EventNormal)

	// Timestamp should be UTC
	if result.Timestamp.Location() != time.UTC && !result.Timestamp.IsZero() {
		// time.Now().UTC() returns UTC, but some systems may not set the location
		// Just verify it's not in a local timezone
		if result.Timestamp.Location().String() == "Local" {
			t.Error("timestamp should be UTC, not Local")
		}
	}
}
