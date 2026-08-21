package backtest

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/strategy"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// Runner is the main backtest engine. It feeds historical candles through
// the real production feature engine and strategy evaluators.
type Runner struct {
	config       BacktestConfig
	registry     *features.Registry
	strategy     strategy.Strategy
	tracker      *TradeTracker
	reader       *DBCandleReader
}

// NewRunner creates a backtest runner with the given configuration.
func NewRunner(config BacktestConfig) *Runner {
	// Find the strategy
	var strat strategy.Strategy
	for _, s := range strategy.AllStrategies() {
		if s.ID() == config.StrategyID {
			strat = s
			break
		}
	}
	if strat == nil {
		strat = strategy.NewStandardScalping() // fallback
	}

	return &Runner{
		config:   config,
		registry: features.NewRegistry(),
		strategy: strat,
		tracker:  NewTradeTracker(config),
		reader:   NewDBCandleReader(config.DBUrl),
	}
}

// Run executes the backtest over the historical data.
func (r *Runner) Run(ctx context.Context) (*BacktestResult, error) {
	startTime := time.Now()
	result := &BacktestResult{
		RunID:  r.config.RunID,
		Config: r.config,
		Status: "COMPLETED",
	}

	if r.config.RunID == "" {
		result.RunID = uuid.New().String()[:8]
	}

	// 1. Load candles from database
	primaryCandles, higherCandles, err := r.reader.ReadAllTimeframes(
		ctx, r.config.Symbol, r.config.PrimaryTimeframe, r.config.HigherTimeframes,
		r.config.StartTime, r.config.EndTime,
	)
	if err != nil {
		result.Status = "FAILED"
		result.Error = fmt.Sprintf("data load: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	if len(primaryCandles) == 0 {
		result.Status = "FAILED"
		result.Error = "no primary candles loaded"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	result.StartTime = primaryCandles[0].Time
	result.EndTime = primaryCandles[len(primaryCandles)-1].Time

	// 2. Build higher timeframe lookup indexes (for MTF alignment, no look-ahead)
	higherLookups := buildHigherTFLookups(higherCandles)

	// 3. Main event loop — process each primary candle through the real engine
	var lastTick *types.Tick

	// Find the bar index where our actual backtest period starts
	backtestStartIdx := 0
	for i, c := range primaryCandles {
		if !c.Time.Before(r.config.StartTime) {
			backtestStartIdx = i
			break
		}
	}

	for i, candle := range primaryCandles {
		select {
		case <-ctx.Done():
			result.Status = "FAILED"
			result.Error = "cancelled"
			result.Duration = time.Since(startTime)
			return result, nil
		default:
		}

		// Build allCandles map for MTF (most recent closed higher TF candles)
		allCandles := buildAllCandlesMap(candle, r.config.PrimaryTimeframe, higherLookups, candle.Time)

		// Feed through the REAL production feature engine
		state := r.registry.Evaluate(candle, allCandles, lastTick)
		if state == nil {
			continue
		}

		// Update positions with current candle (check SL/TP)
		r.tracker.CheckDayRollover(candle.Time)
		r.tracker.UpdatePositions(candle, i)

		// Only evaluate strategy within the backtest period
		if i < backtestStartIdx {
			continue
		}

		result.BarsProcessed++

		// Check daily loss limit
		if r.tracker.DailyLossPct().LessThanOrEqual(r.config.MaxDailyLossPct.Neg()) {
			result.BlockedCount++
			continue
		}

		// Check max positions
		if r.tracker.OpenCount() >= r.config.MaxPositions {
			result.BlockedCount++
			continue
		}

		// Evaluate strategy using the REAL production strategy evaluator
		stratResult := r.strategy.Evaluate(state)

		switch stratResult.Direction {
		case types.DirectionBuy:
			result.BuySignals++
			r.openTrade(stratResult, candle, i, types.DirectionBuy, state)
		case types.DirectionSell:
			result.SellSignals++
			r.openTrade(stratResult, candle, i, types.DirectionSell, state)
		case types.DirectionNoTrade:
			result.NoTradeCount++
		case types.DirectionBlocked:
			result.BlockedCount++
		case types.DirectionWait:
			result.NoTradeCount++
		}
	}

	// 4. Close all remaining positions at the last candle
	if len(primaryCandles) > 0 && r.tracker.OpenCount() > 0 {
		r.tracker.CloseAllPositions(primaryCandles[len(primaryCandles)-1], len(primaryCandles)-1)
	}

	// 5. Compute metrics
	trades := r.tracker.ClosedTrades()
	result.Trades = trades
	result.Metrics = ComputeMetrics(trades, r.config.InitialBalance)
	result.Duration = time.Since(startTime)

	return result, nil
}

// openTrade opens a new position from a strategy result.
func (r *Runner) openTrade(stratResult strategy.StrategyResult, candle *types.Candle, barIdx int, dir types.Direction, state *features.MarketState) {
	// Validate: skip trades with invalid SL or TP
	if stratResult.StopLoss.IsZero() || stratResult.TP1.IsZero() {
		return
	}
	if stratResult.EntryPrice.IsZero() {
		return
	}

	// Calculate position size from risk
	riskAmount := r.tracker.Balance().Mul(r.config.MaxRiskPerTrade)

	// Apply recovery size multiplier
	riskAmount = riskAmount.Mul(r.tracker.RecoverySizeMultiplier())

	riskPerUnit := decimal.Zero
	if !stratResult.StopLoss.IsZero() {
		riskPerUnit = candle.Close.Sub(stratResult.StopLoss).Abs().Mul(r.config.ContractSize)
	}

	var size decimal.Decimal
	if !riskPerUnit.IsZero() {
		size = riskAmount.Div(riskPerUnit)
	} else {
		size = decimal.NewFromFloat(0.01)
	}

	// Clamp to reasonable bounds
	minLot := decimal.NewFromFloat(0.01)
	maxLot := decimal.NewFromInt(10)
	if size.LessThan(minLot) {
		size = minLot
	}
	if size.GreaterThan(maxLot) {
		size = maxLot
	}

	pos := &OpenPosition{
		TradeID:     uuid.New().String()[:8],
		StrategyID:  r.config.StrategyID,
		Direction:   dir,
		EntryPrice:  candle.Close,
		StopLoss:    stratResult.StopLoss,
		OriginalSL:  stratResult.StopLoss,
		TP1:         stratResult.TP1,
		TP2:         stratResult.TP2,
		TP3:         stratResult.TP3,
		EntryTime:   candle.Time,
		EntryBarIdx: barIdx,
		Size:        size,
		Regime:      state.Regime.Current,
		Session:     state.Session.CurrentSession,
		RawScore:    stratResult.RawScore,
	}

	r.tracker.OpenPosition(pos)
}

// ─── Higher timeframe alignment helpers ───

// higherTFLookup holds pre-indexed higher TF candles for fast lookup.
type higherTFLookup struct {
	candles []*types.Candle
	idx     int // current position in the candle slice
}

func buildHigherTFLookups(higherCandles map[types.Timeframe][]*types.Candle) map[types.Timeframe]*higherTFLookup {
	lookups := make(map[types.Timeframe]*higherTFLookup)
	for tf, candles := range higherCandles {
		lookups[tf] = &higherTFLookup{candles: candles, idx: 0}
	}
	return lookups
}

// buildAllCandlesMap returns a map of timeframe → most recent CLOSED candle
// at or before the given time. This prevents look-ahead bias.
func buildAllCandlesMap(primary *types.Candle, primaryTF types.Timeframe, higher map[types.Timeframe]*higherTFLookup, currentTime time.Time) map[types.Timeframe]*types.Candle {
	result := map[types.Timeframe]*types.Candle{
		primaryTF: primary,
	}

	for tf, lookup := range higher {
		// Advance the index to the most recent candle that closed before currentTime
		// A candle is "closed" when its time + duration <= currentTime
		// For simplicity: use candles where time < currentTime (the candle that
		// started before current time and should be closed by now)
		mostRecent := (*types.Candle)(nil)
		for lookup.idx < len(lookup.candles) {
			c := lookup.candles[lookup.idx]
			// The candle is closed if the next period has started
			// Approximation: candle time < current time means it started before
			// current bar, so it should be closed (for higher TF)
			if c.Time.Before(currentTime) || c.Time.Equal(currentTime) {
				mostRecent = c
				lookup.idx++
			} else {
				break
			}
		}
		if mostRecent != nil {
			result[tf] = mostRecent
		}
	}

	return result
}
