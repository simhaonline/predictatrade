// Package ptb — MarketIntelligenceSnapshot and data authenticity guard.
// Stage 4 Sections 1-4, 7: Live-data authenticity, provenance, PTB output contract.
package ptb

import (
	"time"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// MarketIntelligenceSnapshot is the shared evidence layer provided to all four
// strategy engines. Stage 4 Section 7.
//
// CRITICAL: When any module is in SHADOW mode, its values are calculated and
// persisted for observation, but contribute ZERO to strategy scores.
type MarketIntelligenceSnapshot struct {
	Timestamp        time.Time              `json:"timestamp"`
	SourceSnapshotID  string                `json:"source_snapshot_id"`
	DataSource       types.DataSourceType  `json:"data_source"`
	IsLive            bool                  `json:"is_live"`
	DataAgeMs         int64                 `json:"data_age_ms"`

	// Data quality (Section 41)
	DataQuality       DataQualityResult     `json:"data_quality"`

	// Module results — each carries provenance + availability
	LiquidityVoid       ModuleResult         `json:"liquidity_void"`
	WickFill            ModuleResult         `json:"wick_fill"`
	SessionImbalance    ModuleResult         `json:"session_imbalance"`
	CandleRangeProjector ModuleResult        `json:"candle_range_projector"`
	TimeAtMode          ModuleResult         `json:"time_at_mode"`
	EngineeredLiquidity ModuleResult         `json:"engineered_liquidity_proxy"`
	MarketPhase         ModuleResult         `json:"market_phase"`
	RelativeVolumeFlow  ModuleResult         `json:"relative_tick_volume_flow"`
	PriceDelivery       ModuleResult         `json:"price_delivery"`
	StopHuntProxy       ModuleResult         `json:"stop_hunt_proxy"`
	InstitutionalFootprint ModuleResult     `json:"institutional_footprint"`
	TimeCycle           ModuleResult         `json:"time_cycle_analytics"`
	AlgoActivity        ModuleResult         `json:"algo_activity_proxy"`
	CompleteLiquidityMap ModuleResult       `json:"complete_liquidity_map"`
	ManipulationProxy   ModuleResult         `json:"manipulation_proxy"`
	MTFBias             MTFBiasResult        `json:"mtf_bias"`
	VolatilityRegime    VolatilityRegimeResult `json:"volatility_regime"`
	SRQuality           SRQualityResult      `json:"sr_quality"`
	Microstructure      MicrostructureResult `json:"microstructure"`

	// Feature availability map for observability
	FeatureAvailability map[string]string `json:"feature_availability"`
}

// ModuleResult is the generic output envelope for any PTB module.
// Stage 4 Section 44: Complete module provenance.
type ModuleResult struct {
	Module        string      `json:"module"`
	Mode          types.ModuleMode `json:"mode"`
	Available     bool        `json:"available"`
	State         string      `json:"state"`     // e.g. "READY", "WARMING_UP", "UNSUPPORTED", "ERROR"
	Value         interface{} `json:"value,omitempty"`
	Provenance    types.ModuleProvenance `json:"provenance"`
	ScoreContrib  decimal.Decimal `json:"score_contrib"` // ALWAYS ZERO in SHADOW mode
}

// DataQualityResult measures real data quality from observable attributes.
// Stage 4 Section 41: Replace fake assumed quality with measured data quality.
type DataQualityResult struct {
	State   string   `json:"data_quality_state"` // EXCELLENT, GOOD, DEGRADED, STALE, ERROR
	Score   float64  `json:"data_quality_score"` // 0.0-1.0 measured, not assumed
	Reasons []string `json:"reasons"`
}

// MTFBiasResult is the enhanced multi-timeframe bias output.
// Stage 4 Section 23.
type MTFBiasResult struct {
	Bias              string  `json:"bias"`           // LONG, SHORT, NEUTRAL, CONFLICTED
	Alignment         float64 `json:"alignment"`     // -1.0 to +1.0
	TrendStrength     float64 `json:"trend_strength"`
	ConflictStrength  float64 `json:"conflict_strength"`
	TimeframeContrib  map[string]float64 `json:"timeframe_contributions"`
	SampleCount       int     `json:"sample_count"`
}

// VolatilityRegimeResult classifies volatility from real history.
// Stage 4 Section 24.
type VolatilityRegimeResult struct {
	Regime        string  `json:"regime"`         // LOW, NORMAL, HIGH, EXTREME
	ATR           float64 `json:"atr"`
	ATRNormalized float64 `json:"atr_normalized"` // ATR/price
	ATRPercentile float64 `json:"atr_percentile"`
	BBWidth       float64 `json:"bb_width"`
	BBWidthPct    float64 `json:"bb_width_percentile"`
	SampleCount   int     `json:"sample_count"`
}

// SRQualityResult evaluates support/resistance quality.
// Stage 4 Section 25.
type SRQualityResult struct {
	Levels  []SRLevel `json:"levels"`
	SampleCount int `json:"sample_count"`
}

// SRLevel represents a single support/resistance level with quality metrics.
type SRLevel struct {
	Price         decimal.Decimal `json:"price"`
	Type          string  `json:"type"`        // SWING_HIGH, SWING_LOW, PIVOT, FIB, ROUND
	TouchCount    int     `json:"touch_count"`
	AgeBars       int     `json:"age_bars"`
	RejectionMag  float64 `json:"rejection_magnitude"`
	BreakHistory  string  `json:"break_history"` // NEVER_TESTED, FAILED_BREAK, SUCCESSFUL_BREAK
	Distance      float64 `json:"distance_from_price"`
	QualityScore  float64 `json:"quality_score"` // 0.0-1.0
}

// MicrostructureResult measures tick-level microstructure.
// Stage 4 Section 26.
type MicrostructureResult struct {
	Spread           float64 `json:"spread"`
	SpreadPercentile float64 `json:"spread_percentile"`
	TickArrivalRate  float64 `json:"tick_arrival_rate"`
	UpTickBalance    float64 `json:"up_tick_down_tick_balance"` // -1.0 to +1.0
	PriceVelocity    float64 `json:"price_velocity"`
	MicroVolatility  float64 `json:"micro_volatility"`
	SampleCount      int     `json:"sample_count"`
}

// DataAuthenticityGuard enforces that production signal generation only uses
// live Master Node data. Stage 4 Sections 1-4.
type DataAuthenticityGuard struct {
	allowedSources map[types.DataSourceType]bool
}

// NewDataAuthenticityGuard creates a guard that only allows LIVE_AGENT
// for production signal generation. REPLAY is allowed for backtesting only.
func NewDataAuthenticityGuard() *DataAuthenticityGuard {
	return &DataAuthenticityGuard{
		allowedSources: map[types.DataSourceType]bool{
			types.DataSourceLiveAgent: true,
			types.DataSourceReplay:         true, // For backtesting/research only
		},
	}
}

// CheckSource verifies the data source is production-valid.
// Returns false for TEST, MOCK, DEMO, FIXTURE, SYNTHETIC, PLACEHOLDER, UNKNOWN.
func (g *DataAuthenticityGuard) CheckSource(src types.DataSourceType) bool {
	return g.allowedSources[src]
}

// RejectReason returns a human-readable reason for rejected sources.
func (g *DataAuthenticityGuard) RejectReason(src types.DataSourceType) string {
	if g.CheckSource(src) {
		return ""
	}
	return "PRODUCTION_SIGNAL_REJECTED: data source " + string(src) + " is not LIVE_AGENT"
}

// EvaluateDataQuality measures real data quality from observable attributes.
// Stage 4 Section 41: No arbitrary quality=90 without measured inputs.
func EvaluateDataQuality(state *features.MarketState, now time.Time) DataQualityResult {
	result := DataQualityResult{
		State: "EXCELLENT",
		Score: 1.0,
	}

	if state == nil {
		result.State = "ERROR"
		result.Score = 0.0
		result.Reasons = append(result.Reasons, "NO_MARKET_STATE")
		return result
	}

	// Check freshness
	if !state.Timestamp.IsZero() {
		ageMs := now.Sub(state.Timestamp).Milliseconds()
		if ageMs < 0 {
			ageMs = 0
		}
		if ageMs > 30000 { // >30s
			result.State = "STALE"
			result.Score = 0.2
			result.Reasons = append(result.Reasons, "STALE_DATA")
		} else if ageMs > 10000 { // >10s
			result.State = "DEGRADED"
			result.Score = 0.5
			result.Reasons = append(result.Reasons, "DATA_AGE_HIGH")
		}
	}

	// Check tick quality
	if state.LastTick != nil {
		if state.LastTick.Quality == types.QualityStale {
			result.State = "STALE"
			result.Score = 0.2
			result.Reasons = append(result.Reasons, "TICK_STALE")
		} else if state.LastTick.Quality == types.QualityInvalid {
			result.State = "ERROR"
			result.Score = 0.0
			result.Reasons = append(result.Reasons, "TICK_INVALID")
		} else if state.LastTick.Quality == types.QualityUnavailable {
			result.State = "ERROR"
			result.Score = 0.0
			result.Reasons = append(result.Reasons, "TICK_UNAVAILABLE")
		}
	}

	// Check spread sanity
	spread, _ := state.Spread.Float64()
	if spread < 0 {
		result.State = "ERROR"
		result.Score = 0.0
		result.Reasons = append(result.Reasons, "NEGATIVE_SPREAD")
	}

	// Check ATR availability (required for SL/TP)
	if state.Indicators.ATR.IsZero() {
		if result.Score > 0.5 {
			result.State = "DEGRADED"
			result.Score = 0.5
		}
		result.Reasons = append(result.Reasons, "ATR_UNAVAILABLE")
	}

	// Check required history completeness
	if len(state.Candles) == 0 {
		result.State = "ERROR"
		result.Score = 0.0
		result.Reasons = append(result.Reasons, "NO_CANDLES")
	}

	return result
}
