// Package engstatus tracks per-strategy-engine liveness so the admin
// dashboard can show truthful engine state (prompt.md Sections 26, 38, 43-46).
// It is in-memory current-state only — Valkey/DB remain the durable layers.
package engstatus

import (
	"sync"
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

// Snapshot is the truthful liveness state of one strategy engine.
type Snapshot struct {
	Engine             string    `json:"engine"`
	Enabled            bool      `json:"enabled"`
	Running            bool      `json:"running"`
	Health             string    `json:"health"` // LIVE | WAITING | STALE | ERROR
	PrimaryTFs         []string  `json:"primary_timeframes"`
	LastMarketEvent    time.Time `json:"last_market_event"`
	LastMarketTF       string    `json:"last_market_timeframe"`
	LastEvaluation     time.Time `json:"last_evaluation"`
	LastCandidate      time.Time `json:"last_candidate"`
	LastSignalAt       time.Time `json:"last_signal_at"`
	LastSignalRef      string    `json:"last_signal_reference"`
	CurrentDecision    string    `json:"current_decision"`
	CurrentScore       float64   `json:"current_score"`
	Confidence         float64   `json:"confidence"`
	CalibratedProb     float64   `json:"calibrated_probability"`
	HasCalibratedProb  bool      `json:"has_calibrated_probability"`
	DataQuality        string    `json:"data_quality"`
	Regime             string    `json:"regime"`
	EvaluationCount    int64     `json:"evaluation_count"`
	CandidateCount     int64     `json:"candidate_count"`
	SignalCount        int64     `json:"signal_count"`
	NoTradeCount       int64     `json:"no_trade_count"`
	ErrorCount         int64     `json:"error_count"`

	// audit.md Sections 21, 26, 27: data freshness, observability gaps
	DataAgeSeconds       float64  `json:"data_age_seconds"`
	CurrentThreshold     float64  `json:"current_threshold,omitempty"`
	EngineVersion        string   `json:"engine_version,omitempty"`
	LastError            string   `json:"last_error,omitempty"`
	LastErrorAt          time.Time `json:"last_error_at,omitempty"`
	CurrentRejectionReasons []string `json:"current_rejection_reasons,omitempty"`
}

// Tracker keeps current-state snapshots per strategy engine.
type Tracker struct {
	mu sync.RWMutex
	m  map[types.StrategyID]*Snapshot
}

func NewTracker(ids ...types.StrategyID) *Tracker {
	t := &Tracker{m: make(map[types.StrategyID]*Snapshot)}
	for _, id := range ids {
		t.m[id] = &Snapshot{Engine: string(id), Enabled: true, Health: "WAITING", DataQuality: "UNKNOWN", EngineVersion: "1.0.0"}
	}
	return t
}

// Update mutates the snapshot for one engine under lock. The engine is marked
// Running on first update.
func (t *Tracker) Update(id types.StrategyID, fn func(s *Snapshot)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	s, ok := t.m[id]
	if !ok {
		s = &Snapshot{Engine: string(id), Enabled: true, EngineVersion: "1.0.0"}
		t.m[id] = s
	}
	fn(s)
	s.Running = true
}

// RecordEvaluation records one strategy evaluation outcome.
func (t *Tracker) RecordEvaluation(id types.StrategyID, tf types.Timeframe, marketTime time.Time, decision types.Direction, score, confidence, prob float64, hasProb bool, regime, dataQuality string, rejectionReasons []string, threshold float64) {
	t.Update(id, func(s *Snapshot) {
		now := time.Now().UTC()
		s.LastEvaluation = now
		s.LastMarketEvent = marketTime
		s.LastMarketTF = string(tf)
		s.CurrentDecision = string(decision)
		s.CurrentScore = score
		s.Confidence = confidence
		s.CalibratedProb = prob
		s.HasCalibratedProb = hasProb
		s.Regime = regime
		s.DataQuality = dataQuality
		s.EvaluationCount++
		s.DataAgeSeconds = now.Sub(marketTime).Seconds()
		s.CurrentRejectionReasons = rejectionReasons
		s.CurrentThreshold = threshold
		switch decision {
		case types.DirectionBuy, types.DirectionSell:
			s.CandidateCount++
			s.LastCandidate = now
		case types.DirectionNoTrade, types.DirectionWait:
			s.NoTradeCount++
		default:
			s.NoTradeCount++
		}
		s.Health = "LIVE"
	})
}

// RecordIssuedSignal records a confirmed issued signal for an engine.
func (t *Tracker) RecordIssuedSignal(id types.StrategyID, ref string, at time.Time) {
	t.Update(id, func(s *Snapshot) {
		s.SignalCount++
		s.LastSignalAt = at
		s.LastSignalRef = ref
	})
}

// SetStale marks engines whose required inputs are stale (prompt.md #51).
func (t *Tracker) SetStale(id types.StrategyID) {
	t.Update(id, func(s *Snapshot) { s.Health = "STALE" })
}

// SetPrimaryTFs records the declared decision timeframes of an engine.
func (t *Tracker) SetPrimaryTFs(id types.StrategyID, tfs []string) {
	t.Update(id, func(s *Snapshot) { s.PrimaryTFs = tfs })
}

// All returns a stable copy of all snapshots ordered by engine id.
func (t *Tracker) All() []Snapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]Snapshot, 0, len(t.m))
	for _, s := range t.m {
		cp := *s
		out = append(out, cp)
	}
	// Deterministic order
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Engine < out[j-1].Engine; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
