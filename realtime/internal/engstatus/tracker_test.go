package engstatus

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

func TestTrackerRecordsEvaluationLifecycle(t *testing.T) {
	tr := NewTracker(types.StrategyUltraScalping, types.StrategyTrendSwing)

	tr.RecordEvaluation(types.StrategyUltraScalping, types.TFM1,
		time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC),
		types.DirectionBuy, 72.5, 0, 0.63, true, "TREND", "GOOD")

	snaps := tr.All()
	if len(snaps) != 2 {
		t.Fatalf("snapshots = %d, want 2", len(snaps))
	}
	var us *Snapshot
	for i := range snaps {
		if snaps[i].Engine == string(types.StrategyUltraScalping) {
			us = &snaps[i]
		}
	}
	if us == nil {
		t.Fatal("ultra scalping snapshot missing")
	}
	if !us.Running || us.Health != "LIVE" {
		t.Errorf("health = %s running=%v, want LIVE/true", us.Health, us.Running)
	}
	if us.EvaluationCount != 1 || us.CandidateCount != 1 || us.NoTradeCount != 0 {
		t.Errorf("counts eval=%d cand=%d notrade=%d", us.EvaluationCount, us.CandidateCount, us.NoTradeCount)
	}
	if us.CurrentScore != 72.5 || !us.HasCalibratedProb || us.CalibratedProb != 0.63 {
		t.Errorf("score/prob = %v/%v hasProb=%v", us.CurrentScore, us.CalibratedProb, us.HasCalibratedProb)
	}

	// NO-TRADE counts separately (prompt.md #26)
	tr.RecordEvaluation(types.StrategyUltraScalping, types.TFM1, time.Now().UTC(),
		types.DirectionNoTrade, 10, 0, 0, false, "RANGE", "GOOD")
	snaps = tr.All()
	for i := range snaps {
		if snaps[i].Engine == string(types.StrategyUltraScalping) {
			if snaps[i].NoTradeCount != 1 || snaps[i].CandidateCount != 1 {
				t.Errorf("no_trade=%d candidate=%d, want 1/1", snaps[i].NoTradeCount, snaps[i].CandidateCount)
			}
		}
	}

	// Issued signal recorded with reference
	tr.RecordIssuedSignal(types.StrategyUltraScalping, "PAT-XAU-000123", time.Now().UTC())
	snaps = tr.All()
	for i := range snaps {
		if snaps[i].Engine == string(types.StrategyUltraScalping) {
			if snaps[i].SignalCount != 1 || snaps[i].LastSignalRef != "PAT-XAU-000123" {
				t.Errorf("signal tracking broken: %+v", snaps[i])
			}
		}
	}

	// Stale marking (prompt.md #51, #111): stale input must show STALE, not LIVE
	tr.SetStale(types.StrategyTrendSwing)
	snaps = tr.All()
	for i := range snaps {
		if snaps[i].Engine == string(types.StrategyTrendSwing) && snaps[i].Health != "STALE" {
			t.Errorf("trend swing health = %s, want STALE", snaps[i].Health)
		}
	}
}
