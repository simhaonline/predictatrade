// Package marketdata abstracts the source of MarketState snapshots so the engine
// has no hard dependency on any external service. v1 ships a CSV replay provider
// (real-data backtests / shadow runs) plus a clear interface for a future live
// Windows-Agent WS provider.
package marketdata

import (
	"encoding/csv"
	"os"
	"strconv"

	"pat-engine/internal/types"
)

// Provider yields market snapshots one at a time.
type Provider interface {
	Next() (*types.MarketState, bool, error)
	Close() error
}

// CSVReplay reads precomputed MarketState snapshots from a CSV file. Each row is
// one bar's full indicator set, so the engine can validate strategy + policy +
// gates deterministically without a broker connection.
type CSVReplay struct {
	rows [][]string
	idx  int
}

func NewCSVReplay(path string) (*CSVReplay, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	recs, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	// Skip a leading header row if present.
	start := 0
	if len(recs) > 0 && len(recs[0]) > 0 && recs[0][0] == "symbol" {
		start = 1
	}
	return &CSVReplay{rows: recs[start:]}, nil
}

func (c *CSVReplay) Close() error { return nil }

func (c *CSVReplay) Next() (*types.MarketState, bool, error) {
	if c.idx >= len(c.rows) {
		return nil, false, nil
	}
	row := c.rows[c.idx]
	c.idx++
	return parseRow(row), true, nil
}

func f(row []string, i int) float64 {
	if i >= len(row) {
		return 0
	}
	v, _ := strconv.ParseFloat(row[i], 64)
	return v
}

func b(row []string, i int) bool {
	if i >= len(row) {
		return false
	}
	return row[i] == "1" || row[i] == "true" || row[i] == "TRUE"
}

// parseRow maps a fixed-column CSV row to a MarketState. Column order is:
// symbol,tf,price,h1close,spread,atr,ema9,ema21,ema50,adx,adx+,adx-,rsi,
// macdmain,macdsig,osma,stochmain,stochsig,bollu,bolll,vwap,mtfscore,regime,
// session,isoverlap,newsrisk,quality,candle_disp,candle_bull,candle_bear,
// candle_rej,sweep
func parseRow(row []string) *types.MarketState {
	st := &types.MarketState{
		Symbol:       row[0],
		Timeframe:    types.Timeframe(row[1]),
		CurrentPrice: f(row, 2),
		H1Close:      f(row, 3),
		Spread:       f(row, 4),
		ATR:          f(row, 5),
		Indicators: types.Indicators{
			EMA9: f(row, 6), EMA21: f(row, 7), EMA50: f(row, 8),
			ADX: f(row, 9), ADXPlusDI: f(row, 10), ADXMinusDI: f(row, 11),
			RSI: f(row, 12),
			MACDMain: f(row, 13), MACDSignal: f(row, 14),
			OsMA: f(row, 15),
			StochMain: f(row, 16), StochSignal: f(row, 17),
			BollUpper: f(row, 18), BollLower: f(row, 19),
		},
		MTFScore: f(row, 21),
		Regime:   row[22],
		Session: types.Session{
			CurrentSession: row[23],
			IsOverlap:      b(row, 24),
			NewsRisk:       row[25],
		},
		Quality: row[26],
		VWAP:    f(row, 20),
	}
	st.Candle = types.Candle{
		IsDisplacement: b(row, 27),
		IsBullish:      b(row, 28),
		IsBearish:      b(row, 29),
		IsRejection:    b(row, 30),
	}
	if len(row) > 31 {
		dir := row[31]
		if dir == "BUY_SIDE" || dir == "SELL_SIDE" {
			st.Liquidity.RecentSweeps = []types.Sweep{{Direction: dir}}
		}
	}
	// Optional higher-timeframe / swing indicators (zero when absent).
	st.Indicators.SMA200 = f(row, 32)
	st.Indicators.EMA100 = f(row, 33)
	st.Indicators.EMA200 = f(row, 34)
	st.Indicators.CCI = f(row, 35)
	st.Indicators.ParabolicSAR = f(row, 36)
	st.Indicators.ParabolicSARLong = b(row, 37)
	if len(row) > 38 && row[38] != "" {
		st.Structure.LastBOS = &types.BOS{Direction: row[38]}
	}
	return st
}
