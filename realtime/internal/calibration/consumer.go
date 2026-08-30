// Package calibration implements calibration model consumption.
// SOW Section 16: Calibrated probability rather than raw confidence scores.
package calibration

import (
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// Consumer applies calibration models to raw scores.
type Consumer struct {
	models map[types.StrategyID]*CalibrationModel

	// jsonModels holds research-trained calibration models loaded from disk
	// (CALIBRATION_DIR). These are the only source of a calibrated probability
	// that is allowed to reach a subscriber (ProbabilityCalibrated=true).
	jsonModels map[types.StrategyID]jsonModel
}

// CalibrationModel holds calibration parameters for a strategy.
type CalibrationModel struct {
	StrategyID       types.StrategyID
	PredictionTarget string
	SigmoidA         decimal.Decimal // scaling
	SigmoidB         decimal.Decimal // offset
	BrierScore       decimal.Decimal
	ECE              decimal.Decimal
	SampleSize       int64
	WilsonLower      decimal.Decimal
	IsActive         bool
	Status           string // UNVERIFIED, SHADOW, VALIDATED, PROMOTED
}

func NewConsumer() *Consumer {
	return &Consumer{
		models: make(map[types.StrategyID]*CalibrationModel),
	}
}

// SetModel sets the calibration model for a strategy.
func (c *Consumer) SetModel(model *CalibrationModel) {
	c.models[model.StrategyID] = model
}

// minCalibratedOOSAUC is the floor a calibration model must meet (along with
// monotonicity) before it is promoted to VALIDATED and allowed to surface a
// probability to subscribers. The previous 0.5 floor admitted coin-flip models;
// 0.52 is the minimum honest out-of-sample discriminative bar for live promotion
// (a 37-sample model that previously slipped through is now rejected by the
// sample-size gate below as well).
const minCalibratedOOSAUC = 0.52

// minCalibratedSampleSize is the minimum number of resolved outcomes a
// calibration model must be trained on before it is promoted to VALIDATED
// and allowed to surface a probability to subscribers. Small-sample fits are
// statistically untrustworthy and must not reach a subscriber as a calibrated
// probability (BE-5 / MACRO_AUDIT 2.6).
const minCalibratedSampleSize = 100

// Calibrate converts a raw score (0-100) to a calibrated probability (0-1).
// SOW Section 16: calibrated_probability = sigmoid(a * (raw_score/100) + b)
//
// CRITICAL: The raw score MUST be clamped to [0, 100] before normalization.
// The strategy scoring engine scales evidence contributions by 100, and with
// many indicators firing the sum can exceed 100 (e.g., 209.7). Feeding an
// unbounded value into the sigmoid saturates it to ~99% regardless of the
// actual evidence quality, producing a meaningless "probability".
// Clamping ensures the sigmoid input stays in the designed [0, 1] range.
//
// CRITICAL — NO FABRICATION (AGENTS.md / BE-5): a probability is only returned
// when a VALIDATED calibration model is active for the strategy. When no
// validated model is available (including PROVISIONAL/seed placeholders, or
// models that failed the OOS_AUC/monotonicity gate in the loader) Calibrate
// returns (decimal.Zero, false). Callers MUST treat ok==false as "no valid
// probability" and must NOT surface a synthetic or placeholder value.
func (c *Consumer) Calibrate(strategyID types.StrategyID, rawScore decimal.Decimal) (decimal.Decimal, bool) {
	// Clamp raw score to [0, 100] — scores outside this range are not meaningful
	// as calibration inputs (the sigmoid model was designed for 0-100 scale).
	clamped := rawScore
	if clamped.LessThan(decimal.Zero) {
		clamped = decimal.Zero
	}
	hundred := decimal.NewFromInt(100)
	if clamped.GreaterThan(hundred) {
		clamped = hundred
	}

	model, ok := c.models[strategyID]
	// Gate on a genuinely VALIDATED (or PROMOTED-from-validated) model only.
	// Seed/PROVISIONAL models are deliberately NOT eligible: showing a
	// probability derived from an unvalidated model would fabricate confidence.
	if !ok || !model.IsActive ||
		(model.Status != "VALIDATED" && model.Status != "PROMOTED") {
		return decimal.Zero, false
	}

	// Sigmoid calibration: x = clampedScore / 100 ∈ [0, 1]
	x := clamped.Div(hundred)
	scaled := model.SigmoidA.Mul(x).Add(model.SigmoidB)
	// sigmoid(scaled) = 1 / (1 + exp(-scaled))
	scaledF, _ := scaled.Float64()
	prob := 1.0 / (1.0 + mathExp(-scaledF))
	return decimal.NewFromFloat(prob), true
}

// GetModel returns the calibration model for a strategy.
func (c *Consumer) GetModel(strategyID types.StrategyID) *CalibrationModel {
	return c.models[strategyID]
}

// SeedDefaultModels creates reasonable default calibration models.
func (c *Consumer) SeedDefaultModels() {
	defaults := []struct {
		id   types.StrategyID
		a, b float64
	}{
		{types.StrategyStandardScalping, 2.5, -0.5},
		{types.StrategyUltraScalping, 3.0, -0.8},
		{types.StrategyStandardSwing, 2.0, -0.3},
		{types.StrategyTrendSwing, 1.8, -0.2},
	}
	for _, d := range defaults {
		c.SetModel(&CalibrationModel{
			StrategyID:       d.id,
			PredictionTarget: "TP1_HIT",
			SigmoidA:         decimal.NewFromFloat(d.a),
			SigmoidB:         decimal.NewFromFloat(d.b),
			BrierScore:       decimal.NewFromFloat(0.21),
			ECE:              decimal.NewFromFloat(0.05),
			SampleSize:       100,
			WilsonLower:      decimal.NewFromFloat(0.45),
			IsActive:         true,
			Status:           "PROVISIONAL", // Default models — NOT statistically validated, replaced by trained models when OOS data available
		})
	}
}
