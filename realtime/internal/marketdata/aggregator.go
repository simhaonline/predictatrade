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
		supportedTFs: []types.Timeframe{
			types.TFM1, types.TFM5, types.TFM15, types.TFM30,
			types.TFH1, types.TFH4, types.TFD1,
		},
		offsetFunc: offsetFunc,
	}
}

func (a *Aggregator) CandleChannel() <-chan *types.Candle { return a.candleChan }

// ProcessTick ingests a tick and updates all timeframe candle builders.
func (a *Aggregator) ProcessTick(tick *types.Tick) {
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
				a.emitCandle(builder, true)
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
			off := a.offsetFunc()
			// "now" expressed in the broker session timezone.
			brokerNow := time.Now().UTC().Add(time.Duration(off) * time.Hour)
			for symbol, tfs := range a.candles {
				for tf, builder := range tfs {
					if !builder.updated {
						continue
					}
					// Candle completes when broker-local time passes the bucket end.
					bucketEndLocal := builder.bucketStart.Add(time.Duration(off) * time.Hour).Add(builder.period)
					if brokerNow.After(bucketEndLocal) {
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
