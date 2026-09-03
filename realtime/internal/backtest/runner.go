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
	config        BacktestConfig
	registry      *features.Registry
	strategy      strategy.Strategy
	tracker       *TradeTracker
	reader        *DBCandleReader
	cooldownUntil time.Time // production-parity: no new entries before this time
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
		reader:   NewDBCandleReader(config.DBUrl, config.Source),
	}
}

// Run executes the backtest over the historical data.
func (r *Runner) Run(ctx context.Context) (*BacktestResult, error) {
	startTime := time.Now()
	result := &BacktestResult{
		RunID:          r.config.RunID,
		Config:         r.config,
		Status:         "COMPLETED",
		NoTradeReasons: map[string]int{},
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

	// 2b. Expose the loaded history to multi-TF-history strategies (Arcanist)
	// without look-ahead. The store's "cur" is advanced each bar below.
	store := newCandleStore(r.config.PrimaryTimeframe, primaryCandles, higherCandles)
	strategy.SetArcanistCandleProvider(store.get)

	// 3. Main event loop — process each primary candle through the real engine
	var lastTick *types.Tick

	// Production-parity synthetic tick (v1.26): the live engine always has a
	// real tick (Bid≈Close−spread/2, Ask≈Close+spread/2, Spread≈$0.30). With
	// lastTick nil, features.Registry falls back to Bid=candle.Low /
	// Ask=candle.High / Spread=high−low — the FULL BAR RANGE — which makes
	// BUY entries print at the candle HIGH and vetoes geometry on any wide
	// bar (TP1 computed from Close can sit below the Ask entry). That
	// artifact (e.g. MARNIE_FIB "BUY_GEOMETRY_INVALID: TP1 <= Entry" walls)
	// does not exist live. Model a realistic tick instead: mid=Close,
	// spread=config.Spread (typical XAUUSD $0.30).
	syntheticSpread := r.config.Spread
	if syntheticSpread.IsZero() {
		syntheticSpread = decimal.NewFromFloat(0.30)
	}
	halfSynthSpread := syntheticSpread.Div(decimal.NewFromInt(2))
	lastTick = &types.Tick{
		Symbol:  r.config.Symbol,
		Quality: types.QualityAuthoritative,
	}

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

		// Refresh the synthetic tick for THIS bar (mid = close, realistic spread).
		lastTick.Mid = candle.Close
		lastTick.Bid = candle.Close.Sub(halfSynthSpread)
		lastTick.Ask = candle.Close.Add(halfSynthSpread)
		lastTick.Spread = syntheticSpread
		lastTick.SourceTimestamp = candle.Time
		lastTick.GatewayTimestamp = candle.Time

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
		store.cur = candle.Time
		stratResult := r.strategy.Evaluate(state)

		// Production-parity promotion (v1.26): mirror the live engine's
		// EXECUTABLE promotion rule (main.go: rawScore >= tradeThresh &&
		// all gates pass) before opening anything. Without this the runner
		// trades every directional read — measuring the raw direction engine,
		// not the product clients receive.
		if r.config.ProductionParity {
			reason, ok := r.parityGate(stratResult, candle, state)
			if !ok {
				result.ParityBlockedCount++
				result.NoTradeReasons[reason]++
				continue
			}
		}

		switch stratResult.Direction {
		case types.DirectionBuy:
			result.BuySignals++
			r.openTrade(stratResult, candle, i, types.DirectionBuy, state)
		case types.DirectionSell:
			result.SellSignals++
			r.openTrade(stratResult, candle, i, types.DirectionSell, state)
		case types.DirectionNoTrade:
			result.NoTradeCount++
			for _, rc := range stratResult.ReasonCodes {
				result.NoTradeReasons[string(rc)]++
			}
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

// parityGate mirrors the live engine's EXECUTABLE promotion rule before a
// trade is opened. Returns a NoTradeReason label when the read would NOT
// have been promoted to an executable signal live:
//  1. cooldown — the strategy is still in its post-signal cooldown window
//     (live: CooldownManager set on every confirmed signal);
//  2. trade bar — rawScore must reach the regime-specific TRADE threshold
//     (live: scoreDirectionWithThresholds candidate-band reads are advisory);
//  3. unique entry gate — the strategy's own entry gate must pass
//     (StrategyResult.EntryGatePassed, set by applyRefinement);
//  4. profitability — EV <= 0 loss candidates are vetoed fail-closed
//     (StrategyResult.IsLossCandidate; live: capital-protection downgrade).
func (r *Runner) parityGate(res strategy.StrategyResult, candle *types.Candle, state *features.MarketState) (string, bool) {
	// Only directional reads can be promoted at all.
	if res.Direction != types.DirectionBuy && res.Direction != types.DirectionSell {
		return "", true
	}

	if !r.cooldownUntil.IsZero() && candle.Time.Before(r.cooldownUntil) {
		return "PARITY_COOLDOWN", false
	}

	scoreF, _ := res.RawScore.Float64()
	_, tradeThresh, found := strategy.GetThresholds(r.config.StrategyID, state.Regime.Current)
	if found && scoreF < tradeThresh {
		return "PARITY_BELOW_TRADE_BAR", false
	}

	if !res.EntryGatePassed {
		// Sub-reason observability: tally the gate's own rejection codes so
		// entry-gate walls are diagnosable from the run summary alone.
		for _, rc := range res.ReasonCodes {
			if len(rc) > 0 {
				return string(rc), false
			}
		}
		return "PARITY_ENTRY_GATE", false
	}

	if res.IsLossCandidate {
		return "PARITY_NEGATIVE_EV", false
	}

	return "", true
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
		riskPerUnit = stratResult.EntryPrice.Sub(stratResult.StopLoss).Abs().Mul(r.config.ContractSize)
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

	// Honor the strategy's intended entry (e.g. an Arcanist POI limit level) when
	// provided; otherwise fall back to the signal-bar close. This keeps SL/TP
	// geometry consistent with the strategy's own plan.
	entryPx := candle.Close
	if !stratResult.EntryPrice.IsZero() {
		entryPx = stratResult.EntryPrice
	}
	pos := &OpenPosition{
		TradeID:     uuid.New().String()[:8],
		StrategyID:  r.config.StrategyID,
		Direction:   dir,
		EntryPrice:  entryPx,
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

	// Production parity: a confirmed signal starts the strategy's cooldown
	// window (live: CooldownManager.Set on every confirmed executable).
	if stratResult.CooldownMinutes > 0 {
		r.cooldownUntil = candle.Time.Add(time.Duration(stratResult.CooldownMinutes) * time.Minute)
	}

	r.tracker.OpenPosition(pos)
}

// ─── Higher timeframe alignment helpers ───

// candleStore exposes historical candles to strategies that need multi-TF history
// (e.g. Arcanist's BOS/order-block detection) during a backtest. It is filtered
// by an advancing "current time" so look-ahead bias is impossible: only candles
// whose time is <= the bar currently being evaluated are returned.
type candleStore struct {
	byTF map[types.Timeframe][]*types.Candle
	cur  time.Time
}

func newCandleStore(primaryTF types.Timeframe, primary []*types.Candle, higher map[types.Timeframe][]*types.Candle) *candleStore {
	byTF := make(map[types.Timeframe][]*types.Candle, len(higher)+1)
	byTF[primaryTF] = primary
	for tf, cs := range higher {
		byTF[tf] = cs
	}
	return &candleStore{byTF: byTF}
}

// get returns up to limit candles of tf with time <= cur, oldest→newest.
func (s *candleStore) get(_ string, tf types.Timeframe, limit int) ([]*types.Candle, error) {
	cs := s.byTF[tf]
	if len(cs) == 0 {
		return nil, nil
	}
	n := len(cs)
	for i := 0; i < len(cs); i++ {
		if cs[i].Time.After(s.cur) {
			n = i
			break
		}
	}
	start := n - limit
	if start < 0 {
		start = 0
	}
	if n <= start {
		return nil, nil
	}
	out := make([]*types.Candle, 0, n-start)
	out = append(out, cs[start:n]...)
	return out, nil
}

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
			// A higher-TF candle is only usable once it has FULLY closed:
			// its close time (start + tf duration) must be <= currentTime.
			// Using c.Time <= currentTime would leak in-progress bars (look-ahead bias).
			closeTime := c.Time.Add(tf.Duration())
			if closeTime.Before(currentTime) || closeTime.Equal(currentTime) {
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
