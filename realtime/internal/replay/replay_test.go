package replay

import (
	"testing"
	"time"
)

func TestReplayEngine_RunsSuccessfully(t *testing.T) {
	engine := NewReplayEngine()
	config := ReplayConfig{
		StartTime:      time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndTime:        time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC), // 2 days
		CandleInterval: 5 * time.Minute, // M5 candles for speed
		Seed:           42,
		ScenarioType:   "mixed",
		BasePrice:      2400.0,
		Volatility:     0.001,
		TrendStrength:  28.0,
	}

	result := engine.Run(config)

	if result == nil {
		t.Fatal("Replay result should not be nil")
	}
	if result.TotalCandles == 0 {
		t.Fatal("Should have processed candles")
	}

	// Should have regime distribution
	if len(result.RegimeDistribution) == 0 {
		t.Error("Should have regime distribution")
	}

	// Should have strategy funnels for all 4 strategies
	if len(result.StrategyFunnels) != 6 {
		t.Errorf("Expected 6 strategy funnels, got %d", len(result.StrategyFunnels))
	}

	// Each funnel should have evaluations
	for strategyID, funnel := range result.StrategyFunnels {
		if funnel.Evaluations == 0 {
			t.Errorf("Strategy %s should have evaluations", strategyID)
		}
	}
}

func TestReplayEngine_RegimeDistribution(t *testing.T) {
	engine := NewReplayEngine()
	config := DefaultReplayConfig()
	config.EndTime = config.StartTime.Add(48 * time.Hour) // 2 days
	config.CandleInterval = 5 * time.Minute

	result := engine.Run(config)
	dist := result.RegimeDistributionReport()

	if len(dist) == 0 {
		t.Fatal("Distribution should not be empty")
	}

	// Total should be ~100%
	total := 0.0
	for _, pct := range dist {
		total += pct
	}
	if total < 95 || total > 105 {
		t.Errorf("Distribution should sum to ~100%%, got %.1f%%", total)
	}
}

func TestReplayEngine_TransitionMatrix(t *testing.T) {
	engine := NewReplayEngine()
	config := DefaultReplayConfig()
	config.EndTime = config.StartTime.Add(48 * time.Hour)
	config.CandleInterval = 5 * time.Minute

	result := engine.Run(config)

	// With a mixed scenario, there should be transitions
	matrixStr := result.TransitionMatrixReport()
	if matrixStr == "" {
		t.Error("Transition matrix report should not be empty")
	}
}

func TestReplayEngine_FunnelReport(t *testing.T) {
	engine := NewReplayEngine()
	config := DefaultReplayConfig()
	config.EndTime = config.StartTime.Add(24 * time.Hour)
	config.CandleInterval = 5 * time.Minute

	result := engine.Run(config)
	report := result.FunnelReport()

	if report == "" {
		t.Error("Funnel report should not be empty")
	}
}

func TestReplayEngine_ShadowResults(t *testing.T) {
	engine := NewReplayEngine()
	config := DefaultReplayConfig()
	config.EndTime = config.StartTime.Add(24 * time.Hour)
	config.CandleInterval = 5 * time.Minute

	result := engine.Run(config)

	// With mixed market phases, there should be some shadow results
	// (when regime is MEAN_REVERSION but trend strategies reject)
	if len(result.ShadowResults) == 0 {
		t.Log("Warning: No shadow results generated — all strategies may have matched all regimes")
	}

	// All shadow results must be non-executable
	for _, shadow := range result.ShadowResults {
		if shadow.Executable {
			t.Error("Shadow results must never be executable")
		}
		if !shadow.ShadowOnly {
			t.Error("Shadow results must be marked ShadowOnly=true")
		}
	}
}

func TestReplayEngine_RegimeDurationStats(t *testing.T) {
	engine := NewReplayEngine()
	config := DefaultReplayConfig()
	config.EndTime = config.StartTime.Add(48 * time.Hour)
	config.CandleInterval = 5 * time.Minute

	result := engine.Run(config)
	stats := result.RegimeDurationStats()

	if len(stats) == 0 {
		t.Fatal("Duration stats should not be empty")
	}

	for regime, stat := range stats {
		if stat.Count == 0 {
			t.Errorf("Regime %s should have count > 0", regime)
		}
		if stat.Max < stat.Average {
			t.Errorf("Regime %s: max (%v) should be >= average (%v)", regime, stat.Max, stat.Average)
		}
	}
}
