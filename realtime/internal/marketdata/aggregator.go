package marketdata

import (
	"context"
	"sync"
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// Aggregator collects ticks and produces time-aligned candles.
// SOW Section 8, 150: candle buckets are aligned to the BROKER session
// timezone (not UTC) using the offset reported by the Master Node, so D1/H4/
// W1/MN bars open at the broker's session boundaries — critical for correct
// indicator/strategy calculation.
type Aggregator struct {
	mu           sync.Mutex
	candles      map[string]map[types.Timeframe]*candleBuilder
	candleChan   chan *types.Candle
	supportedTFs []types.Timeframe
	// offsetFunc returns the broker UTC offset in hours (+3 = UTC+3). The
	// Master Node supplies this (auto-detected from live ticks or via config).
	offsetFunc func() int
	// externalCandles, when true, means broker CopyRates bars synced from the
	// Master Node are the authoritative candle source. The aggregator then stops
	// emitting tick-built candles so engine candles match MT5 exactly.
	externalCandles bool
	// lastExternal tracks the most recent external (CopyRates) candle time per
	// symbol/timeframe so the aggregator can detect when the Master Node bar
	// stream goes stale and fall back to building fresh candles from the live
	// tick stream. Fixes stale-market on the dashboard when the data agent's
	// CopyRates feed lags (e.g. throttled PENDING-license EA).
	lastExternalMu sync.Mutex
	lastExternal   map[string]map[types.Timeframe]time.Time
	// externalStaleAfter is how long since the last fresh external candle before
	// we treat that timeframe's bar feed as stale and fall back to ticks.
	externalStaleAfter time.Duration
}

// TimeframeDuration returns the duration of a timeframe. Exported so callers
// (e.g. broker bar sync) can compute previous-bar boundaries.
func TimeframeDuration(tf types.Timeframe) time.Duration {
	return timeframeDuration(tf)
}

type candleBuilder struct {
	symbol    string
	timeframe types.Timeframe
	period    time.Duration
	open      decimal.Decimal
	high      decimal.Decimal
	low       decimal.Decimal
	close     decimal.Decimal
	volume    int64
	bucketStart time.Time
	updated   bool
}

// NewAggregator creates a candle aggregator. offsetFunc returns the broker UTC
// offset in hours so buckets align to broker session boundaries.
func NewAggregator(offsetFunc func() int) *Aggregator {
	if offsetFunc == nil {
		offsetFunc = func() int { return 0 }
	}
	return &Aggregator{
		candles: make(map[string]map[types.Timeframe]*candleBuilder),
		candleChan: make(chan *types.Candle, 256),
		lastExternal: make(map[string]map[types.Timeframe]time.Time),
		externalStaleAfter: 90 * time.Second,
		supportedTFs: []types.Timeframe{
			types.TFM1, types.TFM5, types.TFM15, types.TFM30,
			types.TFH1, types.TFH4, types.TFD1,
		},
		offsetFunc: offsetFunc,
	}
}

func (a *Aggregator) CandleChannel() <-chan *types.Candle { return a.candleChan }

// UseExternalCandles switches the aggregator to external (broker CopyRates)
// candle mode: tick-built candle emission is suppressed so the engine uses the
// Master Node's per-TF broker bars verbatim (exact MT5 match). Idempotent.
func (a *Aggregator) UseExternalCandles() {
	a.mu.Lock()
	a.externalCandles = true
	a.mu.Unlock()
}

// PushExternalCandle injects an externally-sourced (broker CopyRates) candle
// into the candle pipeline so it flows to persistence, WebSocket and strategy
// evaluation exactly as received from MT5.
func (a *Aggregator) PushExternalCandle(c *types.Candle) {
	a.lastExternalMu.Lock()
	if a.lastExternal[c.Symbol] == nil {
		a.lastExternal[c.Symbol] = make(map[types.Timeframe]time.Time)
	}
	a.lastExternal[c.Symbol][c.Timeframe] = c.Time
	a.lastExternalMu.Unlock()
	select {
	case a.candleChan <- c:
	default:
	}
}

// externalStale reports whether the external (CopyRates) candle feed for a
// symbol/timeframe is stale (no fresh bar within externalStaleAfter), or has
// never been received. When true, callers should fall back to tick-built
// candles so the engine never serves stale market data.
func (a *Aggregator) externalStale(symbol string, tf types.Timeframe) bool {
	a.lastExternalMu.Lock()
	defer a.lastExternalMu.Unlock()
	tfs, ok := a.lastExternal[symbol]
	if !ok {
		return true
	}
	lt, ok := tfs[tf]
	if !ok {
		return true
	}
	return time.Since(lt) > a.externalStaleAfter
}

// ProcessTick ingests a tick and updates all timeframe candle builders.
func (a *Aggregator) ProcessTick(tick *types.Tick) {
	// When broker CopyRates sync is active, the Master Node supplies authoritative
	// candles. We still aggregate live ticks so we always hold a FRESH fallback
	// candle, but only EMIT tick-built candles when the external bar feed is stale
	// (see externalStale) — otherwise the authoritative external candle wins.
	a.mu.Lock()
	defer a.mu.Unlock()

	symbol := tick.Symbol
	if _, ok := a.candles[symbol]; !ok {
		a.candles[symbol] = make(map[types.Timeframe]*candleBuilder)
	}

	for _, tf := range a.supportedTFs {
		period := timeframeDuration(tf)
		// Align the bucket to the BROKER session boundary, not UTC. The tick's
		// SourceTimestamp is the true UTC instant; shift into broker-local time,
		// truncate to the period, then shift back to UTC. This makes D1/H4/W1/MN
		// candles open at the broker's session start (e.g. NY-close D1), matching
		// what the trader sees on the MT5 chart — eliminating indicator misalignment.
		off := a.offsetFunc()
		brokerLocal := tick.SourceTimestamp.Add(time.Duration(off) * time.Hour)
		bucketStart := brokerLocal.Truncate(period).Add(-time.Duration(off) * time.Hour)

		builder, exists := a.candles[symbol][tf]
		if !exists || !builder.bucketStart.Equal(bucketStart) {
			// Previous candle completed — emit it
			if exists && builder.updated {
				// In external mode, only emit the tick-built closed candle if the
				// Master Node bar feed is stale; otherwise the external candle wins.
				if !a.externalCandles || a.externalStale(symbol, tf) {
					a.emitCandle(builder, true)
				}
			}
			// Start new candle
			a.candles[symbol][tf] = &candleBuilder{
				symbol:      symbol,
				timeframe:   tf,
				period:      period,
				open:        tick.Mid,
				high:        tick.Ask,
				low:         tick.Bid,
				close:       tick.Mid,
				volume:      tick.TickVolume,
				bucketStart: bucketStart,
				updated:     true,
			}
	} else {
		// Update existing candle
		if !builder.updated {
			// Builder is an uninitialized placeholder left by FlushClosedCandles.
			// Fully initialize from this tick so zero-values never leak.
			// Fixes audit 04_DATABASE: 553 corrupted candles with open=0/low=0.
			builder.open = tick.Mid
			builder.high = tick.Ask
			builder.low = tick.Bid
			builder.close = tick.Mid
			builder.volume = tick.TickVolume
			builder.updated = true
		} else {
			if tick.Ask.GreaterThan(builder.high) {
				builder.high = tick.Ask
			}
			if tick.Bid.LessThan(builder.low) {
				builder.low = tick.Bid
			}
			builder.close = tick.Mid
			builder.volume += tick.TickVolume
			builder.updated = true
		}
	}
	}
}

// FlushClosedCandles checks for completed candles and emits them. Completion is
// evaluated in BROKER session time (not UTC) so candles close at the broker's
// bar boundary — critical for correct H4/D1/W1/MN alignment.
func (a *Aggregator) FlushClosedCandles(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			stale := false
			// External (broker CopyRates) mode: the Master Node is authoritative
			// for candle completion. Skip tick-based candles UNLESS the bar feed is
			// stale — then fall back to fresh tick-built candles (see loop body).
			if a.externalCandles {
				// Determine staleness once for this iteration.
				stale = false
				for sym, tfs := range a.candles {
					for tf := range tfs {
						if a.externalStale(sym, tf) {
							stale = true
						}
						break
					}
					if stale {
						break
					}
				}
				if !stale {
					a.mu.Unlock()
					continue
				}
			}
			off := a.offsetFunc()
			// "now" expressed in the broker session timezone.
			brokerNow := time.Now().UTC().Add(time.Duration(off) * time.Hour)
			for symbol, tfs := range a.candles {
				for tf, builder := range tfs {
				if !builder.updated {
					continue
				}
				// Fallback (external stale): surface the live forming candle so
				// dashboards/charts show current price action, not a stale bar.
				if a.externalCandles && a.externalStale(symbol, tf) {
					a.emitCandle(builder, false)
				}
				// Candle completes when broker-local time passes the bucket end.
				bucketEndLocal := builder.bucketStart.Add(time.Duration(off) * time.Hour).Add(builder.period)
				// Only emit tick-built closed candles when NOT in external mode, or when
				// the external bar feed is stale (external handles completion when fresh).
				if (!a.externalCandles || a.externalStale(symbol, tf)) && brokerNow.After(bucketEndLocal) {
					a.emitCandle(builder, true)
						newLocalStart := brokerNow.Truncate(builder.period)
						a.candles[symbol][tf] = &candleBuilder{
							symbol:      symbol,
							timeframe:   tf,
							period:      builder.period,
							open:        builder.close, // Carry forward last close as seed
							high:        builder.close,
							low:         builder.close,
							close:       builder.close,
							bucketStart: newLocalStart.Add(-time.Duration(off) * time.Hour),
							updated:     false,
						}
					}
				}
			}
			a.mu.Unlock()
		}
	}
}

// alignmentForOffset returns the candle alignment profile based on the active
// broker offset. When the offset is non-zero (broker session time collected
// from the Master Node) candles are BROKER_ALIGNED; otherwise UTC_ALIGNED.
func (a *Aggregator) alignmentForOffset() types.AlignmentProfile {
	if a.offsetFunc() != 0 {
		return types.AlignmentBroker
	}
	return types.AlignmentUTC
}

func (a *Aggregator) emitCandle(b *candleBuilder, isClosed bool) {
	// Validate candle integrity before emitting.
	// Fixes audit 04_DATABASE: prevents zero-valued OHLC candles
	// from being broadcast with quality=COMPLETE (553 rows Aug 18-21).
	quality := types.CandleComplete
	if b.open.IsZero() || b.high.IsZero() || b.low.IsZero() || b.close.IsZero() {
		quality = types.CandleInvalid
	} else if b.high.LessThan(b.low) {
		quality = types.CandleInvalid
	}
	candle := &types.Candle{
		Symbol:    b.symbol,
		Timeframe: b.timeframe,
		Time:      b.bucketStart,
		Open:      b.open,
		High:      b.high,
		Low:       b.low,
		Close:     b.close,
		Volume:    b.volume,
		Source:    "AGGREGATOR",
		Quality:   quality,
		IsClosed:  isClosed,
		Alignment: a.alignmentForOffset(),
	}
	if !isClosed {
		candle.Quality = types.CandlePartial
	}
	select {
	case a.candleChan <- candle:
	default:
	}
}

// GetCurrentCandle returns the current (in-progress) candle for a symbol/timeframe.
func (a *Aggregator) GetCurrentCandle(symbol string, tf types.Timeframe) *types.Candle {
	a.mu.Lock()
	defer a.mu.Unlock()
	if tfs, ok := a.candles[symbol]; ok {
		if b, ok := tfs[tf]; ok && b.updated {
			return &types.Candle{
				Symbol:    b.symbol,
				Timeframe: b.timeframe,
				Time:      b.bucketStart,
				Open:      b.open,
				High:      b.high,
				Low:       b.low,
				Close:     b.close,
				Volume:    b.volume,
				Source:    "AGGREGATOR",
				Quality:   types.CandlePartial,
				IsClosed:  false,
				Alignment: a.alignmentForOffset(),
			}
		}
	}
	return nil
}

// GetRecentCandles returns accumulated candles for a symbol/timeframe (limited history in memory).
func (a *Aggregator) GetCandleHistory(symbol string, tf types.Timeframe, count int) []*types.Candle {
	// In a full implementation, this would query the database.
	// For now, return the current candle only.
	c := a.GetCurrentCandle(symbol, tf)
	if c == nil {
		return nil
	}
	return []*types.Candle{c}
}

// GetCurrentCandles returns a snapshot of all in-progress (forming) candles the
// aggregator is currently building from the live tick stream for a symbol. This
// is the freshest possible market view and is used to keep the engine state —
// and therefore the dashboard — current even when the Master Node CopyRates
// (external) feed is stale or the candle pipeline channel is congested.
func (a *Aggregator) GetCurrentCandles(symbol string) map[types.Timeframe]*types.Candle {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make(map[types.Timeframe]*types.Candle)
	if tfs, ok := a.candles[symbol]; ok {
		for tf, b := range tfs {
			if !b.updated {
				continue
			}
			out[tf] = &types.Candle{
				Symbol:    b.symbol,
				Timeframe: b.timeframe,
				Time:      b.bucketStart,
				Open:      b.open,
				High:      b.high,
				Low:       b.low,
				Close:     b.close,
				Volume:    b.volume,
				Source:    "AGGREGATOR",
				Quality:   types.CandlePartial,
				IsClosed:  false,
				Alignment: a.alignmentForOffset(),
			}
		}
	}
	return out
}

// CurrentBucketStart returns the broker-aligned start time of the CURRENT (live)
// candle bucket for a timeframe, computed from the gateway's real UTC clock —
// NOT from the (possibly lagging) Master Node reported timestamp. This keeps the
// live forming candle anchored to real time even when agent ticks/snapshots
// carry a stale SourceTimestamp, so the dashboard never shows a stale bucket.
func (a *Aggregator) CurrentBucketStart(tf types.Timeframe) time.Time {
	off := a.offsetFunc()
	period := timeframeDuration(tf)
	now := time.Now().UTC()
	brokerLocal := now.Add(time.Duration(off) * time.Hour)
	return brokerLocal.Truncate(period).Add(-time.Duration(off) * time.Hour)
}

func timeframeDuration(tf types.Timeframe) time.Duration {
	switch tf {
	case types.TFM1:
		return time.Minute
	case types.TFM5:
		return 5 * time.Minute
	case types.TFM15:
		return 15 * time.Minute
	case types.TFM30:
		return 30 * time.Minute
	case types.TFH1:
		return time.Hour
	case types.TFH4:
		return 4 * time.Hour
	case types.TFD1:
		return 24 * time.Hour
	case types.TFW1:
		return 7 * 24 * time.Hour
	case types.TFMN1:
		return 30 * 24 * time.Hour
	default:
		return time.Minute
	}
}
