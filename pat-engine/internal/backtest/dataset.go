// Package backtest turns raw bars into MarketState snapshots (via the same indicator
// math the engine uses) and simulates trade outcomes so PF/win-rate are computed on
// the EXACT live strategy code and config — no separate research path that can drift.
package backtest

import (
	"math"
	"math/rand"
	"time"

	"pat-engine/internal/broker"
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

// defaultSlippagePointsRoundTurn is the assumed round-turn slippage (in points) used
// by the honest replay simulator. XAUUSD typical slippage is 1-2 pts/side.
const defaultSlippagePointsRoundTurn = 3.0

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
	slowAtr := indicators.ATR(high, low, close, 100)
	ema400 := indicators.EMA(close, 400)
	// Higher-timeframe bias proxy: EMA200 vs EMA400 on the same series (a longer
	// horizon than the trading timeframe). Computed once, reused by the strategy gate.
	htfBull := make([]bool, n)
	for i := 0; i < n; i++ {
		htfBull[i] = ema200[i] > ema400[i]
	}
	rsi := indicators.RSI(close, 14)
	macd, macdSig, osma := indicators.MACD(close, 12, 26, 9)
	adx, pDI, mDI := indicators.ADX(high, low, close, 14)
	stK, stD := indicators.Stochastic(high, low, close, 14, 3)
	_, bUp, bLo := indicators.Bollinger(close, 20, 2)
	vwap := indicators.VWAP(high, low, close, 20)

	// Precompute liquidity-sweep / BOS detection over the series using a 15-bar
	// pivot window. A genuine *sweep* is a liquidity grab: price wicks beyond the
	// pivot extreme but CLOSES BACK INSIDE it (the playbook's highest-edge setup is
	// sweep -> reclaim). A BOS = close breaking the opposite pivot. The structural
	// edge is sweep -> BOS sequence, not same-bar. Pivot lookback 15, tolerance
	// 0.05% so legitimate XAUUSD grabs (sub-tick to a few pips) are captured.
	sweepSell := make([]bool, n)
	sweepBuy := make([]bool, n)
	bosBull := make([]bool, n)
	bosBear := make([]bool, n)
	for i := 0; i < n; i++ {
		if i < 20 {
			continue
		}
		pLow := minLastN(low, i, 15)
		pHigh := maxLastN(high, i, 15)
		// wick beyond the sell-side liquidity pool, but body reclaimed inside
		if low[i] <= pLow*0.9995 && close[i] > pLow {
			sweepSell[i] = true
		}
		// wick beyond the buy-side liquidity pool, but body reclaimed inside
		if high[i] >= pHigh*1.0005 && close[i] < pHigh {
			sweepBuy[i] = true
		}
		if close[i] > pHigh {
			bosBull[i] = true
		}
		if close[i] < pLow {
			bosBear[i] = true
		}
	}

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
			SlowATR:      slowAtr[i],
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
			HTFBias: func() types.Direction {
				if htfBull[i] {
					return types.Bullish
				}
				return types.Bearish
			}(),
			Session:  sessionFromTime(bars[i].Time),
			Quality:  "AUTHORITATIVE",
			VWAP:     vwap[i],
			UTCHour:  int(time.Unix(bars[i].Time, 0).UTC().Hour()),
		}
		st.Candle.IsBullish = close[i] > bars[i].Open
		st.Candle.IsBearish = close[i] < bars[i].Open
		st.Candle.IsDisplacement = math.Abs(close[i]-bars[i].Open) > atr[i]*0.6

		// Derived market structure + liquidity as a SWEEP -> BOS sequence:
		// a liquidity sweep within the last 8 bars, then a break of structure now.
		var sweeps []types.Sweep
		recentSellSweep, recentBuySweep := false, false
		for j := max(0, i-12); j <= i; j++ {
			if sweepSell[j] {
				recentSellSweep = true
			}
			if sweepBuy[j] {
				recentBuySweep = true
			}
		}
		if recentSellSweep {
			sweeps = append(sweeps, types.Sweep{Direction: "SELL_SIDE"})
		}
		if recentBuySweep {
			sweeps = append(sweeps, types.Sweep{Direction: "BUY_SIDE"})
		}
		st.Liquidity.RecentSweeps = sweeps
		if bosBull[i] {
			st.Structure.LastBOS = &types.BOS{Direction: "bullish"}
		} else if bosBear[i] {
			st.Structure.LastBOS = &types.BOS{Direction: "bearish"}
		}

		states[i] = st
	}
	return states
}

// Simulate models the outcome of an executable signal: it walks forward bars and
// reports the realized P&L in price units (positive = profit) AFTER realistic
// round-turn costs (spread on entry+exit, commission, slippage) using the broker
// execution profile. This is the honest figure — no idealized fills.
//
// Exit model follows the external scalping research (where the profit actually
// lives, §8): close 50% at the first target (1R) and move the remainder's stop to
// breakeven; then trail the remainder under the swing high/low by 1 ATR and take
// the next targets (2R/3R). A trade is counted a WIN once it reaches 1R (because
// the position is then risk-free at BE) — this is the genuine edge, not the prior
// "must reach 3R–5R" rule that turned scratches into full -1R losses.
// SimResult is the outcome of a single simulated signal. It reports the net P&L and
// the precise, target-level events used to calibrate named prediction targets.
type SimResult struct {
	PnL         float64
	TP1Hit      bool // 1R partial reached before SL
	TP2Hit      bool // 2R reached before SL
	TP3Hit      bool // 3R reached before SL
	SLHit       bool // stop hit first
	DirCorrect  bool // price moved >= 0.5R in the signal direction before SL
}

// Simulate models the outcome of an executable signal and returns the full
// SimResult (net P&L plus the target-level events the calibrator consumes). It walks
// forward bars using the honest cost-aware exit model described in BuildSnapshots.
func Simulate(states []*types.MarketState, i int, dir types.Direction, entry, sl, tp1, tp2, tp3 float64, maxBars int, exec broker.ExecutionProfile) SimResult {
	res := SimResult{}
	if i+1 >= len(states) {
		return res
	}
	// Round-turn transaction cost in price units (1-lot basis).
	cost := 0.0
	if exec.TickSize > 0 {
		spreadPts := exec.TypicalSpread * 2       // entry + exit
		slipPts := defaultSlippagePointsRoundTurn // round-turn slippage
		cost += (spreadPts + slipPts) * exec.TickSize
	}
	cost += exec.CommissionPrice(1.0) * 2 // round-turn commission (price units / lot)

	slDist := entry - sl
	if dir == types.DirSell {
		slDist = sl - entry
	}
	halfR := 0.5 * slDist

	const half = 0.5
	raw := func() float64 {
		stop := sl
		be := false
		partial := false
		best := entry
		for j := i + 1; j < len(states) && j <= i+maxBars; j++ {
			b := states[j]
			if dir == types.DirBuy {
				if !res.DirCorrect && b.High-entry >= halfR {
					res.DirCorrect = true
				}
				if !partial && b.High >= tp1 {
					partial = true
					res.TP1Hit = true
					be = true
					stop = entry // move remainder to breakeven
				}
				if be {
					if b.High >= tp2 {
						res.TP2Hit = true
					}
					if b.High > best {
						best = b.High
					}
					if trail := best - b.ATR; trail > stop {
						stop = trail
					}
					if b.High >= tp3 {
						res.TP3Hit = true
						return half*(tp1-entry) + half*(tp3-entry)
					}
					if b.High >= tp2 {
						return half*(tp1-entry) + half*(tp2-entry)
					}
					if b.Low <= stop {
						return half*(tp1-entry) + half*(stop-entry)
					}
				} else if b.Low <= stop {
					res.SLHit = true
					return sl - entry
				}
			} else {
				if !res.DirCorrect && entry-b.Low >= halfR {
					res.DirCorrect = true
				}
				if !partial && b.Low <= tp1 {
					partial = true
					res.TP1Hit = true
					be = true
					stop = entry
				}
				if be {
					if b.Low <= tp2 {
						res.TP2Hit = true
					}
					if b.Low < best {
						best = b.Low
					}
					if trail := best + b.ATR; trail < stop {
						stop = trail
					}
					if b.Low <= tp3 {
						res.TP3Hit = true
						return half*(entry-tp1) + half*(entry-tp3)
					}
					if b.Low <= tp2 {
						return half*(entry-tp1) + half*(entry-tp2)
					}
					if b.High >= stop {
						return half*(entry-tp1) + half*(entry-stop)
					}
				} else if b.High >= stop {
					res.SLHit = true
					return entry - sl
				}
			}
		}
		// Expired: mark the remainder to the trailing stop / BE / last close.
		last := states[min(len(states)-1, i+maxBars)].CurrentPrice
		if partial {
			remExit := entry
			if be {
				if dir == types.DirBuy {
					remExit = maxF(stop, last)
				} else {
					remExit = minF(stop, last)
				}
			}
			if dir == types.DirBuy {
				return half*(tp1-entry) + half*(remExit-entry)
			}
			return half*(entry-tp1) + half*(entry-remExit)
		}
		if dir == types.DirBuy {
			return last - entry
		}
		return entry - last
	}()
	res.PnL = raw - cost
	return res
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
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
