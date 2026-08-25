// Command backfill captures historical XAUUSD (and macro) bars from the Twelve Data
// API and stores them in market.candles (TimescaleDB) so users can run online
// backtests on durable, provenance-tagged data.
//
// Usage:
//   backfill --symbol XAUUSD --timeframe M5 --start 2024-01-01 --end 2024-03-01
//   backfill --symbol XAUUSD --timeframe M1 --start 2024-01-01 --end 2024-01-31
//
// Env: TWELVEDATA_API_KEY (required), DATABASE_URL (defaults to local dev DSN).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/predictatrade/realtime/internal/marketdata"
	"github.com/predictatrade/realtime/internal/types"
)

func main() {
	var (
		symbol    = flag.String("symbol", "XAUUSD", "canonical symbol to backfill (XAUUSD, BTCUSD, EURUSD, WTI, VIX)")
		timeframe = flag.String("timeframe", "M5", "candle timeframe (M1,M5,M15,M30,H1,H4,D1)")
		start     = flag.String("start", "", "start date YYYY-MM-DD (required)")
		end       = flag.String("end", "", "end date YYYY-MM-DD (required)")
		apiKey    = flag.String("api-key", os.Getenv("TWELVEDATA_API_KEY"), "Twelve Data API key")
		dbURL     = flag.String("db-url", envOr("DATABASE_URL", "postgresql://pat_admin:pat_local_dev_only@127.0.0.1:5432/predictatrade?sslmode=disable"), "PostgreSQL/TimescaleDB DSN")
	)
	flag.Parse()

	if *start == "" || *end == "" {
		log.Fatal("--start and --end are required (YYYY-MM-DD)")
	}
	if *apiKey == "" {
		log.Fatal("TWELVEDATA_API_KEY is not set")
	}

	tf := types.Timeframe(*timeframe)
	if tf == "" {
		log.Fatalf("invalid timeframe %q", *timeframe)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	provider := marketdata.NewTwelveDataProvider(*apiKey)
	providerSymbol := provider.ResolveProviderSymbol(*symbol)

	persister, err := marketdata.NewPersister(*dbURL)
	if err != nil {
		log.Fatalf("connect DB: %v", err)
	}
	defer persister.Close()

	log.Printf("[BACKFILL] fetching %s (%s) %s from %s to %s", *symbol, providerSymbol, *timeframe, *start, *end)
	candles, err := provider.FetchTimeSeries(ctx, providerSymbol, tf, *start, *end)
	if err != nil {
		log.Fatalf("fetch time_series: %v", err)
	}
	if len(candles) == 0 {
		log.Fatal("no candles returned (check symbol/interval/date range and API quota)")
	}

	saved := 0
	for _, c := range candles {
		if err := persister.SaveCandle(ctx, c); err != nil {
			log.Printf("[BACKFILL] save error: %v", err)
			continue
		}
		saved++
	}

	fmt.Printf("OK backfilled %d/%d candles for %s %s into market.candles (source=TWELVEDATA)\n",
		saved, len(candles), *symbol, *timeframe)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
