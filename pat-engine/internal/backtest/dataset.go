// Package backtest turns raw bars into MarketState snapshots (via the same indicator
// math the engine uses) and simulates trade outcomes so PF/win-rate are computed on
// the EXACT live strategy code and config — no separate research path that can drift.
package backtest

import (
	"math"
	"math/rand"
	"time"

	"pat-engine/internal/indicators"
	"pat-engine/internal/types"
)

// Bar is a single OHLC bar with a spread observation.
type Bar struct {
	Time   int64   `json:"time"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Spread float64 `json:"spread"`
}

// Generate produces a deterministic synthetic XAUUSD-like series with shifting
// regimes (trend / range / volatility). It exists so the harness can run with zero
// external data; real data is plugged in via FromCSV (same Bar shape).
func Generate(n int, seed int64) []Bar {
	r := rand.New(rand.NewSource(seed))
	bars := make([]Bar, n)
	price := 2000.0
	drift := 0.0
	vol := 0.8
	for i := 0; i < n; i++ {
		if i%400 == 0 { // regime shift
			switch r.Intn(3) {
			case 0:
				drift = 0.15 // uptrend
			case 1:
				drift = -0.15 // downtrend
			default:
				drift = 0.0 // range
			}
			vol = 0.5 + r.Float64()*1.2
		}
		o := price
		price = price + drift + (r.Float64()-0.5)*2*vol
		h := math.Max(o, price) + r.Float64()*vol
		l := math.Min(o, price) - r.Float64()*vol
		bars[i] = Bar{
			Time:   int64(i),
			Open:   o,
			High:   h,
			Low:    l,
			Close:  price,
			Spread: 0.20 + r.Float64()*0.25,
		}
	}
	return bars
}

// BuildSnapshots converts bars into MarketState snapshots using the indicator math,
// so the engine evaluates backtest and live identically.
func BuildSnapshots(bars []Bar) []*types.MarketState {
	n := len(bars)
	close := make([]float64, n)
	high := make([]float64, n)
	low := make([]float64, n)
	spread := make([]float64, n)
	for i, b := range bars {
		close[i] = b.Close
		high[i] = b.High
		low[i] = b.Low
		spread[i] = b.Spread
	}

	ema9 := indicators.EMA(close, 9)
	ema21 := indicators.EMA(close, 21)
	ema50 := indicators.EMA(close, 50)
	ema100 := indicators.EMA(close, 100)
	ema200 := indicators.EMA(close, 200)
	sma200 := indicators.SMA(close, 200)
	atr := indicators.ATR(high, low, close, 14)
	rsi := indicators.RSI(close, 14)
	macd, macdSig, osma := indicators.MACD(close, 12, 26, 9)
	adx, pDI, mDI := indicators.ADX(high, low, close, 14)
	stK, stD := indicators.Stochastic(high, low, close, 14, 3)
	_, bUp, bLo := indicators.Bollinger(close, 20, 2)
	vwap := indicators.VWAP(high, low, close, 20)

	states := make([]*types.MarketState, n)
	for i := range bars {
		if i < 200 {
			continue // warm-up
		}
		regime := "RANGE"
		switch {
		case adx[i] > 25 && ema9[i] > ema21[i]:
			regime = "TRENDING_BULLISH"
		case adx[i] > 25 && ema9[i] < ema21[i]:
			regime = "TRENDING_BEARISH"
		case adx[i] > 30:
			regime = "HIGH_VOLATILITY"
		}
		mtf := (ema9[i] - ema21[i]) / math.Max(atr[i], 0.0001) * 10

		st := &types.MarketState{
			Symbol:       "XAUUSD",
			Timeframe:    types.TFM1,
			CurrentPrice: close[i],
			Open:         bars[i].Open,
			High:         high[i],
			Low:          low[i],
			Close:        close[i],
			H1Close:      close[max(0, i-60)],
			Spread:       spread[i],
			ATR:          atr[i],
			Indicators: types.Indicators{
				EMA9: ema9[i], EMA21: ema21[i], EMA50: ema50[i],
				EMA100: ema100[i], EMA200: ema200[i], SMA200: sma200[i],
				ADX: adx[i], ADXPlusDI: pDI[i], ADXMinusDI: mDI[i],
				RSI: rsi[i],
				MACDMain: macd[i], MACDSignal: macdSig[i], OsMA: osma[i],
				StochMain: stK[i], StochSignal: stD[i],
				BollUpper: bUp[i], BollLower: bLo[i],
			},
			MTFScore: mtf,
			Regime:   regime,
			Session:  sessionFromTime(bars[i].Time),
			Quality:  "AUTHORITATIVE",
			VWAP:     vwap[i],
		}
		st.Candle.IsBullish = close[i] > bars[i].Open
		st.Candle.IsBearish = close[i] < bars[i].Open
		st.Candle.IsDisplacement = math.Abs(close[i]-bars[i].Open) > atr[i]*0.6

		// Derived market structure + liquidity (lightweight; enough for the
		// strategy confluence math that expects LastBOS / RecentSweeps).
		var bosDir string
		if i >= 20 {
			if close[i] > maxLastN(close, i, 20) {
				bosDir = "bullish"
			} else if close[i] < minLastN(close, i, 20) {
				bosDir = "bearish"
			}
		}
		if bosDir != "" {
			st.Structure.LastBOS = &types.BOS{Direction: bosDir}
		}
		if i >= 10 {
			if low[i] < minLastN(low, i, 10) {
				st.Liquidity.RecentSweeps = append(st.Liquidity.RecentSweeps, types.Sweep{Direction: "SELL_SIDE"})
			}
			if high[i] > maxLastN(high, i, 10) {
				st.Liquidity.RecentSweeps = append(st.Liquidity.RecentSweeps, types.Sweep{Direction: "BUY_SIDE"})
			}
		}

		states[i] = st
	}
	return states
}

// Simulate models the outcome of an executable signal: it walks forward bars and
// reports the realized P&L in price units (positive = profit).
func Simulate(states []*types.MarketState, i int, dir types.Direction, entry, sl, tp1 float64, maxBars int) float64 {
	if i+1 >= len(states) {
		return 0
	}
	for j := i + 1; j < len(states) && j <= i+maxBars; j++ {
		b := states[j]
		if dir == types.DirBuy {
			if b.Low <= sl {
				return sl - entry // loss
			}
			if b.High >= tp1 {
				return tp1 - entry // win
			}
		} else {
			if b.High >= sl {
				return entry - sl // loss
			}
			if b.Low <= tp1 {
				return entry - tp1 // win
			}
		}
	}
	// Expired without hit: mark to last close.
	last := states[min(len(states)-1, i+maxBars)].CurrentPrice
	if dir == types.DirBuy {
		return last - entry
	}
	return entry - last
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxLastN returns the max of arr in (i-n, i) exclusive of i.
func maxLastN(arr []float64, i, n int) float64 {
	m := arr[max(0, i-n)]
	for j := max(0, i-n); j < i; j++ {
		if arr[j] > m {
			m = arr[j]
		}
	}
	return m
}

// minLastN returns the min of arr in (i-n, i) exclusive of i.
func minLastN(arr []float64, i, n int) float64 {
	m := arr[max(0, i-n)]
	for j := max(0, i-n); j < i; j++ {
		if arr[j] < m {
			m = arr[j]
		}
	}
	return m
}

// sessionFromTime maps a bar timestamp to a gold trading session. The historical
// files are in broker/server time; this is an approximation used for the offline
// replay label. The live gateway uses broker.BrokerPolicy.Session (timezone-aware).
func sessionFromTime(t int64) types.Session {
	if t == 0 {
		return types.Session{CurrentSession: "LONDON"}
	}
	h := time.Unix(t, 0).UTC().Hour()
	switch {
	case h >= 0 && h < 7:
		return types.Session{CurrentSession: "TOKYO"}
	case h >= 7 && h < 13:
		return types.Session{CurrentSession: "LONDON"}
	case h >= 13 && h < 17:
		return types.Session{CurrentSession: "OVERLAP", IsOverlap: true}
	case h >= 17 && h < 22:
		return types.Session{CurrentSession: "NEW_YORK"}
	default:
		return types.Session{CurrentSession: "SYDNEY"}
	}
}
