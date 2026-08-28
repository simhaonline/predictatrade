package backtest

import (
	"encoding/csv"
	"os"
	"strconv"
	"strings"
	"time"

	"pat-engine/internal/indicators"
	"pat-engine/internal/types"
)

// StateFromBars builds a single MarketState for the latest bar using the same
// indicator math as BuildSnapshots. It is used by the live gateway, which keeps a
// rolling window of bars and asks for the current snapshot.
func StateFromBars(bars []Bar) *types.MarketState {
	n := len(bars)
	if n == 0 {
		return nil
	}
	if n < 200 {
		// not enough history for stable indicators
		return nil
	}
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

	i := n - 1
	regime := "RANGE"
	switch {
	case adx[i] > 25 && ema9[i] > ema21[i]:
		regime = "TRENDING_BULLISH"
	case adx[i] > 25 && ema9[i] < ema21[i]:
		regime = "TRENDING_BEARISH"
	case adx[i] > 30:
		regime = "HIGH_VOLATILITY"
	}
	mtf := (ema9[i] - ema21[i]) / maxf(atr[i], 0.0001) * 10

	return &types.MarketState{
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
		Session:  types.Session{CurrentSession: "LONDON"},
		Quality:  "AUTHORITATIVE",
		VWAP:     vwap[i],
	}
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// FromCSV loads raw bars from a CSV with header:
// time,open,high,low,close,spread (spread optional, defaults to 0.20).
// This is the real-data entry point for the backtest harness.
func FromCSV(path string) ([]Bar, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	var bars []Bar
	for idx, row := range rows {
		if idx == 0 {
			continue // header
		}
		if len(row) < 5 {
			continue
		}
		o, _ := strconv.ParseFloat(strings.TrimSpace(row[1]), 64)
		h, _ := strconv.ParseFloat(strings.TrimSpace(row[2]), 64)
		l, _ := strconv.ParseFloat(strings.TrimSpace(row[3]), 64)
		c, _ := strconv.ParseFloat(strings.TrimSpace(row[4]), 64)
		sp := 0.20
		if len(row) >= 6 {
			if v, e := strconv.ParseFloat(strings.TrimSpace(row[5]), 64); e == nil {
				sp = v
			}
		}
		var t int64
		if len(row) >= 1 {
			if v, e := strconv.ParseInt(strings.TrimSpace(row[0]), 10, 64); e == nil {
				t = v
			}
		}
		bars = append(bars, Bar{Time: t, Open: o, High: h, Low: l, Close: c, Spread: sp})
	}
	return bars, nil
}

// FromMetaCSV loads MetaTrader-style bars: "Date;Open;High;Low;Close;Volume"
// (semicolon-delimited, date like "2004.06.11 07:00"). No spread column exists, so
// it defaults to 0.20 (typical XAUUSD). This is the real-history entry point.
func FromMetaCSV(path string) ([]Bar, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = ';'
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	const layout = "2006.01.02 15:04"
	var bars []Bar
	for idx, row := range rows {
		if idx == 0 {
			continue // header
		}
		if len(row) < 5 {
			continue
		}
		o, _ := strconv.ParseFloat(strings.TrimSpace(row[1]), 64)
		h, _ := strconv.ParseFloat(strings.TrimSpace(row[2]), 64)
		l, _ := strconv.ParseFloat(strings.TrimSpace(row[3]), 64)
		c, _ := strconv.ParseFloat(strings.TrimSpace(row[4]), 64)
		t := int64(0)
		if tm, e := time.Parse(layout, strings.TrimSpace(row[0])); e == nil {
			t = tm.Unix()
		}
		bars = append(bars, Bar{Time: t, Open: o, High: h, Low: l, Close: c, Spread: 0.20})
	}
	return bars, nil
}
