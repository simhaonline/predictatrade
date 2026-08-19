package features

import (
	"sync"

	"github.com/shopspring/decimal"

	"github.com/predictatrade/realtime/internal/types"
)

// Registry holds all feature engines and coordinates their evaluation.
type Registry struct {
	cotStatusOverride string
	cotReasonOverride string
	mu               sync.Mutex
	structureEngine   *StructureEngine
	liquidityEngine   *LiquidityEngine
	fvgEngine        *FVGEngine
	vwapEngine       *VWAPEngine
	indicatorEngine  *IndicatorEngine
	regimeEngine     *RegimeEngine
	mtfEngine        *MTFEngine
	sessionEngine    *SessionEngine

	// New engines (SOW Sections 5-13)
	sarEngine        *SAREngine
	ichimokuEngine   *IchimokuEngine
	stochRSIEngine   *StochRSIEngine
	fibonacciEngine  *FibonacciEngine
	candleEngine     *CandleEngine
	pivotEngine      *PivotEngine

	// Rolling statistics (SOW Sections 8-11)
	obvZScore        *RollingStats
	tickVolZScore    *RollingStats
	bbWidthZScore    *RollingStats
}

func NewRegistry() *Registry {
	return &Registry{
		structureEngine:  NewStructureEngine(50),
		liquidityEngine:  NewLiquidityEngine(),
		fvgEngine:       NewFVGEngine(100),
		vwapEngine:      NewVWAPEngine(),
		indicatorEngine: NewIndicatorEngine(200),
		regimeEngine:    NewRegimeEngine(),
		mtfEngine:       NewMTFEngine(),
		sessionEngine:   NewSessionEngine(),

		// New indicator engines
		sarEngine:       NewSAREngine(0.02, 0.20),
		ichimokuEngine:  NewIchimokuEngine(9, 26, 52, 26),
		stochRSIEngine:  NewStochRSIEngine(14, 14, 3, 3),
		fibonacciEngine: NewFibonacciEngine(nil),
		pivotEngine:     NewPivotEngine(),
		candleEngine:     NewCandleEngine(),

		// Rolling statistics — 50-bar window, min 20 samples
		obvZScore:       NewRollingStats(50, 20),
		tickVolZScore:   NewRollingStats(50, 20),
		bbWidthZScore:   NewRollingStats(50, 20),
	}
}

// Evaluate processes a new candle through all feature engines and returns updated MarketState.
func (r *Registry) Evaluate(candle *types.Candle, allCandles map[types.Timeframe]*types.Candle, lastTick *types.Tick) *MarketState {
	r.mu.Lock()
	defer r.mu.Unlock()

	if candle == nil {
		return nil
	}

	structure := r.structureEngine.Process(candle)
	liquidity := r.liquidityEngine.Process(candle, structure.SwingHighs)
	fvg := r.fvgEngine.Process(candle)
	vwap := r.vwapEngine.Process(candle)
	indicators := r.indicatorEngine.Process(candle)
	regime := r.regimeEngine.Process(candle, indicators)
	mtf := r.mtfEngine.Process(allCandles)
	session := r.sessionEngine.Process(candle.Time)

	// New indicators (SOW Sections 5-7)
	sar := r.sarEngine.Process(candle)
	ichimoku := r.ichimokuEngine.Process(candle)
	stochRSI := r.stochRSIEngine.Process(candle)

	// Wire new indicator values into IndicatorFeatures
	indicators.ParabolicSAR = sar.Value
	indicators.ParabolicSARLong = sar.IsLong
	indicators.IchimokuTenkan = ichimoku.Tenkan
	indicators.IchimokuKijun = ichimoku.Kijun
	indicators.IchimokuSenkouA = ichimoku.SenkouA
	indicators.IchimokuSenkouB = ichimoku.SenkouB
	indicators.IchimokuChikou = ichimoku.Chikou
	indicators.IchimokuCloudTop = ichimoku.CloudTop
	indicators.IchimokuCloudBot = ichimoku.CloudBottom
	indicators.IchimokuAboveCloud = ichimoku.AboveCloud
	indicators.IchimokuBelowCloud = ichimoku.BelowCloud
	indicators.IchimokuInCloud = ichimoku.InCloud
	indicators.StochRSI = stochRSI.Raw
	indicators.StochRSIK = stochRSI.K
	indicators.StochRSID = stochRSI.D

	// Rolling Z-scores (SOW Sections 8-10)
	if !indicators.OBV.IsZero() {
		obvF, _ := indicators.OBV.Float64()
		r.obvZScore.Add(obvF)
		indicators.OBVZScore = r.obvZScore.ZScoreDecimal(indicators.OBV)
	}
	if candle.Volume > 0 {
		r.tickVolZScore.Add(float64(candle.Volume))
		indicators.TickVolumeZScore = decimal.NewFromFloat(r.tickVolZScore.ZScore(float64(candle.Volume)))
	}
	if !indicators.BollWidth.IsZero() {
		bwF, _ := indicators.BollWidth.Float64()
		r.bbWidthZScore.Add(bwF)
		indicators.BBWidthZScore = r.bbWidthZScore.ZScoreDecimal(indicators.BollWidth)
	}

	// Fibonacci retracement (SOW Section 12)
	fib := r.fibonacciEngine.Process(candle, structure)

	// Daily/Weekly pivots (SOW Section 13)
	pivots := r.pivotEngine.Process(candle)

	// Build feature readiness map (SOW Section 27)
	readiness := r.buildFeatureReadiness(structure, indicators, sar, ichimoku, stochRSI, fib, pivots, candle)

	state := &MarketState{
		Symbol:     candle.Symbol,
		Timestamp:  candle.Time,
		LastTick:    lastTick,
		CurrentPrice: candle.Close,
		Bid:        candle.Low,
		Ask:        candle.High,
		Spread:     candle.High.Sub(candle.Low),
		Mid:        candle.Close,
		Candles:    allCandles,
		Structure:  structure,
		Liquidity:  liquidity,
		FVG:        fvg,
		VWAP:       vwap,
		Indicators:  indicators,
		Regime:     regime,
		MTF:        mtf,
		Session:    session,
		Candle:     r.candleEngine.Process(candle, indicators.ATR),
		Fibonacci:  fib,
		Pivots:     pivots,
		FeatureReadiness: readiness,
		Quality:     types.QualityAuthoritative,
	}

	if lastTick != nil {
		state.Bid = lastTick.Bid
		state.Ask = lastTick.Ask
		state.Spread = lastTick.Spread
		state.Mid = lastTick.Mid
		state.CurrentPrice = lastTick.Mid
		state.Quality = lastTick.Quality
	}

	return state
}

// buildFeatureReadiness creates a readiness map for all features (SOW Section 27).
// SetCOTStatus sets the COT provider status override for feature readiness reporting.
func (r *Registry) SetCOTStatus(status, reason string) {
	r.cotStatusOverride = status
	r.cotReasonOverride = reason
}

func (r *Registry) buildFeatureReadiness(structure StructureFeatures, ind IndicatorFeatures, sar SARFeatures, ichimoku IchimokuFeatures, stochRSI StochRSIFeatures, fib FibonacciFeatures, pivots PivotFeatures, candle *types.Candle) map[string]FeatureReadiness {
	readiness := make(map[string]FeatureReadiness)

	// Trend indicators
	readiness["EMA9"] = simpleReadiness("READY", "computed", "local")
	readiness["EMA21"] = simpleReadiness("READY", "computed", "local")
	readiness["EMA50"] = simpleReadiness("READY", "computed", "local")
	readiness["MACD"] = simpleReadiness("READY", "computed", "local")
	readiness["ADX"] = simpleReadiness("READY", "computed", "local")

	// Parabolic SAR
	if sar.Ready {
		readiness["ParabolicSAR"] = simpleReadiness("READY", "computed", "local")
	} else {
		readiness["ParabolicSAR"] = simpleReadiness("WARMING_UP", "insufficient history", "local")
	}

	// Ichimoku
	if ichimoku.Ready {
		readiness["Ichimoku"] = simpleReadiness("READY", "computed", "local")
	} else {
		readiness["Ichimoku"] = FeatureReadiness{
			State: "WARMING_UP", Reason: "needs " + itoa(52+26) + " bars",
			RequiredHistory: 78, CurrentHistory: r.ichimokuEngine.barCount(),
			Source: "local",
		}
	}

	// StochRSI
	if stochRSI.Ready {
		readiness["StochRSI"] = simpleReadiness("READY", "computed", "local")
	} else {
		readiness["StochRSI"] = simpleReadiness("WARMING_UP", "needs RSI history", "local")
	}

	// Volume indicators
	readiness["OBV"] = simpleReadiness("READY", "tick_volume", "local")
	if r.obvZScore.Ready() {
		readiness["OBV_ZScore"] = simpleReadiness("READY", "computed", "local")
	} else {
		readiness["OBV_ZScore"] = simpleReadiness("WARMING_UP", "rolling stats warming up", "local")
	}
	if r.tickVolZScore.Ready() {
		readiness["TickVolume_ZScore"] = simpleReadiness("READY", "computed", "local")
	} else {
		readiness["TickVolume_ZScore"] = simpleReadiness("WARMING_UP", "rolling stats warming up", "local")
	}
	if r.bbWidthZScore.Ready() {
		readiness["BBWidth_ZScore"] = simpleReadiness("READY", "computed", "local")
	} else {
		readiness["BBWidth_ZScore"] = simpleReadiness("WARMING_UP", "rolling stats warming up", "local")
	}

	// Structure
	if len(structure.SwingHighs) > 0 && len(structure.SwingLows) > 0 {
		readiness["Structure"] = simpleReadiness("READY", "BOS/CHoCH active", "local")
	} else {
		readiness["Structure"] = simpleReadiness("INSUFFICIENT_HISTORY", "no confirmed swings", "local")
	}

	// Fibonacci
	if fib.Ready {
		readiness["Fibonacci"] = simpleReadiness("READY", "confirmed structural swings", "local")
	} else {
		readiness["Fibonacci"] = simpleReadiness("INSUFFICIENT_STRUCTURE", "no valid swing pair", "local")
	}

	// Pivots
	if pivots.Ready {
		readiness["DailyPivots"] = simpleReadiness("READY", "previous completed day", "local")
	} else {
		readiness["DailyPivots"] = simpleReadiness("WARMING_UP", "awaiting first completed day", "local")
	}
	if pivots.Weekly.Ready {
		readiness["WeeklyPivots"] = simpleReadiness("READY", "previous completed week", "local")
	} else {
		readiness["WeeklyPivots"] = simpleReadiness("WARMING_UP", "awaiting first completed week", "local")
	}

	// Volume Profile — UNSUPPORTED_BY_DATA_SOURCE
	readiness["VolumeProfile"] = FeatureReadiness{
		State:  "UNSUPPORTED_BY_DATA_SOURCE",
		Reason: "broker provides tick volume, not real exchange volume",
		Source: "broker",
	}

	// Cumulative Delta — UNSUPPORTED_BY_DATA_SOURCE
	readiness["CumulativeDelta"] = FeatureReadiness{
		State:  "UNSUPPORTED_BY_DATA_SOURCE",
		Reason: "broker tick data cannot establish buy/sell aggressor volume",
		Source: "broker",
	}

	// COT — configurable via FMP_API_KEY. Fails safe if not configured or restricted.
	cotStatus := "EXTERNAL_DEPENDENCY_NOT_CONFIGURED"
	cotReason := "no COT provider configured — set FMP_API_KEY to enable"
	if r.cotStatusOverride != "" {
		cotStatus = r.cotStatusOverride
		cotReason = r.cotReasonOverride
	}
	readiness["COT"] = FeatureReadiness{
		State:  cotStatus,
		Reason: cotReason,
		Source: "external",
	}

	return readiness
}

func simpleReadiness(state, reason, source string) FeatureReadiness {
	return FeatureReadiness{State: state, Reason: reason, Source: source}
}

// itoa is a simple int to string converter to avoid importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// GetSession returns current session info.
func (r *Registry) GetSession() SessionFeatures {
	return r.sessionEngine.Process(currentTime())
}

// barCount returns the number of bars processed by the Ichimoku engine.
func (e *IchimokuEngine) barCount() int {
	return len(e.highs)
}
