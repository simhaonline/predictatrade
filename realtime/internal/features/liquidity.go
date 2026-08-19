package features

import (
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// LiquidityEngine detects liquidity pools and sweeps.
// SOW Section 12A.3: Liquidity pools/sweeps.
// CRITICAL: CVD/DOM/order-flow are UNAVAILABLE for broker tick data.
type LiquidityEngine struct {
	pools     []LiquidityPool
	sweeps    []SweepEvent
	tolerance decimal.Decimal
}

func NewLiquidityEngine() *LiquidityEngine {
	return &LiquidityEngine{
		tolerance: decimal.NewFromFloat(0.10),
	}
}

func (e *LiquidityEngine) Process(candle *types.Candle, swings []decimal.Decimal) LiquidityFeatures {
	feat := LiquidityFeatures{
		CVDAvailable:    false, // SOW 6A: Never fabricate
		DOMAvailable:    false,
		OrderFlowQuality: types.QualityUnavailable,
	}

	if candle == nil {
		return feat
	}

	// Detect equal highs/lows as liquidity pools
	for _, s := range swings {
		for _, s2 := range swings {
			if s.Equal(s2) {
				continue
			}
			diff := s.Sub(s2).Abs()
			if diff.LessThanOrEqual(e.tolerance) {
				pool := LiquidityPool{
					Price:     s,
					Type:      "EQUAL_HIGHS",
					Strength:  3,
					CreatedAt: candle.Time,
				}
				e.pools = append(e.pools, pool)
			}
		}
	}

	// Detect sweeps: candle wick exceeds pool price but close is below
	for i, pool := range e.pools {
		if pool.Swept {
			continue
		}
		if candle.High.GreaterThanOrEqual(pool.Price) && candle.Close.LessThan(pool.Price) {
			e.pools[i].Swept = true
			e.sweeps = append(e.sweeps, SweepEvent{
				Price:     pool.Price,
				Direction: "SELL_SIDE_SWEEP",
				Time:      candle.Time,
			})
		}
		if candle.Low.LessThanOrEqual(pool.Price) && candle.Close.GreaterThan(pool.Price) {
			e.pools[i].Swept = true
			e.sweeps = append(e.sweeps, SweepEvent{
				Price:     pool.Price,
				Direction: "BUY_SIDE_SWEEP",
				Time:      candle.Time,
			})
		}
	}

	// Keep recent pools/sweeps
	if len(e.pools) > 20 {
		e.pools = e.pools[len(e.pools)-20:]
	}
	if len(e.sweeps) > 10 {
		e.sweeps = e.sweeps[len(e.sweeps)-10:]
	}

	feat.Pools = e.pools
	feat.RecentSweeps = e.sweeps
	return feat
}
