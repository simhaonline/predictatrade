package marketdata

import (
	"context"
	"sync"
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// Aggregator collects ticks and produces time-aligned candles.
// SOW Section 8, 150: UTC-aligned candle buckets.
type Aggregator struct {
	mu            sync.Mutex
	candles       map[string]map[types.Timeframe]*candleBuilder
	candleChan    chan *types.Candle
	supportedTFs  []types.Timeframe
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

func NewAggregator() *Aggregator {
	return &Aggregator{
		candles: make(map[string]map[types.Timeframe]*candleBuilder),
		candleChan: make(chan *types.Candle, 256),
		supportedTFs: []types.Timeframe{
			types.TFM1, types.TFM5, types.TFM15, types.TFM30,
			types.TFH1, types.TFH4, types.TFD1,
		},
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
		bucketStart := tick.GatewayTimestamp.Truncate(period)

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

// FlushClosedCandles checks for completed candles and emits them.
func (a *Aggregator) FlushClosedCandles(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			now := time.Now().UTC()
			for symbol, tfs := range a.candles {
				for tf, builder := range tfs {
					if !builder.updated {
						continue
					}
					nextBucket := builder.bucketStart.Add(builder.period)
					if now.After(nextBucket) {
						a.emitCandle(builder, true)
						a.candles[symbol][tf] = &candleBuilder{
							symbol:      symbol,
							timeframe:   tf,
							period:      builder.period,
							bucketStart: now.Truncate(builder.period),
							updated:     false,
						}
					}
				}
			}
			a.mu.Unlock()
		}
	}
}

func (a *Aggregator) emitCandle(b *candleBuilder, isClosed bool) {
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
		Quality:   types.CandleComplete,
		IsClosed:  isClosed,
		Alignment: types.AlignmentUTC,
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
				Alignment: types.AlignmentUTC,
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
