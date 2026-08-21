// Package cache — Valkey caching for candle data.
//
// Caches bootstrap and chart candles in Valkey to avoid repeated PostgreSQL
// queries. Uses a 60-second TTL for chart data and a 5-minute TTL for
// bootstrap data (historical candles don't change frequently).
package cache

import (
	"encoding/json"
	"fmt"
	"time"
)

// CachedCandle is a JSON-serializable candle for Valkey caching.
type CachedCandle struct {
	Time      string  `json:"time"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    int64   `json:"volume"`
	Source    string  `json:"source"`
	Quality   string  `json:"quality"`
	IsClosed  bool    `json:"is_closed"`
}

// SetBootstrapCandles caches historical candles for engine startup bootstrap.
// TTL: 5 minutes (historical data doesn't change frequently).
func (v *ValkeyCache) SetBootstrapCandles(symbol, timeframe string, candles []CachedCandle) error {
	key := fmt.Sprintf("pat:bootstrap_candles:%s:%s", symbol, timeframe)
	b, err := json.Marshal(candles)
	if err != nil {
		return err
	}
	return v.client.Set(v.ctx, key, b, 5*time.Minute).Err()
}

// GetBootstrapCandles retrieves cached bootstrap candles from Valkey.
// Returns nil, nil if not cached (caller should fall back to PostgreSQL).
func (v *ValkeyCache) GetBootstrapCandles(symbol, timeframe string) ([]CachedCandle, error) {
	key := fmt.Sprintf("pat:bootstrap_candles:%s:%s", symbol, timeframe)
	b, err := v.client.Get(v.ctx, key).Bytes()
	if err != nil {
		return nil, err // cache miss
	}
	var candles []CachedCandle
	if err := json.Unmarshal(b, &candles); err != nil {
		return nil, err
	}
	return candles, nil
}

// SetChartCandles caches candles for chart data endpoint.
// TTL: 60 seconds (chart data refreshes frequently but doesn't need sub-second freshness).
func (v *ValkeyCache) SetChartCandles(symbol, timeframe string, limit int, candles []CachedCandle) error {
	key := fmt.Sprintf("pat:chart_candles:%s:%s:%d", symbol, timeframe, limit)
	b, err := json.Marshal(candles)
	if err != nil {
		return err
	}
	return v.client.Set(v.ctx, key, b, 60*time.Second).Err()
}

// GetChartCandles retrieves cached chart candles from Valkey.
func (v *ValkeyCache) GetChartCandles(symbol, timeframe string, limit int) ([]CachedCandle, error) {
	key := fmt.Sprintf("pat:chart_candles:%s:%s:%d", symbol, timeframe, limit)
	b, err := v.client.Get(v.ctx, key).Bytes()
	if err != nil {
		return nil, err // cache miss
	}
	var candles []CachedCandle
	if err := json.Unmarshal(b, &candles); err != nil {
		return nil, err
	}
	return candles, nil
}
