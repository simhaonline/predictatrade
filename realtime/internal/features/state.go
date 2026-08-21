// Package features implements quantitative feature engines for the realtime pipeline.
// SOW Sections 9, 10, 11, 12, 132, 133
package features

import (
	"sync"
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// MarketState is the normalized current state of the market for a symbol.
type MarketState struct {
	Symbol       string
	Timestamp    time.Time
	Version      string
	LastTick     *types.Tick
	CurrentPrice decimal.Decimal
	Bid          decimal.Decimal
	Ask          decimal.Decimal
	Spread       decimal.Decimal
	Mid          decimal.Decimal

	Candles map[types.Timeframe]*types.Candle

	Structure   StructureFeatures
	Liquidity   LiquidityFeatures
	FVG         FVGFeatures
	VWAP        VWAPFeatures
	Indicators  IndicatorFeatures
	Regime      RegimeFeatures
	MTF         MTFFeatures
	Session     SessionFeatures
	Candle      CandleIntelligence

	// Fibonacci Retracement — locally computed from confirmed structure (SOW Section 12)
	Fibonacci   FibonacciFeatures
	// Daily/Weekly Pivots — locally computed from previous completed periods (SOW Section 13)
	Pivots      PivotFeatures

	// Feature readiness states (SOW Section 27)
	FeatureReadiness map[string]FeatureReadiness

	Quality types.QualityState

	// PTB — Professional Trader Brain shared intelligence layer
	// Stage 4: Advanced market intelligence. All modules start in SHADOW mode
	// with ZERO score impact until validated and explicitly activated.
	// Uses interface{} to avoid import cycle (features ← ptb ← features).
	// Actual type: *ptb.MarketIntelligenceSnapshot
	PTB interface{}
}

// FeatureReadiness represents the readiness state of a single feature.
// SOW Section 27: standardized feature health/readiness.
type FeatureReadiness struct {
	State       string // READY, WARMING_UP, INSUFFICIENT_HISTORY, INSUFFICIENT_STRUCTURE, STALE, EXTERNAL_DEPENDENCY_NOT_CONFIGURED, UNSUPPORTED_BY_DATA_SOURCE, ERROR
	Reason      string
	LastValid   time.Time
	RequiredHistory int
	CurrentHistory  int
	Source      string
}

// StructureFeatures holds market structure analysis.
type StructureFeatures struct {
	SwingHighs     []decimal.Decimal
	SwingLows      []decimal.Decimal
	LastBOS        *StructureEvent
	LastCHoCH      *StructureEvent
	LastMSS        *StructureEvent
	CurrentTrend   string
	StructureBreak bool
}

type StructureEvent struct {
	Type      string
	Direction string
	Price     decimal.Decimal
	Time      time.Time
}

// LiquidityFeatures holds liquidity analysis.
// CVD/DOM are UNAVAILABLE for broker tick data — never fabricated.
type LiquidityFeatures struct {
	Pools             []LiquidityPool
	RecentSweeps      []SweepEvent
	CVDAvailable      bool
	DOMAvailable      bool
	OrderFlowQuality  types.QualityState
}

type LiquidityPool struct {
	Price      decimal.Decimal
	Type       string
	Strength   int
	CreatedAt  time.Time
	Swept      bool
}

type SweepEvent struct {
	Price      decimal.Decimal
	Direction  string
	Time       time.Time
}

// FVGFeatures holds Fair Value Gap and Order Block analysis.
type FVGFeatures struct {
	FVGs        []FVGZone
	IFVGs       []FVGZone
	OrderBlocks []OrderBlock
	Breakers    []OrderBlock
}

type FVGZone struct {
	Upper      decimal.Decimal
	Lower      decimal.Decimal
	Type       string
	Time       time.Time
	Filled     bool
}

type OrderBlock struct {
	Upper      decimal.Decimal
	Lower      decimal.Decimal
	Type       string
	Time       time.Time
	Mitigated  bool
}

type VWAPFeatures struct {
	SessionVWAP  decimal.Decimal
	UpperBand    decimal.Decimal
	LowerBand    decimal.Decimal
	RollingVWAP  decimal.Decimal
}

// IndicatorFeatures holds ALL indicators with provenance tracking.
// 33 scoped features: 13 trend, 4 momentum, 3 volatility, 4 volume,
// 6 liquidity/structure, 2 session, 1 external.
// Indicators marked UNAVAILABLE are not fabricated.
type IndicatorFeatures struct {
	// Trend — computed from candle history or MT5 snapshot
	EMA9        decimal.Decimal
	EMA21       decimal.Decimal
	EMA50       decimal.Decimal
	EMA100      decimal.Decimal // NEW: computed locally
	EMA200      decimal.Decimal // NEW: computed locally
	EMACross921 bool            // NEW: derived — EMA9 > EMA21 = true
	SMA50       decimal.Decimal // NEW: computed locally
	SMA100      decimal.Decimal // NEW: computed locally
	SMA200      decimal.Decimal
	MACDMain      decimal.Decimal
	MACDSignal    decimal.Decimal
	MACDHistogram decimal.Decimal
	MACDBullCross bool // MACD crossed above Signal (prompt.md Section 1.6)
	MACDBearCross bool // MACD crossed below Signal (prompt.md Section 1.6)
	ADX         decimal.Decimal
	ADXPlusDI   decimal.Decimal
	ADXMinusDI  decimal.Decimal

	// Parabolic SAR — locally computed (SOW Section 5)
	ParabolicSAR     decimal.Decimal
	ParabolicSARLong bool

	// Ichimoku Cloud — locally computed (SOW Section 6)
	IchimokuTenkan    decimal.Decimal
	IchimokuKijun     decimal.Decimal
	IchimokuSenkouA   decimal.Decimal
	IchimokuSenkouB   decimal.Decimal
	IchimokuChikou    decimal.Decimal
	IchimokuCloudTop  decimal.Decimal
	IchimokuCloudBot  decimal.Decimal
	IchimokuAboveCloud bool
	IchimokuBelowCloud bool
	IchimokuInCloud    bool

	// Momentum
	RSI         decimal.Decimal
	StochMain   decimal.Decimal
	StochSignal decimal.Decimal

	// Stochastic RSI — locally computed (SOW Section 7)
	StochRSI    decimal.Decimal
	StochRSIK   decimal.Decimal
	StochRSID   decimal.Decimal

	CCI         decimal.Decimal

	// Volatility
	ATR         decimal.Decimal
	BollUpper   decimal.Decimal
	BollLower   decimal.Decimal
	BollMiddle       decimal.Decimal
	BollWidth        decimal.Decimal // (Upper - Lower) / Middle
	BollBullRev      bool // Close crossed back above lower band (prompt.md Section 1.7)
	BollBearRev      bool // Close crossed back below upper band (prompt.md Section 1.7)

	// Volume
	OBV             decimal.Decimal // computed locally from tick volume
	OBVZScore       decimal.Decimal // rolling Z-score of OBV (SOW Section 8)
	TickVolumeZScore decimal.Decimal // rolling Z-score of tick volume (SOW Section 9)
	BBWidthZScore   decimal.Decimal // rolling Z-score of BB width (SOW Section 10)
	// Volume Profile — UNAVAILABLE (requires real volume, broker provides tick volume only)
	// Cumulative Delta — UNAVAILABLE (requires centralized order-flow, broker tick data only)
	VWAP        decimal.Decimal // Session VWAP (from MT5 snapshot)

	// External
	// COT Report — UNAVAILABLE (external feed not connected)
	// cotStatus = "STALE" — does not block signal generation

	// Momentum (additional from MT5)
	Momentum    decimal.Decimal
	OsMA        decimal.Decimal
}

// Provenance tracks data source quality for each feature family.
type FeatureProvenance struct {
	TrendSource     string // "MT5_SNAPSHOT", "LOCAL_COMPUTE", "UNAVAILABLE"
	MomentumSource  string
	VolatilitySource string
	VolumeSource    string // "TICK_VOLUME" (broker), "REAL_VOLUME" (exchange), "UNAVAILABLE"
	StructureSource string
	SessionSource   string
	ExternalSource  string // "COT_STALE", "COT_UNAVAILABLE"
}

type RegimeFeatures struct {
	Current             types.Regime
	Previous            types.Regime
	Volatility          string
	Confidence          float64
	EnteredAt           time.Time
	Age                 time.Duration
	EntryReason         string
	RawRegime           types.Regime
	TransitionCandidate *types.Regime
	TransitionConfidence float64
	ConfirmationCount   int
	RequiredConfirmations int
	HoldReason          string
	RegimeEngineVersion string
}

type MTFFeatures struct {
	Score   float64
	States  map[types.Timeframe]int
}

type SessionFeatures struct {
	CurrentSession string
	IsOverlap      bool
	IsWeekend      bool
	NewsRisk       string
	NextNewsTime   time.Time
}

// CandleIntelligence holds derived candle analysis.
type CandleIntelligence struct {
	BodySize       decimal.Decimal
	UpperWick      decimal.Decimal
	LowerWick      decimal.Decimal
	Range          decimal.Decimal
	BodyRangeRatio decimal.Decimal
	IsBullish      bool
	IsBearish      bool
	IsDoji         bool
	IsPinBar       bool
	IsEngulfing    bool
	IsInsideBar    bool
	IsOutsideBar   bool
	IsDisplacement bool
	IsRejection    bool
	IsBreakout     bool
	IsCompression  bool
	IsExpansion    bool
	ATRNormalized  decimal.Decimal
	ConsecutiveBull int
	ConsecutiveBear int
}

type StateManager struct {
	mu     sync.RWMutex
	states map[string]*MarketState
}

func NewStateManager() *StateManager {
	return &StateManager{states: make(map[string]*MarketState)}
}

func (sm *StateManager) Update(symbol string, update func(*MarketState)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s, ok := sm.states[symbol]
	if !ok {
		s = &MarketState{Symbol: symbol, Candles: make(map[types.Timeframe]*types.Candle)}
		sm.states[symbol] = s
	}
	update(s)
}

func (sm *StateManager) Get(symbol string) *MarketState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.states[symbol]
	if !ok {
		return &MarketState{Symbol: symbol, Candles: make(map[types.Timeframe]*types.Candle)}
	}
	return s
}

func (sm *StateManager) GetAll() []*MarketState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	var result []*MarketState
	for _, s := range sm.states {
		// Clone to prevent concurrent map iteration/write during JSON marshal
		clone := *s
		// Deep-copy map fields that are iterated during JSON marshal
		if s.Candles != nil {
			clone.Candles = make(map[types.Timeframe]*types.Candle, len(s.Candles))
			for k, v := range s.Candles {
				clone.Candles[k] = v
			}
		}
		// PTB is interface{} — nil it out for safe marshaling
		clone.PTB = nil
		result = append(result, &clone)
	}
	return result
}
