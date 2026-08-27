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
		m := jsonModel{
			strategyID: sid,
			method:     f.Method,
			version:    f.Version,
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
		c.jsonModels[sid] = m

		// Mirror the live model into the internal model used by Calibrate() so
		// realtime signal probability (calibratedProb / calibStatus) is driven
		// by the separate live calibration engine, not the static PROVISIONAL
		// seed. Only logistic models can be represented in the sigmoid internal
		// model; isotonic models are still served via ProbabilityFor().
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
			cm.Status = "VALIDATED" // empirically calibrated from real resolved outcomes
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

func clampProb(p float64) float64 {
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}
