// Command backtest-engine runs historical strategy backtests using the
// EXACT production Go feature engine and strategy evaluators.
//
// Usage:
//   backtest-engine --strategy STANDARD_SCALPING --timeframe M5 --start 2025-06-01 --end 2025-06-30
//   backtest-engine --strategy TREND_SWING --timeframe M5 --start 2025-01-01 --end 2025-12-31 --store
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/predictatrade/realtime/internal/backtest"
	"github.com/shopspring/decimal"
	"github.com/predictatrade/realtime/internal/types"
)

func main() {
	strategyID := flag.String("strategy", "STANDARD_SCALPING", "Strategy: STANDARD_SCALPING, ULTRA_SCALPING, STANDARD_SWING, TREND_SWING, MARNIE_FIB")
	timeframe := flag.String("timeframe", "M5", "Primary timeframe")
	startStr := flag.String("start", "", "Start date YYYY-MM-DD")
	endStr := flag.String("end", "", "End date YYYY-MM-DD")
	balance := flag.Float64("balance", 10000, "Initial balance")
	store := flag.Bool("store", false, "Store results in database (trading.backtest_runs)")
	dbURL := flag.String("db-url", "", "PostgreSQL URL (or read from database_url.txt)")
	higherTFs := flag.String("higher-tfs", "M15,H1,H4,D1", "Higher timeframes for MTF alignment")
	source := flag.String("source", "MT5_MASTER", "market.candles.source to use (real feed). Empty = all sources.")
	flag.Parse()

	// Get DB URL
	url := *dbURL
	if url == "" {
		urlFile := "/srv/predictatrade/xauusd/database_url.txt"
		data, err := os.ReadFile(urlFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: no --db-url and cannot read %s: %v\n", urlFile, err)
			os.Exit(1)
		}
		url = strings.TrimSpace(string(data))
	}
	// Fall back to the DATABASE_URL environment variable (M8 fix): lets the
	// control plane pass the connection string via the child process env
	// instead of as a CLI argument (which would expose the password in `ps`).
	if url == "" {
		url = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}

	// Parse dates
	if *startStr == "" || *endStr == "" {
		fmt.Fprintf(os.Stderr, "ERROR: --start and --end are required (YYYY-MM-DD)\n")
		os.Exit(1)
	}
	startTime, err := time.Parse("2006-01-02", *startStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: invalid --start: %v\n", err)
		os.Exit(1)
	}
	endTime, err := time.Parse("2006-01-02", *endStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: invalid --end: %v\n", err)
		os.Exit(1)
	}

	// Build higher TF list
	var htfs []types.Timeframe
	for _, tf := range strings.Split(*higherTFs, ",") {
		htfs = append(htfs, types.Timeframe(strings.TrimSpace(tf)))
	}

	// Build config
	config := backtest.DefaultConfig()
	config.StrategyID = types.StrategyID(*strategyID)
	config.PrimaryTimeframe = types.Timeframe(*timeframe)
	config.HigherTimeframes = htfs
	config.StartTime = startTime.UTC()
	config.EndTime = endTime.UTC()
	config.InitialBalance = decimal.NewFromFloat(*balance)
	config.DBUrl = url
	config.Source = *source

	// Print config
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("Predict-A-Trade Go Backtest Engine")
	fmt.Println("  Uses EXACT production feature engine + strategy evaluators")
	fmt.Printf("  Strategy:    %s\n", config.StrategyID)
	fmt.Printf("  Timeframe:   %s\n", config.PrimaryTimeframe)
	fmt.Printf("  Higher TFs:  %v\n", config.HigherTimeframes)
	fmt.Printf("  Period:      %s → %s\n", startTime.Format("2006-01-02"), endTime.Format("2006-01-02"))
	fmt.Printf("  Source:      %s\n", *source)
	fmt.Printf("  Balance:     $%.2f\n", *balance)
	fmt.Printf("  Store in DB: %v\n", *store)
	fmt.Println(strings.Repeat("=", 70))

	// Run backtest
	ctx := context.Background()
	runner := backtest.NewRunner(config)
	result, err := runner.Run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: backtest failed: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Backtest Run: %s\n", result.RunID)
	fmt.Printf("Status:       %s\n", result.Status)
	if result.Error != "" {
		fmt.Printf("Error:        %s\n", result.Error)
	}
	fmt.Printf("Period:       %s → %s\n", result.StartTime.Format("2006-01-02 15:04"), result.EndTime.Format("2006-01-02 15:04"))
	fmt.Printf("Bars:         %d\n", result.BarsProcessed)
	fmt.Printf("Duration:     %.2fs\n", result.Duration.Seconds())
	fmt.Println()
	fmt.Println("--- Signal Decisions ---")
	fmt.Printf("  BUY signals:     %d\n", result.BuySignals)
	fmt.Printf("  SELL signals:    %d\n", result.SellSignals)
	fmt.Printf("  NO_TRADE:        %d\n", result.NoTradeCount)
	fmt.Printf("  Blocked:         %d\n", result.BlockedCount)
	fmt.Printf("  Total trades:    %d\n", len(result.Trades))
	fmt.Println()
	m := result.Metrics
	fmt.Println("--- Performance Metrics ---")
	fmt.Printf("  Initial balance:  $%.2f\n", *balance)
	fmt.Printf("  Final balance:    $%.2f\n", m.FinalBalance.InexactFloat64())
	fmt.Printf("  Net profit:      $%.2f\n", m.NetProfit.InexactFloat64())
	fmt.Printf("  Total return:    %.2f%%\n", m.TotalReturnPct.InexactFloat64())
	fmt.Printf("  Win rate:        %.1f%%\n", m.WinRate.InexactFloat64())
	fmt.Printf("  Profit factor:   %.2f\n", m.ProfitFactor.InexactFloat64())
	fmt.Printf("  Sharpe ratio:     %.2f\n", m.SharpeRatio.InexactFloat64())
	fmt.Printf("  Sortino ratio:   %.2f\n", m.SortinoRatio.InexactFloat64())
	fmt.Printf("  Max drawdown:    %.2f%%\n", m.MaxDrawdownPct.InexactFloat64())
	fmt.Printf("  Total trades:    %d\n", m.TotalTrades)
	fmt.Printf("  Wins/Losses:     %d/%d\n", m.Wins, m.Losses)
	fmt.Printf("  Avg win:         $%.2f\n", m.AvgWin.InexactFloat64())
	fmt.Printf("  Avg loss:        $%.2f\n", m.AvgLoss.InexactFloat64())
	fmt.Printf("  Expectancy:      $%.2f\n", m.Expectancy.InexactFloat64())
	fmt.Printf("  Best trade:      $%.2f\n", m.BestTrade.InexactFloat64())
	fmt.Printf("  Worst trade:     $%.2f\n", m.WorstTrade.InexactFloat64())
	fmt.Printf("  Avg hold bars:   %d\n", m.AvgHoldingBars)
	fmt.Printf("  Max consec wins: %d\n", m.MaxConsecutiveWins)
	fmt.Printf("  Max consec loss: %d\n", m.MaxConsecutiveLosses)

	// Show first few trades
	if len(result.Trades) > 0 {
		fmt.Println()
		fmt.Println("--- First 10 Trades ---")
		showCount := 10
		if len(result.Trades) < showCount {
			showCount = len(result.Trades)
		}
		for i := 0; i < showCount; i++ {
			t := result.Trades[i]
			fmt.Printf("  %s %s @ %.2f → %.2f | SL=%.2f | %s | P&L=$%.2f | R=%.2f | %s\n",
				t.Direction, t.StrategyID, t.EntryPrice.InexactFloat64(), t.ExitPrice.InexactFloat64(),
				t.StopLoss.InexactFloat64(), t.ExitReason,
				t.RealizedPnL.InexactFloat64(), t.RealizedR.InexactFloat64(),
				t.EntryTime.Format("2006-01-02 15:04"))
		}
	}

	// Store in database
	if *store {
		fmt.Println()
		fmt.Println("--- Storing results in database ---")
		conn, err := pgx.Connect(ctx, url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: cannot connect to DB: %v\n", err)
			os.Exit(1)
		}
		defer conn.Close(ctx)

		if err := backtest.PersistResult(ctx, conn, result); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: persist failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  [OK] Stored as run_id=%s in trading.backtest_runs\n", result.RunID)
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("Backtest complete.")
}

