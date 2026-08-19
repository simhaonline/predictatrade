package features

import (
	"math"
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// HistoryBootstrap calculates the required candle history for each indicator/strategy
// and manages warm-up state transitions.
// SOW Section 3: Market History / Structure Bootstrap
// required_history = max(indicator_lookback) + safety_margin
type HistoryBootstrap struct {
	// Maximum lookback required across all indicators
	maxLookback int
	// Safety margin (extra bars for rolling state stability)
	safetyMargin int
	// Current candle count
	candleCount int
}

// NewHistoryBootstrap creates a new bootstrap calculator.
func NewHistoryBootstrap() *HistoryBootstrap {
	// Maximum lookback across all indicators:
	// EMA200: 200, SMA200: 200, Ichimoku senkouB+displacement: 52+26=78
	// ADX: 28, Bollinger: 20, RSI: 14, StochRSI: 14+14=28
	// Structure: needs ~50 for swing detection
	// Rolling stats: 50 window
	// Max = 200 (EMA/SMA)
	maxLookback := 200
	safetyMargin := 20 // Extra bars for state stability
	return &HistoryBootstrap{
		maxLookback:  maxLookback,
		safetyMargin: safetyMargin,
	}
}

// RequiredHistory returns the total number of candles needed for full readiness.
func (h *HistoryBootstrap) RequiredHistory() int {
	return h.maxLookback + h.safetyMargin
}

// RequiredForStructure returns the minimum bars needed for structure detection.
func (h *HistoryBootstrap) RequiredForStructure() int {
	return 30 // Minimum for swing detection with right-side confirmation
}

// RequiredForIchimoku returns bars needed for Ichimoku readiness.
func (h *HistoryBootstrap) RequiredForIchimoku() int {
	return 78 // senkouB(52) + displacement(26)
}

// RequiredForRollingStats returns bars needed for rolling Z-score readiness.
func (h *HistoryBootstrap) RequiredForRollingStats() int {
	return 20 // minSamples for rolling stats
}

// AddCandle records a new candle and returns the current readiness state.
func (h *HistoryBootstrap) AddCandle() {
	h.candleCount++
}

// CandleCount returns the current number of candles seen.
func (h *HistoryBootstrap) CandleCount() int {
	return h.candleCount
}

// IsReady returns true if all indicators have sufficient history.
func (h *HistoryBootstrap) IsReady() bool {
	return h.candleCount >= h.RequiredHistory()
}

// WarmupProgress returns a 0-1 progress value for warm-up completion.
func (h *HistoryBootstrap) WarmupProgress() float64 {
	if h.candleCount >= h.RequiredHistory() {
		return 1.0
	}
	return float64(h.candleCount) / float64(h.RequiredHistory())
}

// ReadinessState returns a human-readable readiness state string.
func (h *HistoryBootstrap) ReadinessState() string {
	switch {
	case h.candleCount < h.RequiredForStructure():
		return "INSUFFICIENT_HISTORY"
	case h.candleCount < h.RequiredForIchimoku():
		return "WARMING_UP"
	case h.candleCount < h.RequiredHistory():
		return "WARMING_UP"
	default:
		return "READY"
	}
}

// BackfillCandles generates synthetic historical candles from a starting price.
// This is used ONLY when historical MT5 data is not available.
// In production, real historical candles should come from the MT5 Master Node.
// SOW Section 3: "If upstream MT5 historical bars are available, integrate them through
// the existing Master Node/data architecture rather than adding an unrelated data path."
// This function is a fallback for when no historical data source is available.
// The generated candles are clearly labeled as DERIVED quality.
func BackfillCandles(symbol string, timeframe types.Timeframe, count int, endPrice float64, volatility float64, endTime time.Time) []*types.Candle {
	candles := make([]*types.Candle, count)
	price := endPrice

	// Work backwards from endPrice
	interval := timeframeDuration(timeframe)
	for i := count - 1; i >= 0; i-- {
		// Generate a realistic candle with some volatility
		change := (math.Sin(float64(i)*0.3) * volatility * price * 0.001) +
			((math.Cos(float64(i)*0.7) + 0.5) * volatility * price * 0.0005)

		open := price
		close := price - change
		high := math.Max(open, close) + math.Abs(change)*0.3
		low := math.Min(open, close) - math.Abs(change)*0.3

		candleTime := endTime.Add(-time.Duration(count-i) * interval)

		candles[i] = &types.Candle{
			Symbol:    symbol,
			Timeframe: timeframe,
			Time:      candleTime,
			Open:      decimal.NewFromFloat(open),
			High:      decimal.NewFromFloat(high),
			Low:       decimal.NewFromFloat(low),
			Close:     decimal.NewFromFloat(close),
			Volume:    int64(100 + i%50),
			IsClosed:  true,
			Source:    "HISTORICAL_BACKFILL",
			Quality:   types.CandleEstimated, // Clearly labeled as derived, not authoritative
		}

		price = close
	}

	return candles
}

// timeframeDuration converts a timeframe string to a time.Duration.
func timeframeDuration(tf types.Timeframe) time.Duration {
	switch tf {
	case "M1":
		return time.Minute
	case "M5":
		return 5 * time.Minute
	case "M15":
		return 15 * time.Minute
	case "M30":
		return 30 * time.Minute
	case "H1":
		return time.Hour
	case "H4":
		return 4 * time.Hour
	case "D1":
		return 24 * time.Hour
	default:
		return 15 * time.Minute // Default to M15
	}
}
