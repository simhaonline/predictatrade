package crossmarket

import (
	"math"
)

// LeadLagDetector performs cross-correlation lag analysis to determine
// whether another market is presently leading XAUUSD.
// It does NOT claim an asset "leads Gold" unless current data demonstrates it.
type LeadLagDetector struct {
	maxLag int
}

func NewLeadLagDetector(maxLag int) *LeadLagDetector {
	if maxLag <= 0 {
		maxLag = 10
	}
	return &LeadLagDetector{maxLag: maxLag}
}

// LeadLagResult holds the result of a lag analysis.
type LeadLagResult struct {
	LeadingAsset string  `json:"leading_asset"`
	Lag          int     `json:"lag"`           // positive = asset leads XAU by N bars
	Direction    string  `json:"direction"`      // INVERSE, DIRECT, NEUTRAL
	Coefficient  float64 `json:"coefficient"`    // cross-correlation at best lag
	Confidence   float64 `json:"confidence"`     // 0-1
	SampleCount  int     `json:"sample_count"`
}

// Analyze computes cross-correlation at multiple lag offsets and returns the best.
func (d *LeadLagDetector) Analyze(xauReturns, assetReturns []float64, assetName string) LeadLagResult {
	n := len(xauReturns)
	if n < 20 || len(assetReturns) < 20 {
		return LeadLagResult{LeadingAsset: assetName, Confidence: 0, SampleCount: n}
	}

	minLen := n
	if len(assetReturns) < minLen {
		minLen = len(assetReturns)
	}

	bestCoeff := 0.0
	bestLag := 0
	bestDir := "NEUTRAL"

	for lag := -d.maxLag; lag <= d.maxLag; lag++ {
		var coeff float64
		if lag >= 0 {
			// Asset leads XAU by `lag` bars
			if lag >= minLen {
				continue
			}
			coeff = pearsonCorrelation(xauReturns[lag:], assetReturns[:minLen-lag])
		} else {
			// XAU leads asset by |lag| bars
			absLag := -lag
			if absLag >= minLen {
				continue
			}
			coeff = pearsonCorrelation(xauReturns[:minLen-absLag], assetReturns[absLag:])
		}

		if math.IsNaN(coeff) {
			continue
		}

		if math.Abs(coeff) > math.Abs(bestCoeff) {
			bestCoeff = coeff
			bestLag = lag
			if bestCoeff > 0.3 {
				bestDir = "DIRECT"
			} else if bestCoeff < -0.3 {
				bestDir = "INVERSE"
			} else {
				bestDir = "NEUTRAL"
			}
		}
	}

	confidence := 0.0
	if math.Abs(bestCoeff) > 0.5 && minLen > 30 {
		confidence = math.Abs(bestCoeff) * 0.8
	} else if math.Abs(bestCoeff) > 0.3 {
		confidence = math.Abs(bestCoeff) * 0.5
	}
	confidence = math.Min(1.0, confidence)

	return LeadLagResult{
		LeadingAsset: assetName,
		Lag:          bestLag,
		Direction:    bestDir,
		Coefficient:  bestCoeff,
		Confidence:   confidence,
		SampleCount:  minLen,
	}
}
