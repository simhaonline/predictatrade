package backtest

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// PersistResult stores the backtest result in trading.backtest_runs and trading.backtest_trades.
func PersistResult(ctx context.Context, conn *pgx.Conn, result *BacktestResult) error {
	// Insert into trading.backtest_runs
	config, _ := json.Marshal(result.Config)
	execAssumptions := map[string]interface{}{
		"spread":       result.Config.Spread.String(),
		"commission":   result.Config.Commission.String(),
		"contract_size": result.Config.ContractSize.String(),
		"slippage":     result.Config.Slippage.String(),
	}
	execJSON, _ := json.Marshal(execAssumptions)
	riskConfig := map[string]interface{}{
		"max_risk_per_trade": result.Config.MaxRiskPerTrade.String(),
		"max_positions":     result.Config.MaxPositions,
		"max_daily_loss_pct": result.Config.MaxDailyLossPct.String(),
		"trailing_stop":      result.Config.TrailingStopEnabled,
		"break_even":         result.Config.BreakEvenEnabled,
		"conservative_sl_tp":  result.Config.ConservativeSLTP,
	}
	riskJSON, _ := json.Marshal(riskConfig)

	_, err := conn.Exec(ctx, `
		INSERT INTO trading.backtest_runs
		  (run_id, symbol, strategy_id, strategy_mode, primary_timeframe,
		   start_timestamp, end_timestamp, initial_balance, random_seed,
		   status, bars_processed, trades_count, no_trade_count, blocked_count,
		   final_balance, total_return_pct, sharpe_ratio, sortino_ratio,
		   max_drawdown_pct, win_rate_pct, profit_factor, expectancy,
		   configuration, execution_assumptions, risk_config,
		   data_source, feature_version, completed_at, duration_seconds)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
		        $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29)
		ON CONFLICT (run_id) DO NOTHING
	`,
		result.RunID,
		result.Config.Symbol,
		string(result.Config.StrategyID),
		"go_replay",
		string(result.Config.PrimaryTimeframe),
		result.StartTime,
		result.EndTime,
		result.Config.InitialBalance.String(),
		42,
		result.Status,
		result.BarsProcessed,
		len(result.Trades),
		result.NoTradeCount,
		result.BlockedCount,
		result.Metrics.FinalBalance.String(),
		result.Metrics.TotalReturnPct.InexactFloat64(),
		result.Metrics.SharpeRatio.InexactFloat64(),
		result.Metrics.SortinoRatio.InexactFloat64(),
		result.Metrics.MaxDrawdownPct.InexactFloat64(),
		result.Metrics.WinRate.InexactFloat64(),
		result.Metrics.ProfitFactor.InexactFloat64(),
		result.Metrics.Expectancy.InexactFloat64(),
		config,
		execJSON,
		riskJSON,
		"market.candles",
		"go-production-v1.0",
		nil, // completed_at — use default
		result.Duration.Seconds(),
	)
	if err != nil {
		return fmt.Errorf("insert backtest_runs: %w", err)
	}

	// Insert trades
	for _, t := range result.Trades {
		_, err := conn.Exec(ctx, `
			INSERT INTO trading.backtest_trades
			  (run_id, trade_id, strategy_id, direction, entry_time, entry_price, exit_price,
			   exit_reason, size, pnl, pnl_r, regime, session,
			   duration_bars)
			VALUES ($1, $2, $3, $4, $5, $6, $7,
			        $8, $9, $10, $11, $12, $13,
			        $14)
		`,
			result.RunID,
			t.TradeID,
			string(t.StrategyID),
			string(t.Direction),
			t.EntryTime.UTC(),
			t.EntryPrice.String(),
			t.ExitPrice.String(),
			t.ExitReason,
			t.Size.String(),
			t.RealizedPnL.String(),
			t.RealizedR.InexactFloat64(),
			string(t.Regime),
			t.Session,
			t.HoldingBars,
		)
		if err != nil {
			// Log the error to diagnose column mismatches
			fmt.Printf("WARN: trade insert failed: %v\n", err)
			continue
		}
	}

	return nil
}
