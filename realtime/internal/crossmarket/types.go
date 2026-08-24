// Package crossmarket implements the Cross-Market Macro & Intermarket Confluence Engine.
//
// This module is a CONFIRMATION LAYER, NOT a trading trigger.
// It consumes external market data (DXY, EURUSD, yields, VIX, COT, etc.),
// normalizes each driver to a bounded impact score (-100..+100),
// measures agreement/conflict, detects correlation regime shifts,
// and produces a bounded confluence score that ADJUSTS existing signal
// confidence — it never generates BUY/SELL by itself.
//
// Fail-safe: If any provider is unavailable, the engine degrades gracefully
// with lower confidence and missing-driver warnings. Signal generation
// is NEVER blocked by this module — hard gates remain authoritative.
package crossmarket

import "time"

// DriverName identifies a macro/cross-market driver.
type DriverName string

const (
	DriverDXY        DriverName = "dxy"
	DriverEURUSD     DriverName = "eurusd"
	DriverRealYields DriverName = "real_yields"
	DriverFedContext DriverName = "fed_context"
	DriverVIX        DriverName = "vix"
	DriverCOT        DriverName = "cot"
	DriverBTC        DriverName = "btc"
	DriverOil        DriverName = "oil"
	DriverUSDJPY     DriverName = "usdjpy"
	DriverUSDCHF     DriverName = "usdchf"
	DriverETF        DriverName = "etf_flows"
)

// DataQuality represents the state of a data feed.
type DataQuality string

const (
	QualityConnected DataQuality = "CONNECTED"
	QualityDegraded  DataQuality = "DEGRADED"
	QualityStale     DataQuality = "STALE"
	QualityMissing   DataQuality = "MISSING"
	QualityError     DataQuality = "ERROR"
)

// Direction represents the influence direction on XAUUSD.
type Direction string

const (
	DirBullish Direction = "BULLISH"
	DirBearish Direction = "BEARISH"
	DirNeutral Direction = "NEUTRAL"
)

// DivergenceSeverity represents how strongly drivers diverge.
type DivergenceSeverity string

const (
	DivNone     DivergenceSeverity = "NONE"
	DivLow      DivergenceSeverity = "LOW"
	DivModerate DivergenceSeverity = "MODERATE"
	DivHigh     DivergenceSeverity = "HIGH"
	DivExtreme  DivergenceSeverity = "EXTREME"
)

// SafeHavenRegime represents the current safe-haven state.
type SafeHavenRegime string

const (
	SHNORMAL        SafeHavenRegime = "NORMAL"
	SHRiskOn        SafeHavenRegime = "RISK_ON"
	SHRiskOff       SafeHavenRegime = "RISK_OFF"
	SHSafeHavenGold SafeHavenRegime = "SAFE_HAVEN_GOLD"
	SHSafeHavenUSD  SafeHavenRegime = "SAFE_HAVEN_USD"
	SHDualSafeHaven SafeHavenRegime = "DUAL_SAFE_HAVEN"
	SHLiquidityStress SafeHavenRegime = "LIQUIDITY_STRESS"
	SHMixed         SafeHavenRegime = "MIXED"
	SHUnknown       SafeHavenRegime = "UNKNOWN"
)

// CorrelationRegime represents the state of cross-asset correlations.
type CorrelationRegime string

const (
	CorrNormal       CorrelationRegime = "NORMAL_CORRELATION"
	CorrWeak         CorrelationRegime = "WEAK_CORRELATION"
	CorrInverse      CorrelationRegime = "INVERSE_CORRELATION"
	CorrBreakdown    CorrelationRegime = "CORRELATION_BREAKDOWN"
	CorrRegimeShift  CorrelationRegime = "REGIME_SHIFT"
	CorrInsufficient CorrelationRegime = "INSUFFICIENT_DATA"
)

// EventRiskLevel represents macro event risk.
type EventRiskLevel string

const (
	EventNormal   EventRiskLevel = "NORMAL"
	EventElevated EventRiskLevel = "ELEVATED"
	EventHigh     EventRiskLevel = "HIGH"
	EventExtreme  EventRiskLevel = "EXTREME"
)

// Mode controls the module's influence on production signals.
type Mode string

const (
	ModeDisabled Mode = "disabled"
	ModeShadow   Mode = "shadow"
	ModeActive   Mode = "active"
)

// DriverSnapshot is a single normalized driver reading.
type DriverSnapshot struct {
	Name            DriverName   `json:"name"`
	RawValue        float64      `json:"raw_value"`
	NormalizedValue float64      `json:"normalized_value"` // -100 to +100
	ImpactScore     float64      `json:"impact_score"`     // -100 to +100
	Direction       Direction    `json:"direction"`
	Confidence      float64      `json:"confidence"`       // 0 to 1
	Freshness       float64      `json:"freshness"`        // 0 to 1
	Quality         DataQuality  `json:"quality"`
	Source          string       `json:"source"`
	Timeframe       string       `json:"timeframe"`
	Reason          string       `json:"reason"`
	Timestamp       time.Time    `json:"timestamp"`
	BaseWeight      float64      `json:"base_weight"`
	EffectiveWeight float64      `json:"effective_weight"`
}

// ConfluenceResult is the final output of the confluence engine.
type ConfluenceResult struct {
	Score              float64             `json:"score"`               // -100 to +100
	Direction          Direction           `json:"direction"`
	Confidence         float64             `json:"confidence"`          // 0 to 1
	Agreement          float64             `json:"agreement"`           // 0 to 1
	Conflict           float64             `json:"conflict"`            // 0 to 1
	DataQuality        DataQuality         `json:"data_quality"`
	Regime             SafeHavenRegime     `json:"regime"`
	EventRisk          EventRiskLevel      `json:"event_risk"`
	CorrelationRegime  CorrelationRegime   `json:"correlation_regime"`
	PrimaryDrivers     []string            `json:"primary_drivers"`
	OpposingDrivers    []string            `json:"opposing_drivers"`
	MissingDrivers     []string            `json:"missing_drivers"`
	Warnings           []string            `json:"warnings"`
	DivergenceSeverity DivergenceSeverity  `json:"divergence_severity"`
	ScoreAdjustment    float64             `json:"score_adjustment"`    // bounded adjustment
	Mode               Mode                `json:"mode"`
	ModelVersion       string              `json:"model_version"`
	WeightsVersion     string              `json:"weights_version"`
	DriverSnapshot     []DriverSnapshot    `json:"driver_snapshot"`
	Timestamp          time.Time           `json:"timestamp"`
}

// Config holds all cross-market confluence configuration.
type Config struct {
	Enabled            bool
	Mode               Mode
	MaxBonus           float64 // max positive score adjustment
	MaxPenalty         float64 // max negative score adjustment
	DXYEnabled         bool
	EURUSDEnabled      bool
	RealYieldsEnabled  bool
	VIXEnabled         bool
	COTEnabled         bool
	BTCEnabled         bool
	OilEnabled         bool
	USDJPYEnabled      bool
	ETFEnabled         bool
	EventBlackoutEnabled bool
	// Weights (Tier 1 = HIGH, Tier 2 = MEDIUM, Tier 3 = LOW)
	Weights map[DriverName]float64
	// Freshness TTLs in seconds
	FreshnessTTL map[DriverName]int
	// Correlation window
	CorrelationWindow int
}

// DefaultConfig returns safe defaults — SHADOW mode, zero production impact.
func DefaultConfig() Config {
	return Config{
		Enabled:              true,
		Mode:                 ModeShadow,
		MaxBonus:             10.0,
		MaxPenalty:           -15.0,
		DXYEnabled:           true,
		EURUSDEnabled:        true,
		RealYieldsEnabled:    false, // no TIPS feed configured yet
		VIXEnabled:           false, // no VIX feed configured yet
		COTEnabled:           true,
		BTCEnabled:           false, // no BTC feed configured yet
		OilEnabled:           false,
		USDJPYEnabled:        false,
		ETFEnabled:           false,
		EventBlackoutEnabled: true,
		Weights: map[DriverName]float64{
			DriverDXY:        25.0, // HIGH — primary USD driver
			DriverEURUSD:     10.0, // MEDIUM — confirmation (anti-double-count with DXY)
			DriverRealYields: 25.0, // HIGH — but disabled until feed available
			DriverFedContext: 15.0, // HIGH — but disabled until feed available
			DriverVIX:        10.0, // MEDIUM
			DriverCOT:         8.0, // MEDIUM for swing
			DriverBTC:         2.0, // LOW
			DriverOil:         3.0, // LOW
			DriverUSDJPY:      5.0, // LOW-MEDIUM
			DriverUSDCHF:      5.0, // LOW-MEDIUM
			DriverETF:          5.0, // MEDIUM/LOW
		},
		FreshnessTTL: map[DriverName]int{
			DriverDXY:        300,   // 5 minutes
			DriverEURUSD:     300,
			DriverRealYields: 3600,  // 1 hour
			DriverFedContext: 3600,
			DriverVIX:        300,
			DriverCOT:        604800, // 7 days (weekly data)
			DriverBTC:        300,
			DriverOil:        600,
			DriverUSDJPY:     300,
			DriverETF:        86400,  // 1 day
		},
		CorrelationWindow: 50,
	}
}
