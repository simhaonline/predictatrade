// Package ptb — Centralized configuration for all PTB thresholds.
// Stage 4 Section 29: No important thresholds scattered through code.
package ptb

import (
	"os"
	"strconv"
)

// Config holds all PTB configuration with safe defaults.
// All modules default to SHADOW mode — zero production score impact.
type Config struct {
	// Master switch
	Enabled    bool
	ShadowMode bool

	// Minimum thresholds (Section 16)
	MinConfidence  float64
	MinConfluence   float64

	// Grade thresholds (Section 16)
	GradeAPlus float64
	GradeA     float64
	GradeB     float64
	GradeC     float64
	GradeD     float64

	// Position size multipliers (Section 17)
	PosMultAPlus float64
	PosMultA     float64
	PosMultB     float64
	PosMultC     float64
	PosMultD     float64
	PosMultF     float64

	// Stop distance multipliers (Section 18)
	StopMultHighManipulation float64
	StopMultNormal           float64
	StopMultLowVolatility    float64

	// Volatility thresholds (Section 7)
	VolExtremeHigh float64
	VolHigh        float64
	VolNormal      float64
	VolLow         float64

	// Manipulation thresholds (Section 12)
	ManipHighRisk float64
	ManipMedRisk  float64

	// Correlation lookback windows (Section 8)
	CorrShortWindow  int
	CorrMediumWindow int
	CorrLongWindow   int

	// Freshness limits in seconds (Section 21)
	MacroFreshnessSec  int
	MarketFreshnessSec int

	// MTF weights (Section 5) — higher TFs weighted more
	MTFWeightM1  float64
	MTFWeightM5  float64
	MTFWeightM15 float64
	MTFWeightM30 float64
	MTFWeightH1  float64
	MTFWeightH4  float64
	MTFWeightD1  float64

	// Evidence family caps (Section 42)
	FamilyCapTrend       float64
	FamilyCapMomentum    float64
	FamilyCapStructure   float64
	FamilyCapLiquidity   float64
	FamilyCapVolatility  float64
	FamilyCapVolume      float64
	FamilyCapMacro       float64
	FamilyCapSession     float64
	FamilyCapManipulation float64
	FamilyCapML          float64

	// Model version for provenance
	ModelVersion   string
	ConfigVersion  string
}

// DefaultConfig returns safe defaults. All modules SHADOW.
func DefaultConfig() *Config {
	return &Config{
		Enabled:    getEnvBool("PTB_ENABLED", true),
		ShadowMode: getEnvBool("PTB_SHADOW_MODE", true),

		MinConfidence: 65.0,
		MinConfluence: 70.0,

		GradeAPlus: 90.0,
		GradeA:     80.0,
		GradeB:     70.0,
		GradeC:     60.0,
		GradeD:     50.0,

		PosMultAPlus: 1.00,
		PosMultA:     0.80,
		PosMultB:     0.60,
		PosMultC:     0.40,
		PosMultD:     0.20,
		PosMultF:     0.00,

		StopMultHighManipulation: 1.5,
		StopMultNormal:           1.0,
		StopMultLowVolatility:    0.8,

		VolExtremeHigh: 0.005,
		VolHigh:        0.003,
		VolNormal:      0.001,
		VolLow:         0.0005,

		ManipHighRisk: 70.0,
		ManipMedRisk:  40.0,

		CorrShortWindow:  20,
		CorrMediumWindow: 50,
		CorrLongWindow:   100,

		MacroFreshnessSec:  300,
		MarketFreshnessSec: 30,

		MTFWeightM1:  0.5,
		MTFWeightM5:  1.0,
		MTFWeightM15: 1.5,
		MTFWeightM30: 1.0,
		MTFWeightH1:  2.0,
		MTFWeightH4:  2.5,
		MTFWeightD1:  3.0,

		FamilyCapTrend:        0.25,
		FamilyCapMomentum:     0.20,
		FamilyCapStructure:    0.20,
		FamilyCapLiquidity:    0.15,
		FamilyCapVolatility:   0.10,
		FamilyCapVolume:       0.10,
		FamilyCapMacro:        0.15,
		FamilyCapSession:      0.10,
		FamilyCapManipulation: 0.10,
		FamilyCapML:           0.10,

		ModelVersion:  "1.0.0",
		ConfigVersion: "1.0.0",
	}
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v == "true" || v == "1" || v == "TRUE"
}

func getEnvFloat(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}
