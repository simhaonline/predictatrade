package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// TwelveDataTimeSeriesValue is a single OHLCV bar from the /time_series endpoint.
type TwelveDataTimeSeriesValue struct {
	Datetime string `json:"datetime"`
	Open     string `json:"open"`
	High     string `json:"high"`
	Low      string `json:"low"`
	Close    string `json:"close"`
	Volume   string `json:"volume"`
}

// TwelveDataTimeSeries is the response envelope for /time_series.
type TwelveDataTimeSeries struct {
	Meta   map[string]interface{}          `json:"meta"`
	Values []TwelveDataTimeSeriesValue     `json:"values"`
	Status string                          `json:"status"`
	Code   int                             `json:"code"`
	Message string                         `json:"message"`
}

// timeframeToTwelveDataInterval maps internal timeframes to Twelve Data intervals.
func timeframeToTwelveDataInterval(tf types.Timeframe) string {
	switch tf {
	case types.TFM1:
		return "1min"
	case types.TFM5:
		return "5min"
	case types.TFM15:
		return "15min"
	case types.TFM30:
		return "30min"
	case types.TFH1:
		return "1h"
	case types.TFH4:
		return "4h"
	case types.TFD1:
		return "1day"
	default:
		return "5min"
	}
}

// FetchTimeSeries fetches historical OHLCV bars from Twelve Data /time_series
// and returns them as internal candles (ascending by time). Values arrive
// newest-first from the API, so they are reversed here.
//
// start/end use Twelve Data date format (e.g. "2024-01-01"). The returned
// candles carry Source="TWELVEDATA", Quality=COMPLETE and IsClosed=true so they
// are durable historical bars (never overwritten by live partial candles).
func (p *TwelveDataProvider) FetchTimeSeries(ctx context.Context, providerSymbol string, tf types.Timeframe, start, end string) ([]*types.Candle, error) {
	if !p.IsConfigured() {
		return nil, fmt.Errorf("TwelveData provider not configured (no API key)")
	}

	interval := timeframeToTwelveDataInterval(tf)
	url := fmt.Sprintf("%s/time_series?symbol=%s&interval=%s&start_date=%s&end_date=%s&apikey=%s&format=JSON",
		p.apiBase, sanitizeSymbolForURL(providerSymbol), interval, start, end, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var ts TwelveDataTimeSeries
	if err := json.NewDecoder(resp.Body).Decode(&ts); err != nil {
		return nil, err
	}
	if ts.Status != "ok" || len(ts.Values) == 0 {
		msg := ts.Message
		if msg == "" {
			msg = "empty time_series response"
		}
		return nil, fmt.Errorf("twelvedata time_series error: %s (code %d)", msg, ts.Code)
	}

	// Reverse newest-first -> oldest-first.
	candles := make([]*types.Candle, 0, len(ts.Values))
	for i := len(ts.Values) - 1; i >= 0; i-- {
		v := ts.Values[i]
		t, err := time.Parse("2006-01-02 15:04:05", v.Datetime)
		if err != nil {
			t, err = time.Parse("2006-01-02", v.Datetime)
			if err != nil {
				continue
			}
		}
		t = t.UTC()

		o, _ := decimal.NewFromString(v.Open)
		h, _ := decimal.NewFromString(v.High)
		l, _ := decimal.NewFromString(v.Low)
		c, _ := decimal.NewFromString(v.Close)
		vol, _ := strconv.ParseInt(firstNonEmpty(v.Volume, "0"), 10, 64)

		candles = append(candles, &types.Candle{
			Symbol:    providerToCanonical(providerSymbol),
			Timeframe: tf,
			Time:      t,
			Open:      o,
			High:      h,
			Low:       l,
			Close:     c,
			Volume:    vol,
			Source:    "TWELVEDATA",
			Quality:   types.CandleComplete,
			IsClosed:  true,
		})
	}
	return candles, nil
}

func firstNonEmpty(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}

// providerToCanonical maps a Twelve Data provider symbol back to a canonical
// symbol for storage. XAU/USD -> XAUUSD, BTC/USD -> BTCUSD, else the raw symbol.
func providerToCanonical(providerSymbol string) string {
	switch providerSymbol {
	case "XAU/USD":
		return "XAUUSD"
	case "BTC/USD":
		return "BTCUSD"
	case "EUR/USD":
		return "EURUSD"
	default:
		return strings.ReplaceAll(providerSymbol, "/", "")
	}
}

// ResolveProviderSymbol maps a canonical symbol to its Twelve Data provider symbol.
func (p *TwelveDataProvider) ResolveProviderSymbol(canonical string) string {
	if ps, ok := p.symbols[canonical]; ok {
		return ps
	}
	switch canonical {
	case "XAUUSD":
		return "XAU/USD"
	default:
		return strings.ReplaceAll(canonical, "/", "") // already-provider form
	}
}
