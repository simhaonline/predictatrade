// Package calibrate turns realized signal outcomes into a calibrated probability
// for a NAMED prediction target. It deliberately does NOT use the raw strategy
// score as a probability (raw score is not probability, per the SOW). Instead it
// fits a simple, monotonic, empirical mapping from historical setups to realized
// win rate and validates it on a held-out window before any value is published.
//
// The active prediction target is TP1_BEFORE_SL: the probability that price reaches
// the 1R partial target (TP1) before the stop, under the same exit profile the
// backtest simulator uses. Calibration is therefore consistent between research and
// live.
package calibrate

import (
	"encoding/json"

	"pat-engine/internal/types"
)

// TargetTP1BeforeSL is the named prediction target (see package doc).
const TargetTP1BeforeSL = "TP1_BEFORE_SL"

// Outcome is a single labeled signal: did the setup win on the named target?
type Outcome struct {
	StrategyID string  `json:"strategy_id"`
	Regime     string  `json:"regime"`
	RawScore   float64 `json:"raw_score"`
	Win        bool    `json:"win"`
}

// Model predicts a calibrated probability for a signal setup.
type Model interface {
	// Predict returns (probability, target, modelName, ok). ok=false means the
	// model has no basis for this setup and the probability must be treated as
	// UNCALIBRATED (never guessed).
	Predict(strategyID, regime string, rawScore float64) (prob float64, target string, model string, ok bool)
	// Bytes serializes the fitted model for loading by the live engine.
	Bytes() ([]byte, error)
}

// EmpiricalModel buckets outcomes by (strategy, regime) x score-decile and reports a
// Laplace-smoothed win fraction. Intentionally simple: it makes NO claim of market
// prediction power, only that historically similar setups won this often. It is
// fitted on a training window and must be validated on a held-out window.
type EmpiricalModel struct {
	Deciles     int                 `json:"deciles"`
	Buckets     map[string][][2]int `json:"buckets"` // key -> [decile][wins,total]
	StrategyWin map[string][2]int   `json:"strategy_win"` // key -> [wins,total] fallback
	MinScore    map[string]float64  `json:"min_score"`
	MaxScore    map[string]float64  `json:"max_score"`
}

// NewEmpirical builds an empty empirical model with the given decile count.
func NewEmpirical(deciles int) *EmpiricalModel {
	if deciles < 1 {
		deciles = 10
	}
	return &EmpiricalModel{
		Deciles:     deciles,
		Buckets:     map[string][][2]int{},
		StrategyWin: map[string][2]int{},
		MinScore:    map[string]float64{},
		MaxScore:    map[string]float64{},
	}
}

func key(s, r string) string { return s + "|" + r }

// Fit trains the model on a labeled set (offline; call once on a training window).
func (m *EmpiricalModel) Fit(outs []Outcome) {
	// Pass 1: per-key score range.
	for _, o := range outs {
		k := key(o.StrategyID, o.Regime)
		if _, ok := m.Buckets[k]; !ok {
			m.Buckets[k] = make([][2]int, m.Deciles)
			m.MinScore[k] = o.RawScore
			m.MaxScore[k] = o.RawScore
		}
		if o.RawScore < m.MinScore[k] {
			m.MinScore[k] = o.RawScore
		}
		if o.RawScore > m.MaxScore[k] {
			m.MaxScore[k] = o.RawScore
		}
	}
	// Pass 2: assign deciles and tally.
	for _, o := range outs {
		k := key(o.StrategyID, o.Regime)
		b := 0
		span := m.MaxScore[k] - m.MinScore[k]
		if span > 0 {
			b = int((o.RawScore - m.MinScore[k]) / span * float64(m.Deciles))
			if b >= m.Deciles {
				b = m.Deciles - 1
			}
			if b < 0 {
				b = 0
			}
		}
		e := m.Buckets[k][b]
		e[1]++
		if o.Win {
			e[0]++
		}
		m.Buckets[k][b] = e
		s := m.StrategyWin[k]
		s[1]++
		if o.Win {
			s[0]++
		}
		m.StrategyWin[k] = s
	}
}

func laplace(w, t int) float64 {
	return float64(w+1) / float64(t+2)
}

// Predict returns the calibrated probability for a setup. ok=false when the model
// has never seen this (strategy, regime) and cannot fall back.
func (m *EmpiricalModel) Predict(strategyID, regime string, rawScore float64) (float64, string, string, bool) {
	k := key(strategyID, regime)
	if _, ok := m.Buckets[k]; !ok {
		return 0, TargetTP1BeforeSL, "UNCALIBRATED", false
	}
	b := 0
	span := m.MaxScore[k] - m.MinScore[k]
	if span > 0 {
		b = int((rawScore - m.MinScore[k]) / span * float64(m.Deciles))
		if b >= m.Deciles {
			b = m.Deciles - 1
		}
		if b < 0 {
			b = 0
		}
	}
	w, t := m.Buckets[k][b][0], m.Buckets[k][b][1]
	if t == 0 {
		sw := m.StrategyWin[k]
		if sw[1] == 0 {
			return 0, TargetTP1BeforeSL, "UNCALIBRATED", false
		}
		return laplace(sw[0], sw[1]), TargetTP1BeforeSL, "empirical-region", true
	}
	return laplace(w, t), TargetTP1BeforeSL, "empirical-region", true
}

// Bytes serializes the fitted model.
func (m *EmpiricalModel) Bytes() ([]byte, error) {
	return json.Marshal(m)
}

// LoadModel reconstructs a fitted EmpiricalModel from Bytes().
func LoadModel(b []byte) (*EmpiricalModel, error) {
	m := &EmpiricalModel{}
	if err := json.Unmarshal(b, m); err != nil {
		return nil, err
	}
	if m.Buckets == nil {
		m.Buckets = map[string][][2]int{}
	}
	if m.StrategyWin == nil {
		m.StrategyWin = map[string][2]int{}
	}
	if m.MinScore == nil {
		m.MinScore = map[string]float64{}
	}
	if m.MaxScore == nil {
		m.MaxScore = map[string]float64{}
	}
	return m, nil
}

// Attach fills the calibrated-probability fields of a Signal from a model. When the
// model cannot calibrate the setup it marks the signal UNCALIBRATED rather than
// guessing a probability.
func Attach(sig *types.Signal, m Model) {
	if m == nil {
		sig.CalibratedProbability = 0
		sig.ProbabilityTarget = TargetTP1BeforeSL
		sig.ProbabilityModel = "UNCALIBRATED"
		return
	}
	prob, target, model, ok := m.Predict(string(sig.StrategyID), sig.Regime, sig.RawScore)
	if !ok {
		sig.CalibratedProbability = 0
		sig.ProbabilityTarget = target
		sig.ProbabilityModel = "UNCALIBRATED"
		return
	}
	sig.CalibratedProbability = prob
	sig.ProbabilityTarget = target
	sig.ProbabilityModel = model
}
