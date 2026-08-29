// Package calibrate turns realized signal outcomes into a calibrated probability
// for a NAMED prediction target. It deliberately does NOT use the raw strategy
// score as a probability (raw score is not probability, per the SOW). Instead it
// fits a simple, monotonic, empirical mapping from historical setups to realized
// win rate and validates it on a held-out window before any value is published.
//
// The active prediction targets are derived from the SAME exit simulator the
// backtest uses, so calibration is consistent between research and live:
//
//	TP1_BEFORE_SL      — price reaches the 1R partial (TP1) before the SL.
//	TP2_BEFORE_SL      — price reaches the 2R target (TP2) before the SL.
//	DIRECTION_CORRECT  — price moves >= 0.5R in the signal direction before SL.
//
// Calibration is bucketed by an interpretable context (regime, HTF bias, prime
// window, volatility regime, ADX/RSI band, session) plus raw-score decile, with a
// hierarchical fallback. This is the set of "plausible scoring features": they are
// all inputs the live engine already computes, so a model fit on backtest
// transfers to live with no research drift.
package calibrate

import (
	"encoding/json"
	"strings"

	"pat-engine/internal/types"
)

// Named prediction targets (each maps to a precise, simulator-derived event).
const (
	TargetTP1BeforeSL      = "TP1_BEFORE_SL"
	TargetTP2BeforeSL      = "TP2_BEFORE_SL"
	TargetDirectionCorrect = "DIRECTION_CORRECT"
)

// AllTargets lists every named target the calibrator can model (primary first).
var AllTargets = []string{TargetTP1BeforeSL, TargetTP2BeforeSL, TargetDirectionCorrect}

// Features is a small, interpretable, categorical context vector derived from a
// MarketState. These are the "plausible scoring features": inputs the live engine
// already computes, so calibration trained on backtest transfers to live with no
// drift. Bands (not raw floats) keep the empirical buckets stable and
// sample-efficient.
type Features struct {
	Regime    string // TRENDING_BULLISH / TRENDING_BEARISH / HIGH_VOLATILITY / RANGE / ...
	HTFBias   string // BULLISH / BEARISH
	PrimeWin  string // IN / OUT  (playbook prime window 07:00-17:00 UTC)
	VolRegime string // HIGH / NORMAL / LOW  (ATR vs longer ATR baseline)
	ADXBand   string // STRONG / MODERATE / WEAK
	RSIBand   string // OB / NEUTRAL / OS
	Session   string // LONDON / OVERLAP / NEW_YORK / TOKYO / SYDNEY
}

// defaultPrimeWindows mirrors config.primeWindows(): the playbook scalping window
// where the edge concentrates. Kept here (pure, no config import) so the live
// gateway and the backtest share one definition.
var defaultPrimeWindows = [][2]int{{7, 17}}

func inWindows(h int, ws [][2]int) bool {
	for _, w := range ws {
		if h >= w[0] && h < w[1] {
			return true
		}
	}
	return false
}

// FeaturesFromState extracts the interpretable context from a live/backtest state.
func FeaturesFromState(s *types.MarketState) Features {
	if s == nil {
		return Features{}
	}
	f := Features{
		Regime:    s.Regime,
		Session:   s.Session.CurrentSession,
		PrimeWin:  "OUT",
		VolRegime: "NORMAL",
		ADXBand:   "WEAK",
		RSIBand:   "NEUTRAL",
	}
	if s.HTFBias == types.Bullish {
		f.HTFBias = "BULLISH"
	} else if s.HTFBias == types.Bearish {
		f.HTFBias = "BEARISH"
	}
	if inWindows(s.UTCHour, defaultPrimeWindows) {
		f.PrimeWin = "IN"
	}
	if s.SlowATR > 0 {
		ratio := s.ATR / s.SlowATR
		switch {
		case ratio > 1.25:
			f.VolRegime = "HIGH"
		case ratio < 0.8:
			f.VolRegime = "LOW"
		}
	}
	if s.Indicators.ADX >= 30 {
		f.ADXBand = "STRONG"
	} else if s.Indicators.ADX >= 20 {
		f.ADXBand = "MODERATE"
	}
	switch {
	case s.Indicators.RSI > 70:
		f.RSIBand = "OB"
	case s.Indicators.RSI < 30:
		f.RSIBand = "OS"
	}
	return f
}

// contextKey uniquely identifies the categorical context (ignoring score).
func (f Features) contextKey() string {
	return strings.Join([]string{f.Regime, f.HTFBias, f.PrimeWin, f.VolRegime, f.ADXBand, f.RSIBand, f.Session}, "|")
}

// Outcome is a single labeled signal: did the setup win on the named target?
type Outcome struct {
	StrategyID string
	Regime     string // kept for compatibility; redundant with Features.Regime
	RawScore   float64
	Win        bool
	Target     string // named target; "" defaults to TargetTP1BeforeSL
	Features   Features
}

func (o Outcome) target() string {
	if o.Target == "" {
		return TargetTP1BeforeSL
	}
	return o.Target
}

// Model predicts calibrated probabilities for signal setups.
type Model interface {
	// Predict returns (probability, target, modelName, ok). ok=false means the
	// model has no basis for this setup and the probability must be treated as
	// UNCALIBRATED (never guessed). modelName conveys the calibration confidence
	// level: empirical-direct / empirical-context / empirical-region /
	// empirical-strategy / UNCALIBRATED.
	Predict(strategyID, regime string, rawScore float64, feat Features) (prob float64, target string, model string, ok bool)
	// PredictTarget is like Predict but for an explicit named target.
	PredictTarget(strategyID, regime string, rawScore float64, feat Features, target string) (prob float64, model string, ok bool)
	// Bytes serializes the fitted model for loading by the live engine.
	Bytes() ([]byte, error)
}

// calTarget holds the empirical buckets for a single named target.
type calTarget struct {
	// strategyID -> contextKey -> [decile][wins, total]
	Buckets map[string]map[string][][2]int `json:"buckets"`
	// strategyID -> contextKey -> [wins, total]
	ContextWin map[string]map[string][2]int `json:"context_win"`
	// strategyID -> regionKey(strat|regime) -> [wins, total]
	RegionWin map[string]map[string][2]int `json:"region_win"`
	// strategyID -> [wins, total]
	StrategyWin map[string][2]int `json:"strategy_win"`
	// strategyID -> observed raw-score range
	MinScore map[string]float64 `json:"min_score"`
	MaxScore map[string]float64 `json:"max_score"`
}

func newCalTarget() *calTarget {
	return &calTarget{
		Buckets:     map[string]map[string][][2]int{},
		ContextWin:  map[string]map[string][2]int{},
		RegionWin:   map[string]map[string][2]int{},
		StrategyWin: map[string][2]int{},
		MinScore:    map[string]float64{},
		MaxScore:    map[string]float64{},
	}
}

// EmpiricalModel buckets outcomes by (strategy) x context x score-decile and
// reports a Laplace-smoothed win fraction with hierarchical fallback. It makes NO
// claim of market prediction power, only that historically similar setups won this
// often. It is fitted on a training window and must be validated on a held-out
// window.
type EmpiricalModel struct {
	Deciles int                  `json:"deciles"`
	Targets map[string]*calTarget `json:"targets"` // target -> per-strategy calibration
}

// NewEmpirical builds an empty empirical model with the given decile count.
func NewEmpirical(deciles int) *EmpiricalModel {
	if deciles < 1 {
		deciles = 10
	}
	return &EmpiricalModel{Deciles: deciles, Targets: map[string]*calTarget{}}
}

func (m *EmpiricalModel) target(t string) *calTarget {
	ct, ok := m.Targets[t]
	if !ok {
		ct = newCalTarget()
		m.Targets[t] = ct
	}
	return ct
}

func regKey(s, r string) string { return s + "|" + r }

// Fit trains the model on a labeled set (offline; call once on a training window).
// It runs in two passes: first it establishes the per-strategy raw-score range, then
// it assigns score deciles (so a point's decile is computed against the full range,
// never against a range that includes only itself).
func (m *EmpiricalModel) Fit(outs []Outcome) {
	// Pass 1: per-strategy raw-score range.
	for _, o := range outs {
		ct := m.target(o.target())
		s := o.StrategyID
		if _, ok := ct.Buckets[s]; !ok {
			ct.Buckets[s] = map[string][][2]int{}
			ct.ContextWin[s] = map[string][2]int{}
			ct.RegionWin[s] = map[string][2]int{}
			ct.StrategyWin[s] = [2]int{}
			ct.MinScore[s] = o.RawScore
			ct.MaxScore[s] = o.RawScore
		}
		if o.RawScore < ct.MinScore[s] {
			ct.MinScore[s] = o.RawScore
		}
		if o.RawScore > ct.MaxScore[s] {
			ct.MaxScore[s] = o.RawScore
		}
	}
	// Pass 2: assign deciles and tallies.
	for _, o := range outs {
		ct := m.target(o.target())
		s := o.StrategyID
		ck := o.Features.contextKey()
		rk := regKey(s, o.Regime)

		if _, ok := ct.Buckets[s][ck]; !ok {
			ct.Buckets[s][ck] = make([][2]int, m.Deciles)
		}
		b := decileOf(o.RawScore, ct.MinScore[s], ct.MaxScore[s], m.Deciles)
		e := ct.Buckets[s][ck][b]
		e[1]++
		if o.Win {
			e[0]++
		}
		ct.Buckets[s][ck][b] = e

		cw := ct.ContextWin[s][ck]
		cw[1]++
		if o.Win {
			cw[0]++
		}
		ct.ContextWin[s][ck] = cw

		rw := ct.RegionWin[s][rk]
		rw[1]++
		if o.Win {
			rw[0]++
		}
		ct.RegionWin[s][rk] = rw

		sw := ct.StrategyWin[s]
		sw[1]++
		if o.Win {
			sw[0]++
		}
		ct.StrategyWin[s] = sw
	}
}

func laplace(w, t int) float64 {
	return float64(w+1) / float64(t+2)
}

// decileOf maps a raw score to a decile index given the observed training range.
func decileOf(score, min, max float64, deciles int) int {
	span := max - min
	if span <= 0 {
		return 0
	}
	b := int((score - min) / span * float64(deciles))
	if b >= deciles {
		b = deciles - 1
	}
	if b < 0 {
		b = 0
	}
	return b
}

// PredictTarget returns the calibrated probability for a setup on a named target.
// ok=false when the model has never seen this strategy and cannot fall back.
// model conveys the confidence level of the returned value.
func (m *EmpiricalModel) PredictTarget(strategyID, regime string, rawScore float64, feat Features, target string) (float64, string, bool) {
	ct, ok := m.Targets[target]
	if !ok {
		return 0, "UNCALIBRATED", false
	}
	s := strategyID
	if _, ok := ct.Buckets[s]; !ok {
		return 0, "UNCALIBRATED", false
	}
	ck := feat.contextKey()
	rk := regKey(s, regime)

	// Level 1: exact context + score decile.
	if cks, ok := ct.Buckets[s]; ok {
		if buckets, ok := cks[ck]; ok {
			b := decileOf(rawScore, ct.MinScore[s], ct.MaxScore[s], m.Deciles)
			w, t := buckets[b][0], buckets[b][1]
			if t > 0 {
				return laplace(w, t), "empirical-direct", true
			}
		}
	}
	// Level 2: exact context (all scores).
	if cks, ok := ct.ContextWin[s]; ok {
		if cw, ok := cks[ck]; ok && cw[1] > 0 {
			return laplace(cw[0], cw[1]), "empirical-context", true
		}
	}
	// Level 3: region (strategy + regime).
	if rks, ok := ct.RegionWin[s]; ok {
		if rw, ok := rks[rk]; ok && rw[1] > 0 {
			return laplace(rw[0], rw[1]), "empirical-region", true
		}
	}
	// Level 4: whole-strategy fallback.
	if sw, ok := ct.StrategyWin[s]; ok && sw[1] > 0 {
		return laplace(sw[0], sw[1]), "empirical-strategy", true
	}
	return 0, "UNCALIBRATED", false
}

// Predict returns the calibrated probability for the DEFAULT target (TP1_BEFORE_SL).
func (m *EmpiricalModel) Predict(strategyID, regime string, rawScore float64, feat Features) (float64, string, string, bool) {
	p, model, ok := m.PredictTarget(strategyID, regime, rawScore, feat, TargetTP1BeforeSL)
	return p, TargetTP1BeforeSL, model, ok
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
	if m.Targets == nil {
		m.Targets = map[string]*calTarget{}
	}
	for _, ct := range m.Targets {
		if ct.Buckets == nil {
			ct.Buckets = map[string]map[string][][2]int{}
		}
		if ct.ContextWin == nil {
			ct.ContextWin = map[string]map[string][2]int{}
		}
		if ct.RegionWin == nil {
			ct.RegionWin = map[string]map[string][2]int{}
		}
		if ct.StrategyWin == nil {
			ct.StrategyWin = map[string][2]int{}
		}
		if ct.MinScore == nil {
			ct.MinScore = map[string]float64{}
		}
		if ct.MaxScore == nil {
			ct.MaxScore = map[string]float64{}
		}
	}
	return m, nil
}

// NamedProb pairs a target with its calibrated probability (defined in package types
// to avoid an import cycle).

// Attach fills the calibrated-probability fields of a Signal from a model. When the
// model cannot calibrate the setup it marks the signal UNCALIBRATED rather than
// guessing a probability. The primary target (TP1_BEFORE_SL) populates the scalar
// fields; additional targets populate the Calibrated slice.
func Attach(sig *types.Signal, m Model, feat Features) {
	if m == nil {
		sig.CalibratedProbability = 0
		sig.ProbabilityTarget = TargetTP1BeforeSL
		sig.ProbabilityModel = "UNCALIBRATED"
		sig.Calibrated = nil
		return
	}
	p, target, model, ok := m.Predict(string(sig.StrategyID), sig.Regime, sig.RawScore, feat)
	if !ok {
		sig.CalibratedProbability = 0
		sig.ProbabilityTarget = target
		sig.ProbabilityModel = "UNCALIBRATED"
		sig.Calibrated = nil
		return
	}
	sig.CalibratedProbability = p
	sig.ProbabilityTarget = target
	sig.ProbabilityModel = model

	var extra []types.NamedProb
	for _, t := range AllTargets {
		if t == TargetTP1BeforeSL {
			continue
		}
		pp, mm, ook := m.PredictTarget(string(sig.StrategyID), sig.Regime, sig.RawScore, feat, t)
		if !ook {
			continue
		}
		extra = append(extra, types.NamedProb{Target: t, Prob: pp, Model: mm})
	}
	sig.Calibrated = extra
}
