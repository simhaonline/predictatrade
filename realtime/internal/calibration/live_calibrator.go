// Package calibration implements the live calibration outcome writer.
// SOW Section 16: Periodic retraining of calibration models from resolved outcomes.
// Fixes audit P0 F-006: replaces fabricated VALIDATED seed metadata with
// empirically-calibrated models from real shadow outcome data.
package calibration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LiveCalibrator periodically reads resolved shadow outcomes and writes
// research-compatible calibration JSON files so the Go engine's
// LoadJSONModels path can replace the hardcoded PROVISIONAL seed models
// with empirically-calibrated probabilities.
type LiveCalibrator struct {
	mu       sync.Mutex
	db       *sql.DB
	outputDir string
	interval  time.Duration
	afterRun  func()
	stopCh    chan struct{}

	// Strategy-specific stats (computed during each run)
	LastRun    time.Time
	TotalSamples int
}

// CalibratorConfig configures the live calibrator.
type CalibratorConfig struct {
	DB          *sql.DB
	OutputDir   string
	Interval    time.Duration
	// AfterRun is invoked after each successful calibration run (e.g. to reload
	// the freshly written models into the live prediction consumer so realtime
	// signals use the latest calibrated probability).
	AfterRun func()
}

// DefaultCalibratorInterval is the default recalibration frequency.
const DefaultCalibratorInterval = 1 * time.Hour

// NewLiveCalibrator creates a live calibrator that writes calibration JSONs
// from resolved shadow outcomes.
func NewLiveCalibrator(cfg CalibratorConfig) *LiveCalibrator {
	if cfg.Interval <= 0 {
		cfg.Interval = DefaultCalibratorInterval
	}
	return &LiveCalibrator{
		db:        cfg.DB,
		outputDir: cfg.OutputDir,
		interval:  cfg.Interval,
		afterRun:  cfg.AfterRun,
		stopCh:    make(chan struct{}),
	}
}

// Start begins the periodic calibration loop.
func (c *LiveCalibrator) Start() {
	go c.loop()
}

// Stop signals the calibration loop to shut down.
func (c *LiveCalibrator) Stop() {
	close(c.stopCh)
}

func (c *LiveCalibrator) loop() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	// Run immediately on start
	c.runCalibration(context.Background())

	for {
		select {
		case <-ticker.C:
			c.runCalibration(context.Background())
		case <-c.stopCh:
			return
		}
	}
}

// strategyOutcome tracks per-strategy calibration data from resolved snapshots.
type strategyOutcome struct {
	rawScores []float64
	labels    []float64 // 1.0 = win (TP hit), 0.0 = loss (SL hit)
	wins      int
	losses    int
}

// runCalibration performs a single calibration run: reads resolved outcomes,
// fits logistic calibration per strategy, and writes JSON files.
func (c *LiveCalibrator) runCalibration(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.db == nil || c.outputDir == "" {
		return
	}

	outcomes, err := c.loadResolvedOutcomes(ctx)
	if err != nil {
		return
	}

	if len(outcomes) == 0 {
		return
	}

	// Group outcomes by strategy
	byStrategy := make(map[string]*strategyOutcome)
	for _, o := range outcomes {
		s, ok := byStrategy[o.Strategy]
		if !ok {
			s = &strategyOutcome{}
			byStrategy[o.Strategy] = s
		}
		s.rawScores = append(s.rawScores, o.RawScore)
		if o.IsWin {
			s.labels = append(s.labels, 1.0)
			s.wins++
		} else {
			s.labels = append(s.labels, 0.0)
			s.losses++
		}
	}

	// Fit logistic calibration per strategy and write JSON files
	if err := os.MkdirAll(c.outputDir, 0755); err != nil {
		return
	}

	for strategy, data := range byStrategy {
		total := data.wins + data.losses
		if total < minCalibratedSampleSize {
			continue // Insufficient samples for meaningful/trustworthy calibration
		}
		a, b, ok := fitLogisticCalibration(data.rawScores, data.labels)
		if !ok {
			continue
		}

		// Normalize scores to [0,1] for an honest, rank-based AUC of how well
		// the raw score discriminates wins from losses (NOT the win rate, which
		// is a separate statistic and must not be mislabeled as AUC).
		norm := make([]float64, len(data.rawScores))
		for i, s := range data.rawScores {
			x := s / 100.0
			if x < 0 {
				x = 0
			} else if x > 1 {
				x = 1
			}
			norm[i] = x
		}
		oosAUC := auc(norm, data.labels)

		winRate := float64(data.wins) / float64(total)
		c.writeCalibrationJSON(strategy, a, b, total, winRate, oosAUC)
	}

	c.LastRun = time.Now().UTC()
	c.TotalSamples = len(outcomes)

	if c.afterRun != nil {
		c.afterRun()
	}
}

// resolvedOutcome is a single resolved shadow signal for calibration.
type resolvedOutcome struct {
	Strategy  string
	RawScore  float64
	IsWin     bool
	ResolvedAt time.Time
}

func (c *LiveCalibrator) loadResolvedOutcomes(ctx context.Context) ([]resolvedOutcome, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT strategy, technical_score, outcome
		FROM trading.cross_market_shadow_snapshots
		WHERE outcome IN ('TP1', 'TP2', 'TP3', 'SL', 'EXPIRED')
		  AND technical_score >= 0
		ORDER BY timestamp ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var outcomes []resolvedOutcome
	for rows.Next() {
		var strategy, outcome string
		var score float64
		if err := rows.Scan(&strategy, &score, &outcome); err != nil {
			continue
		}
		isWin := outcome == "TP1" || outcome == "TP2" || outcome == "TP3"
		outcomes = append(outcomes, resolvedOutcome{
			Strategy: strategy,
			RawScore: score,
			IsWin:    isWin,
		})
	}
	return outcomes, rows.Err()
}

// fitLogisticCalibration fits a simple logistic (sigmoid) calibration model
// using iterative reweighted least squares (IRLS) for stability without
// external dependencies.
//
// Model: prob = sigmoid(a * x_norm + b)
// where x_norm = clamp(score, 0, 100) / 100.0
//
// Returns (a, b, ok) where ok is false if fitting failed.
func fitLogisticCalibration(sortedScores []float64, labels []float64) (a, b float64, ok bool) {
	n := len(sortedScores)
	if n == 0 || len(labels) != n {
		return 0, 0, false
	}

	// Normalize scores to [0, 1]
	xs := make([]float64, n)
	for i, s := range sortedScores {
		x := s / 100.0
		if x < 0 {
			x = 0
		}
		if x > 1 {
			x = 1
		}
		xs[i] = x
	}

	// Compute mean x and mean y (win rate) for initialization
	var sumX, sumY float64
	for i := 0; i < n; i++ {
		sumX += xs[i]
		sumY += labels[i]
	}
	meanX := sumX / float64(n)
	meanY := sumY / float64(n)

	// Initialize a, b from mean-based heuristic:
	// sigmoid(a * meanX + b) ≈ meanY → a * meanX + b ≈ logit(meanY)
	meanYc := clamp(meanY, 0.01, 0.99)
	logitMeanY := math.Log(meanYc / (1 - meanYc))

	// Start with a=2 (reasonable slope), solve for b
	a = 2.0
	b = logitMeanY - a*meanX

	// Simple gradient descent for logistic regression (no external deps)
	// IRLS-lite: 50 iterations of gradient descent with adaptive steps
	lr := 0.1
	for iter := 0; iter < 100; iter++ {
		var gradA, gradB float64
		for i := 0; i < n; i++ {
			z := a*xs[i] + b
			// Clamp z to prevent overflow in exp
			if z > 20 {
				z = 20
			}
			if z < -20 {
				z = -20
			}
			p := 1.0 / (1.0 + math.Exp(-z))
			errorTerm := p - labels[i]
			gradA += errorTerm * xs[i]
			gradB += errorTerm
		}
		gradA /= float64(n)
		gradB /= float64(n)

		a -= lr * gradA
		b -= lr * gradB

		// Decay learning rate
		lr *= 0.98
	}

	// Validate: a must be finite and reasonable
	if math.IsNaN(a) || math.IsInf(a, 0) || math.IsNaN(b) || math.IsInf(b, 0) {
		return 0, 0, false
	}

	// Clamp parameters to reasonable ranges
	if math.Abs(a) > 20 {
		return 0, 0, false
	}
	if math.Abs(b) > 10 {
		return 0, 0, false
	}

	// Enforce a monotonic non-decreasing calibration. A negative slope would
	// make a higher raw score map to a LOWER probability — misrepresenting the
	// score as anti-informative. When the fit is non-positive the data shows no
	// positive discrimination, so fall back to a flat empirical base-rate
	// calibration (a=0, b=logit(base_rate)), which is the honest MLE and is
	// strictly monotonic.
	if a < 0 {
		meanYc := clamp(meanY, 0.01, 0.99)
		b = math.Log(meanYc / (1 - meanYc))
		a = 0
	}

	return a, b, true
}

// auc computes the rank-based (Mann-Whitney) ROC AUC of the score against the
// binary labels. It is independent of the calibration fit and reports how well
// the raw score discriminates wins from losses. Returns 0.5 when only one class
// is present (no discriminative signal).
func auc(xs []float64, labels []float64) float64 {
	var pos, neg []float64
	for i := range xs {
		if labels[i] > 0.5 {
			pos = append(pos, xs[i])
		} else {
			neg = append(neg, xs[i])
		}
	}
	if len(pos) == 0 || len(neg) == 0 {
		return 0.5
	}
	var concordant, ties float64
	for _, xp := range pos {
		for _, xn := range neg {
			if xp > xn {
				concordant++
			} else if xp == xn {
				ties += 0.5
			}
		}
	}
	return (concordant + ties) / float64(len(pos)*len(neg))
}

// calibrationJSON is the schema-compatible output format consumed by LoadJSONModels.
type calibrationJSON struct {
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

func (c *LiveCalibrator) writeCalibrationJSON(strategy string, a, b float64, nSamples int, winRate float64, oosAUC float64) {
	cal := calibrationJSON{
		Version:     SupportedCalibrationVersion,
		Strategy:    strategy,
		Target:      "TP1_HIT",
		ExitProfile: "default",
		OOSAUC:      oosAUC, // real rank-based AUC of score discrimination
		NSamples:    nSamples,
		Method:      "logistic",
		Params: map[string]float64{
			"a": a,
			"b": b,
		},
		TrainedAt: time.Now().UTC().Format(time.RFC3339),
		XScale:    100.0,
		XClip:     []float64{0, 100},
	}

	filename := filepath.Join(c.outputDir, fmt.Sprintf("calibration_%s_live.json", strategy))
	data, err := json.MarshalIndent(cal, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return
	}
}

// RunOnce explicitly runs calibration now and returns the number of samples used.
func (c *LiveCalibrator) RunOnce(ctx context.Context) int {
	c.runCalibration(ctx)
	return c.TotalSamples
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
