package calibration

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// SupportedCalibrationVersion is the calibration JSON schema version the engine
// understands. Files with a different version are ignored (treated as missing),
// so subscribers never receive a fabricated or incompatible probability.
const SupportedCalibrationVersion = "1.0.0"

// CalibrationFile is the on-disk representation exported by
// research/src/patresearch/backtesting/calibration.py (ProbabilityCalibrator.export_json).
//
// Schema (canonical normalization shared with research):
//
//	x = clamp(score, 0, 100) / 100.0
//
// then:
//   - logistic:  prob = sigmoid(a*x + b)   (params.a, params.b)
//   - isotonic:  prob = linear-interp of monotonic_bins at x
type CalibrationFile struct {
	Version       string             `json:"version"`
	Strategy      string             `json:"strategy"`
	Target        string             `json:"target"`
	ExitProfile   string             `json:"exit_profile"`
	OOSAUC        float64            `json:"oos_auc"`
	NSamples      int                `json:"n_samples"`
	Method        string             `json:"method"`
	Params        map[string]float64 `json:"params"`
	TrainedAt     string             `json:"trained_at"`
	MonotonicBins []CalibrationBin   `json:"monotonic_bins"`
	XScale        float64            `json:"x_scale"`
	XClip         []float64          `json:"x_clip"`
}

// CalibrationBin is a single (x, p) knot for isotonic calibration.
type CalibrationBin struct {
	X float64 `json:"x"`
	P float64 `json:"p"`
}

// jsonModel is a parsed calibration model used for live probability mapping.
type jsonModel struct {
	strategyID types.StrategyID
	method     string
	a          float64
	b          float64
	bins       []CalibrationBin
	version    string
	nSamples   int
}

// LoadJSONModels scans dir for *.json calibration files and registers each
// schema-versioned, parseable model. Files that are missing, unparseable, or
// carry an unsupported version are silently skipped (the live path then falls
// back to Probability=0, ProbabilityCalibrated=false — never a fabricated value).
func (c *Consumer) LoadJSONModels(dir string) {
	if dir == "" {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var f CalibrationFile
		if err := json.Unmarshal(data, &f); err != nil {
			continue
		}
		if f.Version != SupportedCalibrationVersion {
			continue
		}
		sid := types.StrategyID(f.Strategy)
		if sid == "" {
			continue
		}

		// ─── Quality gate (BE-5 / BE-9) ─────────────────────────────────────
		// A loaded model is only promoted to VALIDATED when it is monotonic AND
		// meets a minimum out-of-sample quality bar. Models below the bar are
		// skipped entirely so the engine reports "no valid probability" rather
		// than surfacing a fabricated/placeholder value. This also fixes BE-9:
		// promotion is no longer driven by sample count alone — quality gates it.
		if f.OOSAUC < minCalibratedOOSAUC {
			continue
		}
		// Sample-size gate: a model trained on fewer than
		// minCalibratedSampleSize resolved outcomes cannot be trusted to
		// surface a calibrated probability to subscribers. Reject it so the
		// engine reports "no valid probability" rather than a low-sample fit.
		if f.NSamples < minCalibratedSampleSize {
			continue
		}
		if f.Method == "isotonic" && !binsMonotonic(f.MonotonicBins) {
			continue
		}

		m := jsonModel{
			strategyID: sid,
			method:     f.Method,
			version:    f.Version,
			nSamples:   f.NSamples,
		}
		if f.Method == "logistic" {
			m.a = f.Params["a"]
			m.b = f.Params["b"]
		} else if f.Method == "isotonic" {
			m.bins = f.MonotonicBins
		} else {
			continue
		}
		if c.jsonModels == nil {
			c.jsonModels = make(map[types.StrategyID]jsonModel)
		}
		// Precedence: never downgrade to a lower-sample calibration. The
		// research-trained model (typically thousands of labeled outcomes) must
		// not be clobbered by a low-sample live retrain; the live model takes
		// over only once it has accumulated more samples. Both the incumbent and
		// the candidate have already passed the quality gate above, so this is a
		// pure higher-sample-wins decision among trustworthy models.
		if existing, ok := c.jsonModels[sid]; ok && existing.nSamples >= m.nSamples {
			continue
		}
		c.jsonModels[sid] = m

		// Mirror the live model into the internal model used by Calibrate() so
		// realtime signal probability (calibratedProb / calibStatus) is driven
		// by the separate live calibration engine, not the static PROVISIONAL
		// seed. Only logistic models can be represented in the sigmoid internal
		// model; isotonic models are still served via ProbabilityFor(). The
		// VALIDATED status is only set here because the quality gate above has
		// already been satisfied.
		if f.Method == "logistic" {
			if c.models == nil {
				c.models = make(map[types.StrategyID]*CalibrationModel)
			}
			if _, exists := c.models[sid]; !exists {
				c.models[sid] = &CalibrationModel{StrategyID: sid, PredictionTarget: "TP1_HIT"}
			}
			cm := c.models[sid]
			cm.SigmoidA = decimal.NewFromFloat(f.Params["a"])
			cm.SigmoidB = decimal.NewFromFloat(f.Params["b"])
			cm.SampleSize = int64(f.NSamples)
			cm.IsActive = true
			cm.Status = "VALIDATED" // empirically calibrated + passed OOS_AUC/monotonicity gate
		}
	}
}

// ProbabilityFor maps a raw score to a calibrated probability using a
// research-trained JSON model. It returns (0, false) when no matching,
// version-compatible model is loaded — the safe fallback that guarantees no
// fabricated probability reaches the subscriber.
func (c *Consumer) ProbabilityFor(strategyID types.StrategyID, rawScore decimal.Decimal) (float64, bool) {
	m, ok := c.jsonModels[strategyID]
	if !ok {
		return 0.0, false
	}
	x := clampScore(rawScore) / 100.0
	if m.method == "isotonic" {
		return interpBins(m.bins, x), true
	}
	// logistic: sigmoid(a*x + b)
	z := m.a*x + m.b
	return 1.0 / (1.0 + math.Exp(-z)), true
}

// clampScore mirrors research _norm: clamp raw score to [0,100].
func clampScore(s decimal.Decimal) float64 {
	f, _ := s.Float64()
	if f < 0 {
		return 0
	}
	if f > 100 {
		return 100
	}
	return f
}

// interpBins performs monotonic linear interpolation over (x,p) knots clamped
// to [0,1]. Bins are assumed sorted ascending by x (as written by research).
func interpBins(bins []CalibrationBin, x float64) float64 {
	if len(bins) == 0 {
		return 0.0
	}
	if x <= bins[0].X {
		return clampProb(bins[0].P)
	}
	if x >= bins[len(bins)-1].X {
		return clampProb(bins[len(bins)-1].P)
	}
	for i := 1; i < len(bins); i++ {
		x0, p0 := bins[i-1].X, bins[i-1].P
		x1, p1 := bins[i].X, bins[i].P
		if x <= x1 {
			if x1 == x0 {
				return clampProb(p0)
			}
			t := (x - x0) / (x1 - x0)
			return clampProb(p0 + t*(p1-p0))
		}
	}
	return clampProb(bins[len(bins)-1].P)
}

// binsMonotonic reports whether the calibration knots are non-decreasing in
// probability (a calibrated model must be monotonically increasing in score →
// probability; non-monotonic fits are not trustworthy and must not be promoted).
func binsMonotonic(bins []CalibrationBin) bool {
	if len(bins) == 0 {
		return false
	}
	prev := bins[0].P
	for i := 1; i < len(bins); i++ {
		if bins[i].P < prev {
			return false
		}
		prev = bins[i].P
	}
	return true
}

func clampProb(p float64) float64 {
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}
