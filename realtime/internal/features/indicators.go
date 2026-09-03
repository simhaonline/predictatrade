package features

import (
	"math"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// IndicatorEngine computes ALL locally-calculable technical indicators.
// Mathematical formulas follow canonical definitions (Wilder RSI, Wilder ATR, etc.).
// Numerical safety: all calculations handle NaN, infinity, division by zero,
// insufficient history, and invalid candle data explicitly.
// Indicators not available (SAR, Ichimoku, StochRSI, Volume Profile, Cumulative Delta)
// are UNAVAILABLE and never fabricated.
//
// PERFORMANCE (2026-09-03, pprof-backed):
// The original implementation recomputed every indicator from a 200-element
// decimal.Decimal window each bar. decimal→float64 conversion (big.Rat +
// Lehmer GCD + big division inside shopspring Decimal.Float64) dominated CPU
// (~54% of total samples; EMA alone 22.9% via 5×200 Float64 calls per bar),
// giving ~3.5ms/bar — a 6.7-year M1 backtest would take ~2 hours.
// The engine now keeps PARALLEL float64 windows: each new candle is converted
// ONCE per field, and all indicator math runs in pure float64 with the SAME
// algorithms (identical warmup/seeding rules), converting only the final
// feature values back to decimal. Values match the previous implementation
// within float64 rounding, which the math package already standardized on.
type IndicatorEngine struct {
	closes     []decimal.Decimal
	highs      []decimal.Decimal
	lows       []decimal.Decimal
	volumes    []int64
	lookback   int
	prevEMA9   decimal.Decimal
	prevEMA21  decimal.Decimal
	prevMACD       decimal.Decimal
	prevMACDSignal decimal.Decimal
	prevRSI        decimal.Decimal
	prevClose      decimal.Decimal
	prevBollLower  decimal.Decimal
	prevBollUpper  decimal.Decimal
	macdHist       []decimal.Decimal
	stochHist  []decimal.Decimal // %K history for Stochastic signal line (3-period SMA)

	// float64 window mirrors — each appended value is converted exactly once
	fCloses  []float64
	fHighs   []float64
	fLows    []float64
	fVolumes []float64
	// running OBV over the float window (recomputed per bar over the window,
	// matching the original semantics, but in cheap float math)
}

func NewIndicatorEngine(lookback int) *IndicatorEngine {
	return &IndicatorEngine{lookback: lookback}
}

func (e *IndicatorEngine) Process(candle *types.Candle) IndicatorFeatures {
	feat := IndicatorFeatures{}
	if candle == nil {
		return feat
	}

	// Numerical safety: validate candle data
	if candle.High.LessThan(candle.Low) || candle.High.LessThan(candle.Open) ||
		candle.High.LessThan(candle.Close) || candle.Low.GreaterThan(candle.Open) ||
		candle.Low.GreaterThan(candle.Close) {
		return feat // Invalid candle — return zeros (UNAVAILABLE, not neutral)
	}

	e.closes = append(e.closes, candle.Close)
	e.highs = append(e.highs, candle.High)
	e.lows = append(e.lows, candle.Low)
	e.volumes = append(e.volumes, candle.Volume)

	if len(e.closes) > e.lookback {
		e.closes = e.closes[len(e.closes)-e.lookback:]
	}
	if len(e.highs) > e.lookback {
		e.highs = e.highs[len(e.highs)-e.lookback:]
	}
	if len(e.lows) > e.lookback {
		e.lows = e.lows[len(e.lows)-e.lookback:]
	}
	if len(e.volumes) > e.lookback {
		e.volumes = e.volumes[len(e.volumes)-e.lookback:]
	}

	// Maintain float64 mirrors (one conversion per field per bar)
	ch, _ := candle.High.Float64()
	cl, _ := candle.Low.Float64()
	cc, _ := candle.Close.Float64()
	e.fCloses = append(e.fCloses, cc)
	e.fHighs = append(e.fHighs, ch)
	e.fLows = append(e.fLows, cl)
	e.fVolumes = append(e.fVolumes, float64(candle.Volume))
	if len(e.fCloses) > e.lookback {
		e.fCloses = e.fCloses[len(e.fCloses)-e.lookback:]
	}
	if len(e.fHighs) > e.lookback {
		e.fHighs = e.fHighs[len(e.fHighs)-e.lookback:]
	}
	if len(e.fLows) > e.lookback {
		e.fLows = e.fLows[len(e.fLows)-e.lookback:]
	}
	if len(e.fVolumes) > e.lookback {
		e.fVolumes = e.fVolumes[len(e.fVolumes)-e.lookback:]
	}

	nC := len(e.fCloses)

	// === Trend Indicators ===

	// EMA 9/21/50/100/200 — canonical EMA over the window (identical to
	// pkg/math.EMA: seed with first value, recursive float smoothing).
	if nC >= 9 {
		feat.EMA9 = fdec(fEMAWindow(e.fCloses, 9))
	}
	if nC >= 21 {
		feat.EMA21 = fdec(fEMAWindow(e.fCloses, 21))
	}
	if nC >= 50 {
		feat.EMA50 = fdec(fEMAWindow(e.fCloses, 50))
	}
	if nC >= 100 {
		feat.EMA100 = fdec(fEMAWindow(e.fCloses, 100))
	}
	if nC >= 200 {
		feat.EMA200 = fdec(fEMAWindow(e.fCloses, 200))
	}

	// EMA Cross 9/21 — detect actual crossover event, not just alignment
	if !e.prevEMA9.IsZero() && !e.prevEMA21.IsZero() && !feat.EMA9.IsZero() && !feat.EMA21.IsZero() {
		if e.prevEMA9.LessThanOrEqual(e.prevEMA21) && feat.EMA9.GreaterThan(feat.EMA21) {
			feat.EMACross921 = true // Bullish cross event
		}
	}
	// Alignment (separate from cross event) — stored in the comparison itself

	// SMA 50/100/200 (cheap running sums in float64)
	if nC >= 50 {
		feat.SMA50 = fdec(fSMA(e.fCloses, 50))
	}
	if nC >= 100 {
		feat.SMA100 = fdec(fSMA(e.fCloses, 100))
	}
	if nC >= 200 {
		feat.SMA200 = fdec(fSMA(e.fCloses, 200))
	}

	// MACD (12, 26, 9)
	if nC >= 26 {
		ema12 := fdec(fEMAWindow(e.fCloses, 12))
		ema26 := fdec(fEMAWindow(e.fCloses, 26))
		feat.MACDMain = ema12.Sub(ema26)

		// Proper MACD signal: EMA9 of MACD line history
		e.macdHist = append(e.macdHist, feat.MACDMain)
		if len(e.macdHist) > 20 {
			e.macdHist = e.macdHist[len(e.macdHist)-20:]
		}
		if len(e.macdHist) >= 9 {
			feat.MACDSignal = fdec(fEMAOverDecimals(e.macdHist, 9))
		}
	}

	// ADX 14 with +DI / -DI using full Wilder's method (prompt.md Section 1.5)
	if len(e.fHighs) >= 28 {
		adx, plusDI, minusDI := fADXWilder(e.fHighs, e.fLows, e.fCloses, 14)
		feat.ADX = fdec(adx)
		feat.ADXPlusDI = fdec(plusDI)
		feat.ADXMinusDI = fdec(minusDI)
	}

	// === Momentum Indicators ===

	// OsMA — MACD_Main - MACD_Signal
	if !feat.MACDMain.IsZero() && !feat.MACDSignal.IsZero() {
		feat.OsMA = feat.MACDMain.Sub(feat.MACDSignal)
	}

	// RSI 14 (Wilder's method)
	if nC >= 14 {
		feat.RSI = fdec(fRSIWilder(e.fCloses, 14))
	}

	// Stochastic 14/3/3
	if nC >= 14 && len(e.fHighs) >= 14 && len(e.fLows) >= 14 {
		highestHigh := fMax(e.fHighs[nC-14:])
		lowestLow := fMin(e.fLows[nC-14:])
		rangeVal := highestHigh - lowestLow
		if rangeVal > 0 {
			feat.StochMain = fdec(100.0 * (cc - lowestLow) / rangeVal)
			// Stoch Signal = 3-period SMA of %K
			e.stochHist = append(e.stochHist, feat.StochMain)
			if len(e.stochHist) > 10 {
				e.stochHist = e.stochHist[len(e.stochHist)-10:]
			}
			if len(e.stochHist) >= 3 {
				feat.StochSignal = fdec(fSMAOverDecimals(e.stochHist, 3))
			} else if len(e.stochHist) > 0 {
				feat.StochSignal = fdec(fSMAOverDecimals(e.stochHist, len(e.stochHist)))
			}
		}
	}

	// Stochastic RSI — UNAVAILABLE (requires RSI history)

	// CCI 20
	if nC >= 20 && len(e.fHighs) >= 20 && len(e.fLows) >= 20 {
		tp := make([]float64, 20)
		for i := 0; i < 20; i++ {
			tp[i] = (e.fHighs[nC-20+i] + e.fLows[nC-20+i] + e.fCloses[nC-20+i]) / 3.0
		}
		smaTP := fSMA(tp, 20)
		meanDev := fMeanDeviation(tp, smaTP)
		if meanDev > 0 {
			feat.CCI = fdec((tp[19] - smaTP) / (0.015 * meanDev))
		}
	}

	// === Volatility Indicators ===

	// ATR 14 (Wilder's method) — skip zero high/low candles (MT5 data gaps)
	if nC >= 14 {
		feat.ATR = fdec(fATRWilder(e.fHighs, e.fLows, e.fCloses, 14))
	}

	// Bollinger Bands 20/2
	if nC >= 20 {
		sma := fSMA(e.fCloses, 20)
		stdDev := fStdDev(e.fCloses[nC-20:], sma)
		feat.BollMiddle = fdec(sma)
		feat.BollUpper = fdec(sma + 2.0*stdDev)
		feat.BollLower = fdec(sma - 2.0*stdDev)
		if sma > 0 {
			feat.BollWidth = fdec((4.0 * stdDev) / sma)
		}
	}

	// === Volume Indicators ===

	// OBV — over the float window (identical semantics to the decimal loop)
	if nC >= 2 {
		obv := 0.0
		for i := 1; i < nC; i++ {
			if e.fCloses[i] > e.fCloses[i-1] {
				obv += e.fVolumes[i]
			} else if e.fCloses[i] < e.fCloses[i-1] {
				obv -= e.fVolumes[i]
			}
		}
		feat.OBV = fdec(obv)
	}

	// Volume Profile — UNAVAILABLE (requires real volume, broker provides tick volume only)
	// Cumulative Delta — UNAVAILABLE (requires centralized order-flow)
	// VWAP — set from MT5 snapshot (session-anchored)

	// Store previous values for cross detection
	e.prevEMA9 = feat.EMA9
	e.prevEMA21 = feat.EMA21
	e.prevMACD = feat.MACDMain
	e.prevMACDSignal = feat.MACDSignal
	e.prevRSI = feat.RSI
	e.prevClose = candle.Close
	e.prevBollLower = feat.BollLower
	e.prevBollUpper = feat.BollUpper

	return feat
}

// fdec converts a float64 result to decimal (the ONLY conversion per output).
func fdec(v float64) decimal.Decimal {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return decimal.Zero
	}
	return decimal.NewFromFloat(v)
}

// fEMAWindow computes EMA over a float window with the canonical seeding
// used by pkg/math.EMA (seed = first value, alpha = 2/(period+1)).
func fEMAWindow(values []float64, period int) float64 {
	if len(values) == 0 || period <= 0 {
		return 0
	}
	multiplier := 2.0 / float64(period+1)
	ema := values[0]
	for i := 1; i < len(values); i++ {
		ema = values[i]*multiplier + ema*(1-multiplier)
	}
	return ema
}

// fEMAOverDecimals: EMA over a small decimal history (MACD signal line).
func fEMAOverDecimals(values []decimal.Decimal, period int) float64 {
	if len(values) < period || period <= 0 {
		return 0
	}
	vals := make([]float64, len(values))
	for i, v := range values {
		vals[i], _ = v.Float64()
	}
	return fEMA(vals, period)
}

// fEMA computes EMA with SMA seeding (canonical definition used by calcEMA).
func fEMA(values []float64, period int) float64 {
	if len(values) < period || period <= 0 {
		return 0
	}
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += values[i]
	}
	ema := sum / float64(period)
	alpha := 2.0 / float64(period+1)
	for i := period; i < len(values); i++ {
		ema = alpha*values[i] + (1-alpha)*ema
	}
	return ema
}

func fSMA(values []float64, period int) float64 {
	if len(values) < period || period <= 0 {
		return 0
	}
	sum := 0.0
	for i := len(values) - period; i < len(values); i++ {
		sum += values[i]
	}
	return sum / float64(period)
}

func fSMAOverDecimals(values []decimal.Decimal, period int) float64 {
	if len(values) < period || period <= 0 {
		return 0
	}
	sum := 0.0
	for i := len(values) - period; i < len(values); i++ {
		f, _ := values[i].Float64()
		sum += f
	}
	return sum / float64(period)
}

func fStdDev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sumSq := 0.0
	for _, v := range values {
		d := v - mean
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(values)))
}

func fMax(values []float64) float64 {
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func fMin(values []float64) float64 {
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func fMeanDeviation(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Abs(v - mean)
	}
	return sum / float64(len(values))
}

// fRSIWilder — Wilder RSI, identical algorithm to pkg/math.RSIWilder.
func fRSIWilder(closes []float64, period int) float64 {
	n := len(closes)
	if n <= period || period <= 0 {
		return 0
	}
	changes := make([]float64, n-1)
	for i := 1; i < n; i++ {
		changes[i-1] = closes[i] - closes[i-1]
	}
	var avgGain, avgLoss float64
	for i := 0; i < period; i++ {
		if changes[i] > 0 {
			avgGain += changes[i]
		} else {
			avgLoss += -changes[i]
		}
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)
	for i := period; i < len(changes); i++ {
		gain, loss := 0.0, 0.0
		if changes[i] > 0 {
			gain = changes[i]
		} else {
			loss = -changes[i]
		}
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
	}
	if avgLoss == 0 && avgGain == 0 {
		return 50
	}
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100.0 - 100.0/(1.0+rs)
}

// fATRWilder — Wilder ATR with zero-high/low skip, identical to pkg/math.ATRWilder.
func fATRWilder(highs, lows, closes []float64, period int) float64 {
	n := len(highs)
	if len(lows) < n {
		n = len(lows)
	}
	if len(closes) < n {
		n = len(closes)
	}
	if n <= period || period <= 0 {
		return 0
	}
	trs := make([]float64, 0, n-1)
	for i := 1; i < n; i++ {
		h, l, pc := highs[i], lows[i], closes[i-1]
		if h <= 0 || l <= 0 {
			continue
		}
		hl := math.Abs(h - l)
		hc := math.Abs(h - pc)
		lc := math.Abs(l - pc)
		tr := hl
		if hc > tr {
			tr = hc
		}
		if lc > tr {
			tr = lc
		}
		trs = append(trs, tr)
	}
	if len(trs) < period {
		return 0
	}
	atr := 0.0
	for i := 0; i < period; i++ {
		atr += trs[i]
	}
	atr /= float64(period)
	for i := period; i < len(trs); i++ {
		atr = (atr*float64(period-1) + trs[i]) / float64(period)
	}
	return atr
}

// fADXWilder — full Wilder ADX with +DI/-DI, identical to pkg/math.ADXWilder
// (including the zero-high/low skip inside the loop).
func fADXWilder(highs, lows, closes []float64, period int) (adx, plusDI, minusDI float64) {
	n := len(highs)
	if len(lows) < n {
		n = len(lows)
	}
	if len(closes) < n {
		n = len(closes)
	}
	if n <= period*2 || period <= 0 {
		return 0, 0, 0
	}
	plusDMs := make([]float64, n-1)
	minusDMs := make([]float64, n-1)
	trs := make([]float64, n-1)
	for i := 1; i < n; i++ {
		h, l, ph, pl, pc := highs[i], lows[i], highs[i-1], lows[i-1], closes[i-1]
		if h <= 0 || l <= 0 || ph <= 0 || pl <= 0 {
			continue
		}
		upMove := h - ph
		downMove := pl - l
		var plusDM, minusDM float64
		if upMove > downMove && upMove > 0 {
			plusDM = upMove
		}
		if downMove > upMove && downMove > 0 {
			minusDM = downMove
		}
		hl := math.Abs(h - l)
		hc := math.Abs(h - pc)
		lc := math.Abs(l - pc)
		tr := hl
		if hc > tr {
			tr = hc
		}
		if lc > tr {
			tr = lc
		}
		plusDMs[i-1] = plusDM
		minusDMs[i-1] = minusDM
		trs[i-1] = tr
	}
	avgTR, avgPlusDM, avgMinusDM := 0.0, 0.0, 0.0
	for i := 0; i < period; i++ {
		avgTR += trs[i]
		avgPlusDM += plusDMs[i]
		avgMinusDM += minusDMs[i]
	}
	avgTR /= float64(period)
	avgPlusDM /= float64(period)
	avgMinusDM /= float64(period)
	var adxVals []float64
	for i := period; i < len(trs); i++ {
		avgTR = (avgTR*float64(period-1) + trs[i]) / float64(period)
		avgPlusDM = (avgPlusDM*float64(period-1) + plusDMs[i]) / float64(period)
		avgMinusDM = (avgMinusDM*float64(period-1) + minusDMs[i]) / float64(period)
		var pdi, mdi float64
		if avgTR > 0 {
			pdi = 100.0 * avgPlusDM / avgTR
			mdi = 100.0 * avgMinusDM / avgTR
		}
		if pdi+mdi > 0 {
			adxVals = append(adxVals, 100.0*math.Abs(pdi-mdi)/(pdi+mdi))
		}
		if i == len(trs)-1 {
			plusDI, minusDI = pdi, mdi
		}
	}
	// Wilder smoothing over DX values, mirroring pkg/math.ADXWilder
	if len(adxVals) == 0 {
		return 0, plusDI, minusDI
	}
	adxSum := 0.0
	seed := len(adxVals)
	if seed > period {
		seed = period
	}
	for i := 0; i < seed; i++ {
		adxSum += adxVals[i]
	}
	adx = adxSum / float64(seed)
	for i := seed; i < len(adxVals); i++ {
		adx = (adx*float64(period-1) + adxVals[i]) / float64(period)
	}
	return adx, plusDI, minusDI
}
// === Decimal helpers retained for other engines (ichimoku, stochrsi, marnie) ===

func simpleMA(values []decimal.Decimal, period int) decimal.Decimal {
	if len(values) < period || period <= 0 {
		return decimal.Zero
	}
	sum := decimal.Zero
	for i := len(values) - period; i < len(values); i++ {
		sum = sum.Add(values[i])
	}
	return sum.Div(decimal.NewFromInt(int64(period)))
}

func maxSlice(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}
	m := values[0]
	for _, v := range values[1:] {
		if v.GreaterThan(m) {
			m = v
		}
	}
	return m
}

func minSlice(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}
	m := values[0]
	for _, v := range values[1:] {
		if v.LessThan(m) {
			m = v
		}
	}
	return m
}
