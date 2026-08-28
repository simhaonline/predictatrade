// Package indicators provides the minimal technical-indicator math the strategies
// consume. It is pure Go (no deps) and used by the backtest to turn raw bars into
// the MarketState snapshots the strategies evaluate — so the backtest runs on the
// exact same strategy code and config as live.
package indicators

// EMA returns the exponential moving average, aligned to the input length. The
// first value is seeded with the simple average of the first `period` samples.
func EMA(in []float64, period int) []float64 {
	out := make([]float64, len(in))
	if len(in) == 0 || period <= 0 {
		return out
	}
	k := 2.0 / float64(period+1)
	seed := 0.0
	for i := 0; i < period && i < len(in); i++ {
		seed += in[i]
	}
	seed /= float64(period)
	out[period-1] = seed
	prev := seed
	for i := period; i < len(in); i++ {
		prev = in[i]*k + prev*(1-k)
		out[i] = prev
	}
	return out
}

// SMA returns the simple moving average aligned to the input length (0 before seed).
func SMA(in []float64, period int) []float64 {
	out := make([]float64, len(in))
	if period <= 0 {
		return out
	}
	sum := 0.0
	for i := 0; i < len(in); i++ {
		sum += in[i]
		if i >= period {
			sum -= in[i-period]
		}
		if i >= period-1 {
			out[i] = sum / float64(period)
		}
	}
	return out
}

// ATR computes the Wilder-smoothed Average True Range.
func ATR(high, low, close []float64, period int) []float64 {
	n := len(close)
	tr := make([]float64, n)
	for i := 0; i < n; i++ {
		if i == 0 {
			tr[i] = high[i] - low[i]
			continue
		}
		tr[i] = max3(high[i]-low[i], abs(high[i]-close[i-1]), abs(low[i]-close[i-1]))
	}
	return wilder(tr, period)
}

// RSI computes the Wilder Relative Strength Index (0-100).
func RSI(close []float64, period int) []float64 {
	n := len(close)
	out := make([]float64, n)
	if n < 2 {
		return out
	}
	gain, loss := 0.0, 0.0
	for i := 1; i <= period && i < n; i++ {
		ch := close[i] - close[i-1]
		if ch >= 0 {
			gain += ch
		} else {
			loss -= ch
		}
	}
	ag := gain / float64(period)
	al := loss / float64(period)
	for i := period + 1; i < n; i++ {
		ch := close[i] - close[i-1]
		g, l := 0.0, 0.0
		if ch >= 0 {
			g = ch
		} else {
			l = -ch
		}
		ag = (ag*float64(period-1) + g) / float64(period)
		al = (al*float64(period-1) + l) / float64(period)
	}
	for i := period; i < n; i++ {
		if i == period {
			// seed handled below
		}
		// recompute RS from current ag/al (approximate steady-state)
		rs := 0.0
		if al != 0 {
			rs = ag / al
		}
		out[i] = 100 - 100/(1+rs)
	}
	// seed the first RSI value using the initial averages
	rs0 := 0.0
	if (loss/float64(period)) != 0 {
		rs0 = (gain / float64(period)) / (loss / float64(period))
	}
	out[period] = 100 - 100/(1+rs0)
	return out
}

// MACD returns macd line, signal line, and histogram (osma).
func MACD(close []float64, fast, slow, signal int) (macd, sig, hist []float64) {
	ef := EMA(close, fast)
	es := EMA(close, slow)
	n := len(close)
	macd = make([]float64, n)
	for i := 0; i < n; i++ {
		macd[i] = ef[i] - es[i]
	}
	sig = EMA(macd, signal)
	hist = make([]float64, n)
	for i := 0; i < n; i++ {
		hist[i] = macd[i] - sig[i]
	}
	return
}

// ADX computes ADX plus +DI/-DI using Wilder smoothing.
func ADX(high, low, close []float64, period int) (adx, plusDI, minusDI []float64) {
	n := len(close)
	adx = make([]float64, n)
	plusDI = make([]float64, n)
	minusDI = make([]float64, n)
	if n < 2 {
		return
	}
	tr := make([]float64, n)
	ppDM := make([]float64, n)
	pmDM := make([]float64, n)
	for i := 1; i < n; i++ {
		tr[i] = max3(high[i]-low[i], abs(high[i]-close[i-1]), abs(low[i]-close[i-1]))
		up := high[i] - high[i-1]
		dn := low[i-1] - low[i]
		if up > dn && up > 0 {
			ppDM[i] = up
		}
		if dn > up && dn > 0 {
			pmDM[i] = dn
		}
	}
	atr := wilder(tr, period)
	pd := wilder(ppDM, period)
	md := wilder(pmDM, period)
	dx := make([]float64, n)
	for i := period; i < n; i++ {
		if atr[i] != 0 {
			plusDI[i] = 100 * pd[i] / atr[i]
			minusDI[i] = 100 * md[i] / atr[i]
		}
		denom := plusDI[i] + minusDI[i]
		if denom != 0 {
			dx[i] = 100 * abs(plusDI[i]-minusDI[i]) / denom
		}
	}
	adxSmooth := wilder(dx, period)
	copy(adx, adxSmooth)
	return
}

// Stochastic returns %K and %D (smoothed) over the given lookback/shift.
func Stochastic(high, low, close []float64, lookback, shift int) (k, d []float64) {
	n := len(close)
	k = make([]float64, n)
	d = make([]float64, n)
	for i := lookback - 1; i < n; i++ {
		hh := high[i]
		ll := low[i]
		for j := i - lookback + 1; j <= i; j++ {
			if high[j] > hh {
				hh = high[j]
			}
			if low[j] < ll {
				ll = low[j]
			}
		}
		if hh != ll {
			k[i] = 100 * (close[i] - ll) / (hh - ll)
		}
	}
	ds := SMA(k, shift)
	copy(d, ds)
	return
}

// Bollinger returns upper/lower bands (middle = SMA20).
func Bollinger(close []float64, period int, mult float64) (mid, upper, lower []float64) {
	n := len(close)
	mid = SMA(close, period)
	upper = make([]float64, n)
	lower = make([]float64, n)
	for i := period - 1; i < n; i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += (close[j] - mid[i]) * (close[j] - mid[i])
		}
		sd := sqrt(sum / float64(period))
		upper[i] = mid[i] + mult*sd
		lower[i] = mid[i] - mult*sd
	}
	return
}

// VWAP approximates a session VWAP using typical price (volume-free proxy).
func VWAP(high, low, close []float64, period int) []float64 {
	n := len(close)
	tp := make([]float64, n)
	for i := 0; i < n; i++ {
		tp[i] = (high[i] + low[i] + close[i]) / 3
	}
	return SMA(tp, period)
}

func wilder(in []float64, period int) []float64 {
	n := len(in)
	out := make([]float64, n)
	if n < period {
		return out
	}
	sum := 0.0
	for i := 1; i <= period && i < n; i++ {
		sum += in[i]
	}
	prev := sum / float64(period)
	out[period] = prev
	for i := period + 1; i < n; i++ {
		prev = (prev*float64(period-1) + in[i]) / float64(period)
		out[i] = prev
	}
	return out
}

func max3(a, b, c float64) float64 {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	g := x
	for i := 0; i < 20; i++ {
		g = (g + x/g) / 2
	}
	return g
}
